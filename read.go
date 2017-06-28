package main

import (
	"log"
	"time"
	"os"
	"path"
	"syscall"
	"io/ioutil"
	"strings"
	"sync"
)

var pipeMutex = sync.Mutex{}
var pipes = map[string]bool{}

func watchPipes(accPath, server, accessToken string) {
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
		go readPipe(pipe, server, accessToken)
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

		msgMutex.Lock()
		bufferedMessages[string(str)] = roomID
		msgMutex.Unlock()

		sendMessage(host, roomID, string(str), token)
	}
}
