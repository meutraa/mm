package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

var sesh session
var host, username, pass, accPath string

type data struct {
	NextBatch string `json:"next_batch"`
	Rooms     rooms  `json:"rooms"`
}

type rooms struct {
	Join map[string]joinedRooms `json:"join"`
}

type joinedRooms struct {
	State    state    `json:"state"`
	Timeline timeline `json:"timeline"`
}

type state struct {
	Events []event `json:"events"`
}

type timeline struct {
	Events []event `json:"events"`
}

type event struct {
	Timestamp int     `json:"origin_server_ts"`
	EventId   string  `json:"event_id"`
	Type      string  `json:"type"`
	Content   content `json:"content"`
	Sender    string  `json:"sender"`
}

type message struct {
	Body string `json:"body"`
	Type string `json:"msgtype"`
}

type content struct {
	message
	Name   string `json:"name"`
	Url    string `json:"url"`
	GeoUri string `json:"geo_uri"`
}

type session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Homeserver   string `json:"home_server"`
	UserId       string `json:"user_id"`
	DeviceId     string `json:"device_id"`
	CurrentBatch string
	TxnId        int64
}

func apistr(str string) string {
	return host + "/_matrix/client/r0/" + str + "access_token=" + sesh.AccessToken
}

func sendMessage(roomId string, text string) {
	b, _ := json.Marshal(message{text, "m.text"})
	var client http.Client
	req, _ := http.NewRequest("PUT", apistr("rooms/"+roomId+
		"/send/m.room.message/"+strconv.FormatInt(sesh.TxnId, 10)+"?"),
		bytes.NewBuffer(b))
	sesh.TxnId += 1
	res, err := client.Do(req)
	body := check(res, err)
	if len(body) == 0 {
		fmt.Println("Unable to send message:", roomId, ":", text)
	}
}

var walker = func(path string, info os.FileInfo, err error) error {
	if info.Name() == "in" {
		go readPipe(path)
	}
	return nil
}

func main() {
	usr, _ := user.Current()
	flag.StringVar(&host, "s", "", "homeserver url <https://matrix.org>")
	flag.StringVar(&username, "u", "", "not full qualified username <bob>")
	flag.StringVar(&pass, "p", "", "password <pass1234>")
	flag.StringVar(&accPath, "d", usr.HomeDir+"/mm", "directory path")
	flag.Parse()

	if host == "" || username == "" || pass == "" {
		flag.PrintDefaults()
		return
	}

	login()
	accPath = path.Join(accPath, sesh.Homeserver, sesh.UserId)
	mkdir(accPath)
	filepath.Walk(accPath, walker)

	for ; ; time.Sleep(5 * time.Second) {
		sync()
	}
	logout()
}

func mkdir(dir string) bool {
	_, stat := os.Stat(dir)
	created := os.IsNotExist(stat)
	if os.IsNotExist(stat) {
		os.MkdirAll(dir, 0700)
	}
	return created
}

func readPipe(pipe string) {
	roomId := path.Base(path.Dir(pipe))
	for {
		str, err := ioutil.ReadFile(pipe)
		if nil != err || len(str) == 0 {
			fmt.Println("Could not read message:", roomId, err)
			continue
		}
		sendMessage(roomId, string(str))
	}
}

func addfifo(roomPath string) {
	pipe := path.Join(roomPath, "in")
	syscall.Mkfifo(pipe, syscall.S_IFIFO|0600)
	go readPipe(pipe)
}

func mkroom(id string, states []event) string {
	roomPath := path.Join(accPath, id)
	if mkdir(roomPath) {
		/* Is there a better way to get the room name/member? */
		addfifo(roomPath)
		var name string
		for _, ev := range states {
			if ev.Type == "m.room.name" {
				name = ev.Content.Name
				break
			}
			if ev.Type == "m.room.member" && ev.Sender != sesh.UserId {
				name = ev.Sender
				break
			}
		}
		os.Symlink(roomPath, path.Join(accPath, name))
	}
	return roomPath
}

func sync() {
	str := "sync?"
	if sesh.CurrentBatch != "" {
		str += "since=" + sesh.CurrentBatch + "&"
	}
	res, err := http.Get(apistr(str))
	body := check(res, err)
	if len(body) == 0 {
		fmt.Println("Unable to sync data")
		return
	}

	var d data
	if json.Unmarshal(body, &d) != nil {
		fmt.Println("Unable to parse data:", body)
		return
	}
	sesh.CurrentBatch = d.NextBatch

	for id, room := range d.Rooms.Join {
		roomPath := mkroom(id, room.State.Events)
		for i, ev := range room.Timeline.Events {
			if i == len(room.Timeline.Events)-1 {
				pes, per := http.Post(apistr("rooms/"+id+
					"/receipt/m.read/"+ev.EventId+"?"),
					"application/json", bytes.NewBuffer([]byte("")))
				check(pes, per)
			}

			if ev.Type != "m.room.message" {
				continue
			}
			/* Set directory to most recent message timestamp from sender. */
			sendPath := path.Join(roomPath, ev.Sender)
			mkdir(sendPath)
			content := ev.Content.Body
			switch ev.Content.Type {
			case "m.image", "m.file", "m.video", "m.audio":
				content += ": " + ev.Content.Url
			case "m.location":
				content += ": " + ev.Content.GeoUri
			}

			tm := time.Unix(int64(ev.Timestamp/1000), int64(1000*(ev.Timestamp%1000)))
			file := path.Join(sendPath, strconv.Itoa(ev.Timestamp))
			ioutil.WriteFile(file, []byte(content+"\n"), 0644)
			os.Chtimes(file, tm, tm)
		}
	}
}

func check(res *http.Response, err error) []byte {
	defer res.Body.Close()
	body, err2 := ioutil.ReadAll(res.Body)
	if err != nil || res.StatusCode != 200 {
		fmt.Println(err, res.StatusCode, http.StatusText(res.StatusCode))
		if err2 == nil {
			fmt.Printf("%s\n", body)
		}
		return []byte("")
	}
	return body
}

func logout() {
	res, err := http.Post(apistr("logout?"), "application/json", nil)
	check(res, err)
}

func login() {
	res, err := http.Post(host+"/_matrix/client/r0/login", "application/json",
		bytes.NewBuffer([]byte("{\"type\":\"m.login.password\",\"user\":\""+username+"\",\"password\":\""+pass+"\"}")))

	body := check(res, err)
	if len(body) == 0 {
		log.Fatalln("Login failed")
	}
	err = json.Unmarshal(body, &sesh)
	if err != nil || sesh.AccessToken == "" {
		log.Fatalf("Login response not decoded: %s, %s\n", err, body)
	}
}
