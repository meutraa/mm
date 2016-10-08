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
	"strconv"
	"time"
)

var sesh session
var host, username, pass, path string

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
	Type      string  `json:"type"`
	Content   content `json:"content"`
	Sender    string  `json:"sender"`
}

type content struct {
	Name        string `json:"name"`
	Body        string `json:"body"`
	MessageType string `json:"msgtype"`
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

func sendMessage(roomId string, message string) {
	b, _ := json.Marshal(content{"", message, "m.text"})
	var client http.Client
	req, _ := http.NewRequest("PUT", apistr("rooms/"+roomId+
		"/send/m.room.message/"+strconv.FormatInt(sesh.TxnId, 10)+"?"),
		bytes.NewBuffer(b))
	sesh.TxnId += 1
	res, err := client.Do(req)
	body := check(res, err)
	if len(body) == 0 {
		fmt.Println("Unable to send message:", roomId, ":", message)
	}
}

func main() {
	usr, _ := user.Current()
	flag.StringVar(&host, "s", "", "homeserver url <https://matrix.org>")
	flag.StringVar(&username, "u", "", "not full qualified username <bob>")
	flag.StringVar(&pass, "p", "", "password <pass1234>")
	flag.StringVar(&path, "d", usr.HomeDir+"/mm", "directory path")
	flag.Parse()

	if host == "" || username == "" || pass == "" {
		flag.PrintDefaults()
		return
	}

	login()
	sesh.TxnId = time.Now().UnixNano()
	path += "/" + sesh.Homeserver + "/"
	for {
		sync()
		time.Sleep(5 * time.Second)
	}
	logout()
}

func sync() {
	var res *http.Response
	var err error
	if sesh.CurrentBatch == "" {
		res, err = http.Get(apistr("sync?"))
	} else {
		res, err = http.Get(apistr("sync?since=" + sesh.CurrentBatch + "&"))
	}
	body := check(res, err)
	if len(body) == 0 {
		fmt.Println("Unable to sync data")
		return
	}

	d := data{}
	if json.Unmarshal(body, &d) != nil {
		fmt.Println("Unable to parse data:", body)
		return
	}
	sesh.CurrentBatch = d.NextBatch

	for id, room := range d.Rooms.Join {
		os.MkdirAll(path+id, os.ModeDir|os.ModePerm)
		os.Create(path + id + "/in")
		os.Chmod(path+id+"/in", os.ModeNamedPipe|0600)
		/* Is there a better way to get the room name/member? */
		for _, ev := range room.State.Events {
			if ev.Type == "m.room.name" {
				os.Symlink(path+id, path+ev.Content.Name)
				break
			}
			if ev.Type == "m.room.member" && ev.Sender != sesh.UserId {
				os.Symlink(path+id, path+ev.Sender)
				break
			}
		}
		for _, ev := range room.Timeline.Events {
			tm := time.Unix(int64(ev.Timestamp/1000), int64(1000*(ev.Timestamp%1000)))
			os.Mkdir(path+id+"/"+ev.Sender, os.ModeDir|os.ModePerm)
			mtime := strconv.Itoa(ev.Timestamp)
			file := path + id + "/" + ev.Sender + "/" + mtime
			ioutil.WriteFile(file, []byte(ev.Content.Body+"\n"), 0644)
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
