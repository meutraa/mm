package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/matrix-org/gomatrix"
)

func xdgConfigDir(home string) string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if "" == dir {
		dir = path.Join(home, ".config", "mm")
	} else {
		dir = path.Join(dir, "mm")
	}
	return dir
}

func xdgDataDir(home string) string {
	dir := os.Getenv("XDG_DATA_HOME")
	if "" == dir {
		dir = path.Join(home, ".local", "share", "mm")
	} else {
		dir = path.Join(dir, "mm")
	}
	return dir
}

func main() {
	usr, err := user.Current()
	if nil != err {
		log.Println("Unable to get current user:", err)
		quit(nil)
	}

	var server, username, pass, cert, root string
	flag.StringVar(&server, "s", "https://matrix.org", "homeserver")
	flag.StringVar(&username, "u", usr.Username, "username")
	flag.StringVar(&pass, "p", "", "password")
	flag.StringVar(&root, "d", xdgDataDir(usr.HomeDir), "data storage directory")
	flag.StringVar(&cert, "c", path.Join(xdgConfigDir(usr.HomeDir), "cert.pem"), "certificate path")
	flag.Parse()

	cli, err := gomatrix.NewClient(server, "", "")
	if nil != err {
		log.Println("Unable to parse homeserver url:", err)
		quit(cli)
	}
	cli.Client = createClient(cert)

	login(cli, username, pass)

	/* Change the root to the user account. */
	root = path.Join(root, cli.HomeserverURL.Hostname(), cli.UserID)

	/* Logout on interrupt signal. */
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	go func() {
		for _ = range c {
			quit(cli)
		}
	}()

	go createRooms(cli, root)

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

	quit(cli)
}

func createClient(cert string) *http.Client {
	tr := &http.Transport{
		DisableKeepAlives: true,
		IdleConnTimeout:   10 * time.Second,
	}

	/* Self signed certificate */
	rootPEM, _ := ioutil.ReadFile(cert)
	if string(rootPEM) != "" {
		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM(rootPEM)
		if !ok {
			log.Println("failed to parse certificate:", cert)
		}
		tr.TLSClientConfig = &tls.Config{
			RootCAs: roots,
		}
	}

	return &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}
}

func login(cli *gomatrix.Client, user, pass string) {
	resp, err := cli.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     user,
		Password: pass,
	})
	if nil != err {
		log.Println("Unable to login:", err)
		quit(cli)
	}
	cli.SetCredentials(resp.UserID, resp.AccessToken)
}

func handleMessage(ev *gomatrix.Event, root string) {
	/* Parse the body of the message. */
	msg, ok := ev.Body()
	if !ok {
		log.Println("Failed to parse body of eventID:", ev.ID)
		return
	}

	file := path.Join(root, ev.RoomID, ev.Sender, ev.ID)
	writeString(msg, file)

	/* Set the creation time of the file to the timestamp of the server. */
	t := time.Unix(int64(ev.Timestamp/1000), 0)
	if err := os.Chtimes(file, time.Now(), t); nil != err {
		log.Println("Failed to set timestamp on message:", err)
	}

	/* Print the file path to stdout for clients. */
	fmt.Println(path.Join(root, ev.RoomID, ev.Sender, ev.ID))
}

func quit(cli *gomatrix.Client) {
	/* If the client has not connected yet, just quit. */
	if nil == cli {
		os.Exit(0)
	}

	/* Logout from the server. */
	if _, err := cli.Logout(); nil != err {
		log.Println(err)
	}

	os.Exit(0)
}

func rooms(cli *gomatrix.Client) []string {
	rooms, err := cli.JoinedRooms()
	if nil != err {
		log.Println("Unable to get list of joined rooms:", err)
		return []string{}
	}
	return rooms.JoinedRooms
}

func forEachMember(cli *gomatrix.Client, room string, forMember func(id, avatar, name string)) {
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
func createRooms(cli *gomatrix.Client, root string) {
	for _, room := range rooms(cli) {
		/* Read the input pipe and send messages. */
		go readMessagePipe(cli, path.Join(root, room, "in"), room)

		/* For each joined member of the room. */
		forEachMember(cli, room, func(id, avatar, name string) {
			memberPath := path.Join(root, room, id)

			/* If this user is yourself. */
			if cli.UserID == id {
				go readTypingPipe(cli, path.Join(memberPath, "typing"), room)
			}

			/* Write the members display name and avatar to file. */
			writeString(name, root, room, id, "name")
			writeString(avatar, root, room, id, "avatar")
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
		log.Println("Failed to create directory:", err)
		return errors.New("Failed to create directory")
	}
	return nil
}

/* Read a pipe and send messages to the client each new line. */
func readMessagePipe(cli *gomatrix.Client, pipe, roomID string) {
	readPipe(pipe, func(line string) {
		if _, err := cli.SendText(roomID, line); nil != err {
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
func readTypingPipe(cli *gomatrix.Client, pipe, roomID string) {
	readPipe(pipe, func(line string) {
		var typing bool
		if strings.TrimSpace(line) == "1" {
			typing = true
		}

		if _, err := cli.UserTyping(roomID, typing, 15000); nil != err {
			log.Println("Failed to send typing status:", err)
		}
	})
}
