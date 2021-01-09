package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func main() {
	if err := run(); nil != err {
		log.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func run() error {
	client := &Client{}
	if err := client.Initialize(); nil != err {
		return err
	}

	// Test our access token, if 401, login
	_, err := client.Matrix.Whoami()
	if err != nil {
		e, ok := err.(mautrix.HTTPError)
		if !ok {
			return errors.Wrap(err, "unable to test credentials")
		}
		if e.Response.StatusCode == 401 {
			if err := client.Login(); nil != err {
				return errors.Wrap(err, "unable to login")
			}
		} else {
			return errors.Wrap(err, "unable to test credentials")
		}
	}

	// Change the filesystem root to the user account
	client.AccountRoot = path.Join(
		client.Config.Directory,
		client.Matrix.HomeserverURL.Hostname(),
		string(client.Matrix.UserID),
	)

	go createRooms(client.Matrix, client.AccountRoot)

	// Create our syncer. What will happen when we receive an event. */
	client.Matrix.Syncer.(*mautrix.DefaultSyncer).OnEventType(
		event.EventMessage, func(source mautrix.EventSource, ev *event.Event) {
			handleMessage(ev, client.AccountRoot)
		},
	)

	if err := client.Matrix.Sync(); nil != err {
		return errors.Wrap(err, "sync loop exited")
	}

	return nil
}

// This handles only MessageEventType events
func handleMessage(ev *event.Event, root string) {
	/* Parse the body of the message. */
	msg, ok := ev.Content.Parsed.(*event.MessageEventContent)
	if !ok {
		log.Printf("Received a normal message of type %T\n", ev.Content.Parsed)
		return
	}

	file := path.Join(root,
		ev.RoomID.String(),
		ev.Sender.String(),
		ev.ID.String(),
	)
	writeString(msg.Body, file)

	/* Set the creation time of the file to the timestamp of the server. */
	t := time.Unix(int64(ev.Timestamp/1000), 0)
	if err := os.Chtimes(file, time.Now(), t); nil != err {
		log.Println("Failed to set timestamp on message:", err)
	}

	/* Print the file path to stdout for clients. */
	fmt.Println(file)
}

func rooms(cli *mautrix.Client) []id.RoomID {
	rooms, err := cli.JoinedRooms()
	if nil != err {
		log.Println("Unable to get list of joined rooms:", err)
		return []id.RoomID{}
	}
	return rooms.JoinedRooms
}

func forEachMember(cli *mautrix.Client, room id.RoomID, forMember func(id id.UserID, avatar, name string)) {
	members, err := cli.JoinedMembers(room)
	if nil != err {
		log.Println("Unable to get members of room:", room, ":", err)
		return
	}
	for id, member := range members.Joined {
		var avatar, name string
		if nil != member.AvatarURL {
			avatar = *member.AvatarURL
		}
		if nil != member.DisplayName {
			name = *member.DisplayName
		}
		forMember(id, avatar, name)
	}
}

/* Get a list of joined rooms and set up a pipe for reading messages from. */
func createRooms(cli *mautrix.Client, root string) {
	for _, room := range rooms(cli) {
		/* Read the input pipe and send messages. */
		go readMessagePipe(cli, path.Join(root, room.String(), "in"), room.String())

		/* For each joined member of the room. */
		forEachMember(cli, room, func(id id.UserID, avatar, name string) {
			memberPath := path.Join(root, room.String(), id.String())

			/* If this user is yourself. */
			if cli.UserID == id {
				go readTypingPipe(cli, path.Join(memberPath, "typing"), room.String())
			}

			/* Write the members display name and avatar to file. */
			writeString(name, root, room.String(), id.String(), "name")
			writeString(avatar, root, room.String(), id.String(), "avatar")
		})
	}
}

/* Ensure the containing directory exists first, then write the file. */
func writeString(data string, elems ...string) {
	p := path.Join(elems...)
	if nil != ensureDir(p) {
		return
	}

	if err := ioutil.WriteFile(p, []byte(data), 0600); nil != err {
		log.Println("Unable to write string to file:", err)
	}
}

func ensureDir(dir string) error {
	dir = path.Dir(dir)
	_, err := os.Stat(dir)
	if os.IsExist(err) {
		return nil
	}
	if err := os.MkdirAll(dir, os.ModeDir|0700); nil != err {
		return errors.Wrap(err, "unable to create directory")
	}
	return nil
}

/* Read a pipe and send messages to the client each new line. */
func readMessagePipe(cli *mautrix.Client, pipe, roomID string) {
	readPipe(pipe, func(line string) {
		if _, err := cli.SendText(id.RoomID(roomID), line); nil != err {
			log.Println("Failed to send message:", err)
		}
	})
}

func readPipe(pipe string, onLine func(line string)) {
	if nil != ensureDir(pipe) {
		return
	}

	_, err := os.Stat(pipe)
	if os.IsNotExist(err) {
		if err := syscall.Mkfifo(pipe, syscall.S_IFIFO|0600); nil != err {
			log.Println("Failed to create pipe:", err)
			return
		}
	}

	for {
		str, err := ioutil.ReadFile(pipe)
		if err != nil {
			log.Println("Failed to read pipe:", err)
			continue
		}
		onLine(string(str))
	}
}

/* Read a pipe and send messages to the client each new line. */
func readTypingPipe(cli *mautrix.Client, pipe, roomID string) {
	readPipe(pipe, func(line string) {
		var typing bool
		if strings.TrimSpace(line) == "1" {
			typing = true
		}

		if _, err := cli.UserTyping(id.RoomID(roomID), typing, 15000); nil != err {
			log.Println("Failed to send typing status:", err)
		}
	})
}
