package main

import (
	"log"
)

func receiptAddress(host, roomId, eventId string) string {
	return host + "/_matrix/client/r0/rooms/" + roomId + "/receipt/m.read/" + eventId
}

func (e event) sendReceipt(host, roomId, accessToken string) {
	if e.EventId == "" {
		return
	}
	address := authenticate(receiptAddress(host, roomId, e.EventId), accessToken)
	res := PostJSON(address, "")
	if 200 != res.StatusCode {
		log.Println(res.Status)
	}
}