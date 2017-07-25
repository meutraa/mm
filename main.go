package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path"
	"strings"
	"syscall"

	"github.com/matrix-org/gomatrix"
)

func main() {
	usr, err := user.Current()
	if nil != err {
		log.Println("Unable to get current user:", err)
		quit(nil, "")
	}

	/* Find the default data storage location. */
	xdgdir := os.Getenv("XDG_DATA_HOME")
	if "" == xdgdir {
		xdgdir = path.Join(usr.HomeDir, ".local", "share", "mm")
	}

	var server, username, pass, root string
	flag.StringVar(&server, "s", "https://matrix.org", "homeserver")
	flag.StringVar(&username, "u", usr.Username, "username")
	flag.StringVar(&pass, "p", "", "password")
	flag.StringVar(&root, "d", xdgdir, "data storage directory")
	flag.Parse()

	if pass == "" {
		flag.PrintDefaults()
		os.Exit(2)
	}

	cli, err := gomatrix.NewClient(server, "", "")
	if nil != err {
		log.Println("Unable to create a new client:", err)
		quit(cli, root)
	}

	resp, err := cli.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     username,
		Password: pass,
	})
	if nil != err {
		log.Println("Unable to login:", err)
		quit(cli, root)
	}
	cli.SetCredentials(resp.UserID, resp.AccessToken)

	/* Change the root to the user account. */
	root = path.Join(root, cli.HomeserverURL.Hostname(), cli.UserID)
	if err = os.MkdirAll(root, 0700); nil != err {
		log.Println("Unable to create account directory:", err)
		quit(nil, root)
	}

	/* Logout on interrupt signal. */
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	go func() {
		for _ = range c {
			quit(cli, root)
		}
	}()

	go createRooms(cli, root)

	/* Read the nextbatch file if it exists. We only care here if it is non null. */
	if nbBytes, _ := ioutil.ReadFile(path.Join(root, "nextbatch")); nil != nbBytes {
		cli.Store.SaveNextBatch(cli.UserID, string(nbBytes))
	}

	/* Create our syncer. What will happen when we receive an event. */
	cli.Syncer.(*gomatrix.DefaultSyncer).OnEventType(
		"m.room.message", func(ev *gomatrix.Event) {
			handleMessage(ev, root)
		})

	/* Sync loop. */
	err = cli.Sync()
	if nil != err {
		println("Sync loop exited:", err)
	}

	quit(cli, root)
}

func handleMessage(ev *gomatrix.Event, root string) {
	/* Ensure the sender directory exists. */
	msgPath := path.Join(root, ev.RoomID, ev.Sender, ev.ID)
	if err := os.MkdirAll(path.Dir(msgPath), os.ModeDir|0700); nil != err {
		log.Println("Failed to create message's directory:", err)
		return
	}

	/* Parse the body of the message. */
	msg, ok := ev.Body()
	if !ok {
		log.Println("Failed to parse body of eventID:", ev.ID)
		return
	}

	/* Write the body to a file. */
	if err := ioutil.WriteFile(msgPath, []byte(msg), 0600); nil != err {
		log.Println("Failed to write message:", err)
		return
	}

	/* Print the file path to stdout for clients. */
	fmt.Println(msgPath)
}

func quit(cli *gomatrix.Client, root string) {
	/* If the client has not connected yet, just quit. */
	if nil == cli || "" == root {
		os.Exit(0)
	}

	/* Write the nextbatch file. */
	nextbatch := []byte(cli.Store.LoadNextBatch(cli.UserID))
	if err := ioutil.WriteFile(path.Join(root, "nextbatch"), nextbatch, 0600); nil != err {
		log.Println("Unable to save nextbatch file:", err)
	}

	/* Logout from the server. */
	if _, err := cli.Logout(); nil != err {
		log.Println(err)
	}

	os.Exit(0)
}

/* Get a list of joined rooms and set up a pipe for reading messages from. */
func createRooms(cli *gomatrix.Client, root string) {
	rooms, err := cli.JoinedRooms()
	if nil != err {
		log.Println("Unable to get list of joined rooms:", err)
		return
	}

	for _, room := range rooms.JoinedRooms {
		pipe := path.Join(root, room, "in")

		/* Ensure the room directory exists. */
		if err = os.MkdirAll(path.Dir(pipe), 0700); nil != err {
			log.Println("Unable to create room directory:", err)
			continue
		}

		/* Get a list of all joined members of this room. */
		members, err := cli.JoinedMembers(room)
		if nil != err {
			log.Println("Unable to get members of room:", room, ":", err)
		} else {
			/* For each joined member of the room. */
			for id, member := range members.Joined {
				memberPath := path.Join(root, room, id)

				/* Ensure the member directory exists. */
				if err = os.MkdirAll(memberPath, 0700); nil != err {
					log.Println("Unable to create member directory:", err)
					continue
				}

				/* If this user is yourself. */
				if cli.UserID == id {
					typingPipe := path.Join(memberPath, "typing")

					/* Create the input pipe. */
					if err = syscall.Mkfifo(typingPipe, syscall.S_IFIFO|0600); nil != err {
						log.Println("Pipe error for:", room, err)
					}

					/* Read the typing pipe and send status. */
					go readTypingPipe(cli, typingPipe, room)
				}

				/* Write the members display name to file. */
				if nil != member.DisplayName && len(*(member.DisplayName)) > 0 {
					if err2 := ioutil.WriteFile(path.Join(memberPath, "name"), []byte(*(member.DisplayName)), 0600); nil != err2 {
						log.Println("Unable to write member name file:", err2)
					}
				}

				/* Write the member's avartar url to file. */
				if nil != member.AvatarURL && len(*(member.AvatarURL)) > 0 {
					if err2 := ioutil.WriteFile(path.Join(memberPath, "avatar"), []byte(*(member.AvatarURL)), 0600); nil != err2 {
						log.Println("Unable to write member avatar file:", err2)
					}
				}
			}
		}

		/* Create the input pipe. */
		if err = syscall.Mkfifo(pipe, syscall.S_IFIFO|0600); nil != err {
			log.Println("Pipe error for:", room, err)
		}

		/* Read the input pipe and send messages. */
		go readMessagePipe(cli, pipe, room)
	}
}

/* Read a pipe and send messages to the client each new line. */
func readMessagePipe(cli *gomatrix.Client, pipe, roomID string) {
	for {
		str, err := ioutil.ReadFile(pipe)
		if err != nil {
			log.Println(err)
			continue
		}

		if _, err = cli.SendText(roomID, string(str)); nil != err {
			log.Println("Failed to send message:", err)
		}
	}
}

/* Read a pipe and send messages to the client each new line. */
func readTypingPipe(cli *gomatrix.Client, pipe, roomID string) {
	for {
		str, err := ioutil.ReadFile(pipe)
		if err != nil {
			log.Println("Failed to read typing pipe:", err)
			continue
		}

		var typing bool
		if strings.TrimSpace(string(str)) == "1" {
			typing = true
		}

		if _, err = cli.UserTyping(roomID, typing, 15000); nil != err {
			log.Println("Failed to send typing status:", err)
		}
	}
}
