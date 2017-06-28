package main

import (
	"strconv"
	"time"
	"sync"
)

var msgMutex = sync.Mutex{}
var bufferedMessages = map[string]string{}

type SendMessageRequest struct {
	Type string `json:"msgtype"`
	Body string `json:"body"`
}

func sendMessageAddress(host, roomId string) string {
	return host + "/_matrix/client/r0/rooms/" + roomId + "/send/m.room.message/" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func sendMessage(host, roomId, message, accessToken string) {
	address := authenticate(sendMessageAddress(host, roomId), accessToken)
	res := PutJSON(address, SendMessageRequest{"m.text", message})
	if 200 != res.StatusCode {
		panic(res.Status)
	}

	msgMutex.Lock()
	delete(bufferedMessages, message)
	msgMutex.Unlock()
}

func sendBufferedMessages(host, accessToken string) {
	for message, roomId := range bufferedMessages {
		go sendMessage(host, roomId, message, accessToken)
	}
}
