package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path"
	"syscall"
	"strings"
	"sync"
	"time"
	"net/http"
)

//var client = &http.Client{}
var currentBatch string

var pipeMutex = sync.Mutex{}
var pipes = map[string]bool{}

func main() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	var server, username, pass, accPath, cert string
	flag.StringVar(&server, "s", "", "homeserver url <https://matrix.org>")
	flag.StringVar(&username, "u", usr.Username, "not full qualified username <bob>")
	flag.StringVar(&pass, "p", "", "password <pass1234>")
	flag.StringVar(&accPath, "d", usr.HomeDir+"/mm", "directory path")
	flag.StringVar(&cert, "c", "", "certificate path")
	flag.Parse()

	if server == "" || pass == "" {
		flag.PrintDefaults()
		os.Exit(2)
	}

	/* Self signed certificate */
	//if "" != cert {
	//	rootPEM, err := ioutil.ReadFile(cert)
	//	if err != nil {
	//		log.Println(err)
	//	} else if len(rootPEM) > 0 {
	//		roots := x509.NewCertPool()
	//		if roots.AppendCertsFromPEM(rootPEM) {
	//			client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots}}}
	//		} else {
	//			log.Println("failed to parse certificate:", cert)
	//		}
	//	}
	//}

	http.Get(server)
	//client.Timeout = time.Second * 15

	session := login(server, username, pass)

	/* Logout on interrupt signal. */
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	go func() {
		for _ = range c {
			logout(server, session.AccessToken)
			os.Exit(1)
		}
	}()

	accPath = path.Join(accPath, session.HomeServer, session.UserId)
	os.MkdirAll(accPath, 0700)

	/* Start reading existing pipes for sending. */
	rooms, err := ioutil.ReadDir(accPath)
	if err != nil {
		panic(err)
	}

	for _, v := range rooms {
		if !strings.HasPrefix(v.Name(), "!") {
			continue
		}
		pipe := path.Join(accPath, v.Name(), "in")
		pipeMutex.Lock()
		pipes[pipe] = true
		pipeMutex.Unlock()
		go readPipe(pipe, server, session.AccessToken)
	}

	for {
		syncronize(server, session, accPath)
	}
}

func readPipe(pipe string, host string, token string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(recover())
			time.Sleep(time.Second * 10)
		}
	}()

	pipeMutex.Lock()
	if !pipes[pipe] {
		if err := os.MkdirAll(path.Dir(pipe), 0700); err != nil {
			panic(err)
		}
		if err := syscall.Mkfifo(pipe, syscall.S_IFIFO|0600); err != nil {
			panic(err)
		}
		pipes[pipe] = true
	}
	pipeMutex.Unlock()

	roomID := path.Base(path.Dir(pipe))
	for {
		str, err := ioutil.ReadFile(pipe)
		if err != nil {
			log.Println(err)
			continue
		}
		sendMessage(host, roomID, string(str), token)
	}
}
