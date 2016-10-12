package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
	var server, username, pass, accPath string
	flag.StringVar(&server, "s", "", "homeserver url <https://matrix.org>")
	flag.StringVar(&username, "u", "", "not full qualified username <bob>")
	flag.StringVar(&pass, "p", "", "password <pass1234>")
	flag.StringVar(&accPath, "d", usr.HomeDir+"/mm", "directory path")
	flag.Parse()

	host, err := url.Parse(server)
	if server == "" || username == "" || pass == "" || err != nil {
		flag.PrintDefaults()
		os.Exit(2)
	}
	if host.Scheme == "" {
		host.Scheme = "https"
	}

	/* Account login and setup. */
	var sesh session
	b, _ := json.Marshal(auth{"m.login.password", username, pass})
	body, _ := readBody(http.Post(host.String()+"/_matrix/client/r0/login", Json, bytes.NewBuffer(b)))
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
			go readPipe(path.Join(accPath, v.Name(), "in"), host.String(), sesh.Token)
		}
	}

	/* Sync loop. */
	for ; ; time.Sleep(5 * time.Second) {
		sesh.CurrentBatch = sync(host.String(), sesh, accPath)
	}

	/* Revoke access_token. */
	http.Post(apistr(host.RawPath, "logout?", sesh.Token), Json, nil)
}

func apistr(host string, call string, token string) string {
	return host + "/_matrix/client/r0/" + call + "access_token=" + token
}

func readPipe(pipe string, host string, token string) {
	roomId := path.Base(path.Dir(pipe))
	for {
		str, err := ioutil.ReadFile(pipe)
		if err != nil {
			log.Println("Could not read message:", pipe, roomId, err)
			continue
		}

		/* Send a message. */
		b, _ := json.Marshal(message{string(str), "m.text"})
		url := "rooms/" + roomId + "/send/m.room.message/" + strconv.FormatInt(time.Now().UnixNano(), 10) + "?"
		req, _ := http.NewRequest("PUT", apistr(host, url, token), bytes.NewBuffer(b))
		readBody(http.DefaultClient.Do(req))
	}
}

func sync(host string, sesh session, accPath string) string {
	str := "sync?"
	if sesh.CurrentBatch != "" {
		str += "since=" + sesh.CurrentBatch + "&"
	}
	body, _ := readBody(http.Get(apistr(host, str, sesh.Token)))

	var d data
	if json.Unmarshal(body, &d) != nil || d.NextBatch == "" {
		return sesh.CurrentBatch
	}

	for id, room := range d.Rooms.Join {
		roomPath := path.Join(accPath, id)
		os.Mkdir(roomPath, 0700)

		pipe := path.Join(roomPath, "in")
		_, stat := os.Stat(pipe)
		if os.IsNotExist(stat) {
			syscall.Mkfifo(pipe, syscall.S_IFIFO|0600)
			go readPipe(pipe, host, sesh.Token)
		}
		var lastId string
		for _, e := range room.Timeline.Events {
			lastId = e.EventId
			if e.Type != "m.room.message" {
				continue
			}
			file := path.Join(roomPath, e.Sender, e.EventId)
			os.Mkdir(path.Dir(file), 0700)

			_, stat = os.Stat(file)
			if os.IsNotExist(stat) {
				s := strings.TrimSpace(e.Content.Body + " " + e.Content.Url + " " + e.Content.GeoUri)
				ioutil.WriteFile(file, []byte(s+"\n"), 0644)

				t := time.Unix(e.Timestamp/1000, 1000*(e.Timestamp%1000))
				os.Chtimes(file, t, t)
				fmt.Println(file)
			}
		}
		/* Send a read receipt. */
		if lastId != "" {
			url := "rooms/" + id + "/receipt/m.read/" + lastId + "?"
			readBody(http.Post(apistr(host, url, sesh.Token), Json, nil))
		}
	}
	return d.NextBatch
}

func readBody(res *http.Response, err error) ([]byte, error) {
	if err != nil {
		log.Println(err)
		return []byte(""), err
	}
	/* Body always non-nil when err == nil. */
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil || res.StatusCode != 200 {
		log.Println(err, res.StatusCode, http.StatusText(res.StatusCode))
	}
	return body, err
}
