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
	"syscall"

	"github.com/matrix-org/gomatrix"
)

func main() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	var server, username, pass, root, cert string
	flag.StringVar(&server, "s", "", "homeserver url <https://matrix.org>")
	flag.StringVar(&username, "u", usr.Username, "not full qualified username <bob>")
	flag.StringVar(&pass, "p", "", "password <pass1234>")
	flag.StringVar(&root, "d", usr.HomeDir+"/.mm-test", "directory path")
	flag.StringVar(&cert, "c", "", "certificate path")
	flag.Parse()

	if server == "" || pass == "" {
		flag.PrintDefaults()
		os.Exit(2)
	}

	cli, err := gomatrix.NewClient(server, "", "")
	if err != nil {
		panic(err)
	}

	resp, err := cli.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     username,
		Password: pass,
	})
	if err != nil {
		panic(err)
	}
	cli.SetCredentials(resp.UserID, resp.AccessToken)

	/* Change the root to the user account. */
	root = path.Join(root, cli.HomeserverURL.Hostname(), cli.UserID)
	if err = os.MkdirAll(root, 0700); nil != err {
		log.Println("Unable to create account directory:", err)
	}

	/* From now on Logout after panicking. */
	defer func() {
		if r := recover(); nil != r {
			log.Println(r)
			logout(cli, root)
		}
	}()

	/* Logout on interrupt signal. */
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	go func() {
		for _ = range c {
			logout(cli, root)
		}
	}()

	/* Read the nextbatch file if it exists. We only care here if nbBytes
	   is non null. */
	nbBytes, _ := ioutil.ReadFile(path.Join(root, "nextbatch"))
	if nil != nbBytes {
		cli.Store.SaveNextBatch(cli.UserID, string(nbBytes))
	}

	syncer := cli.Syncer.(*gomatrix.DefaultSyncer)
	syncer.OnEventType("m.room.message", func(ev *gomatrix.Event) {
		msgDir := path.Join(root, ev.RoomID, ev.Sender)
		msgPath := path.Join(msgDir, ev.ID)
		if nil != os.MkdirAll(msgDir, os.ModeDir|0700) {
			log.Println(err)
			return
		}

		msg, ok := ev.Body()
		if !ok {
			log.Println("Failed to parse body of eventID:", ev.ID)
			return
		}

		if err = ioutil.WriteFile(msgPath, []byte(msg), 0600); nil != err {
			log.Println("Failed to write message:", err)
			return
		}
		fmt.Println(msgPath)
	})
	syncer.OnEventType("m.room.typing", func(ev *gomatrix.Event) {
		fmt.Println(ev.Body())
	})

	/* Start reading existing pipes for sending. */
	// watchPipes(accPath, server, session.AccessToken)

	if err := cli.Sync(); err != nil {
		panic(err)
	}
}

func logout(cli *gomatrix.Client, root string) {
	/* Write the nextbatch file. */
	nextbatch := []byte(cli.Store.LoadNextBatch(cli.UserID))
	if err := ioutil.WriteFile(path.Join(root, "nextbatch"), nextbatch, 0600); nil != err {
		log.Println("Unable to save nextbatch file:", err)
	} else {
		log.Println("Saved nextbatch file")
	}

	/* Logout from the server. */
	if _, err := cli.Logout(); nil != err {
		log.Println(err)
	} else {
		log.Println("Logged out")
	}
	os.Exit(1)
}
