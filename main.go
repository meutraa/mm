package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const Json = "application/json"

func main() {
	usr, _ := user.Current()
	var host, username, pass, accPath string
	flag.StringVar(&host, "s", "", "homeserver url <https://matrix.org>")
	flag.StringVar(&username, "u", "", "not full qualified username <bob>")
	flag.StringVar(&pass, "p", "", "password <pass1234>")
	flag.StringVar(&accPath, "d", usr.HomeDir+"/mm", "directory path")
	flag.Parse()

	if host == "" || username == "" || pass == "" {
		flag.PrintDefaults()
		os.Exit(2)
	}

	/* Account login and setup. */
	var sesh session
	b, _ := json.Marshal(auth{"m.login.password", username, pass})
	body := readBody(http.Post(host+"/_matrix/client/r0/login", Json, bytes.NewBuffer(b)))
	json.Unmarshal(body, &sesh)
	if sesh.Token == "" {
		os.Exit(1)
	}
	accPath = path.Join(accPath, sesh.Homeserver, sesh.UserId)
	os.MkdirAll(accPath, 0700)

	/* Start reading existing pipes for sending. */
	rooms, _ := ioutil.ReadDir(accPath)
	for _, v := range rooms {
		if strings.HasPrefix(v.Name(), "!") {
			go readPipe(path.Join(accPath, v.Name(), "in"), host, sesh.Token)
		}
	}

	/* Sync loop. */
	for ; ; time.Sleep(5 * time.Second) {
		sync(host, sesh, accPath)
	}

	/* Revoke access_token. */
	readBody(http.Post(apistr(host, "logout?", sesh.Token), Json, nil))
}

func apistr(host string, call string, token string) string {
	return host + "/_matrix/client/r0/" + call + "access_token=" + token
}

func readPipe(pipe string, host string, token string) {
	roomId := path.Base(path.Dir(pipe))
	for {
		str, err := ioutil.ReadFile(pipe)
		if nil != err {
			fmt.Println("Could not read message:", roomId, err)
			continue
		}

		/* Send a message. */
		b, _ := json.Marshal(message{string(str), "m.text"})
		url := "rooms/" + roomId + "/send/m.room.message/" + strconv.FormatInt(time.Now().UnixNano(), 10) + "?"
		req, _ := http.NewRequest("PUT", apistr(host, url, token), bytes.NewBuffer(b))
		readBody(http.DefaultClient.Do(req))
	}
}

func sync(host string, sesh session, accPath string) {
	str := "sync?"
	if sesh.CurrentBatch != "" {
		str += "since=" + sesh.CurrentBatch + "&"
	}
	body := readBody(http.Get(apistr(host, str, sesh.Token)))

	var d data
	json.Unmarshal(body, &d)
	if d.NextBatch == "" {
		return
	}
	sesh.CurrentBatch = d.NextBatch

	for id, room := range d.Rooms.Join {
		roomPath := path.Join(accPath, id)
		os.Mkdir(roomPath, 0700)

		pipe := path.Join(roomPath, "in")
		_, stat := os.Stat(pipe)
		if os.IsNotExist(stat) {
			syscall.Mkfifo(pipe, syscall.S_IFIFO|0600)
			go readPipe(pipe, host, sesh.Token)
		}
		evs := &room.Timeline.Events
		for _, ev := range *evs {
			if ev.Type == "m.room.message" {
				/* Set directory to most recent message timestamp from sender. */
				sendPath := path.Join(roomPath, ev.Sender)
				os.Mkdir(sendPath, 0700)
				content := ev.Content.Body
				switch ev.Content.Type {
				case "m.image", "m.file", "m.video", "m.audio":
					content += ": " + ev.Content.Url
				case "m.location":
					content += ": " + ev.Content.GeoUri
				}

				tm := time.Unix(int64(ev.Timestamp/1000), int64(1000*(ev.Timestamp%1000)))
				file := path.Join(sendPath, ev.EventId)
				ioutil.WriteFile(file, []byte(content+"\n"), 0644)
				os.Chtimes(file, tm, tm)
			}
		}
		/* Send a read receipt. */
		if len(*evs) > 0 {
			url := "rooms/" + id + "/receipt/m.read/" + (*evs)[len(*evs)-1].EventId + "?"
			readBody(http.Post(apistr(host, url, sesh.Token), Json, nil))
		}
	}
}

func readBody(res *http.Response, err error) []byte {
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	if err != nil || res.StatusCode != 200 {
		fmt.Println(err, res.StatusCode, http.StatusText(res.StatusCode), string(body))
	}
	return body
}
