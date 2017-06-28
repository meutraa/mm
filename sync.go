package main

import (
	"log"
	"net/url"
	"os"
	"path"
	"runtime/debug"
	"time"
)

type SyncResponse struct {
	NextBatch string `json:"next_batch"`
	Rooms     rooms  `json:"rooms"`
}

type rooms struct {
	Join map[string]joinedRooms `json:"join"`
}

type joinedRooms struct {
	Timeline timeline `json:"timeline"`
}

type timeline struct {
	Events    []event `json:"events"`
	PrevBatch string  `json:"prev_batch"`
}

type event struct {
	Timestamp int64   `json:"origin_server_ts"`
	EventId   string  `json:"event_id"`
	Type      string  `json:"type"`
	Content   content `json:"content"`
	Sender    string  `json:"sender"`
}

type content struct {
	Body     string `json:"body"`
	Type     string `json:"msgtype"`
	Name     string `json:"name"`
	Url      string `json:"url"`
	GeoUri   string `json:"geo_uri"`
	FileInfo info   `json:"info"`
}

type info struct {
	Height   int    `json:"h"`
	Width    int    `json:"w"`
	Size     int    `json:"size"`
	MimeType string `json:"mimetype"`
}

var syncAddress = "/_matrix/client/r0/sync"

func synchronize(host string, session LoginResponse, accPath string) {
	/* After an error, wait 30s, otherwise sync again. */
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
			log.Println(recover())
			time.Sleep(time.Second * 10)
		}
	}()

	u, err := url.Parse(host + syncAddress)
	if err != nil {
		panic(err)
	}

	params := u.Query()
	if currentBatch != "" {
		params.Set("since", currentBatch)
		params.Set("timeout", "15000")
	}
	u.RawQuery = params.Encode()

	address := authenticate(u.String(), session.AccessToken)

	res, err := client().Get(address)
	if nil != res {
		defer res.Body.Close()
	}

	if nil != err {
		panic(res.Status)
	}
	if 200 != res.StatusCode {
		panic(res.Status)
	}

	var data SyncResponse
	ReadJSON(res, &data)

	for roomID, room := range data.Rooms.Join {
		roomPath := path.Join(accPath, roomID)

		pipe := path.Join(roomPath, "in")
		_, err := os.Stat(pipe)
		if os.IsNotExist(err) {
			go readPipe(pipe, host, session.AccessToken)
		}

		var lastEvent event
		for _, e := range room.Timeline.Events {
			lastEvent = e
			if e.Type != "m.room.message" {
				continue
			}
			e.Save(path.Join(roomPath, e.Sender, e.EventId), host)
		}
		lastEvent.sendReceipt(host, roomID, session.AccessToken)
	}
	currentBatch = data.NextBatch
}
