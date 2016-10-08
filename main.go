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
	Type    string  `json:"type"`
	Content content `json:"content"`
	Sender  string  `json:"sender"`
}

type content struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

type session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Homeserver   string `json:"home_server"`
	UserId       string `json:"user_id"`
	DeviceId     string `json:"device_id"`
}

func apistr(str string) string {
	return host + "/_matrix/client/r0/" + str + "access_token=" + sesh.AccessToken
}

func main() {
	usr, _ := user.Current()
	flag.StringVar(&host, "host", "", "matrix homeserver")
	flag.StringVar(&username, "user", "", "username")
	flag.StringVar(&pass, "pass", "", "password")
	flag.StringVar(&path, "path", usr.HomeDir+"/mm", "path")
	flag.Parse()

	if host == "" || username == "" || pass == "" {
		log.Fatalln("Usage: mm -host=https://server.org -user=bob -pass=1234 [-path=/home/bob/mm]")
	}

	login()
	sync()
	logout()
}

func sync() {
	res, _ := http.Get(apistr("sync?"))
	defer res.Body.Close()

	d := data{}
	if json.NewDecoder(res.Body).Decode(&d) != nil {
		fmt.Println("Unable to parse data")
	}

	for k, v := range d.Rooms.Join {
		fmt.Printf("%s : ", k)
		hostPath := path + "/" + sesh.Homeserver + "/"
		os.MkdirAll(hostPath+k, os.ModeDir|os.ModePerm)
		var name string
		for _, w := range v.State.Events {
			if w.Type == "m.room.name" {
				name = w.Content.Name
				break
			} else if w.Type == "m.room.member" && w.Sender != sesh.UserId {
				name = w.Sender
				break
			}
		}
		os.Symlink(hostPath+k, hostPath+name)
		fmt.Printf("%s\n", name)
	}
}

func logout() {
	res, _ := http.Post(apistr("logout?"), "application/json", nil)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		body, _ := ioutil.ReadAll(res.Body)
		fmt.Printf("Logout unsuccessful: %s\n", body)
	}
}

func login() {
	res, err := http.Post(host+"/_matrix/client/r0/login", "application/json",
		bytes.NewBuffer([]byte("{\"type\":\"m.login.password\",\"user\":\""+username+"\",\"password\":\""+pass+"\"}")))
	if err != nil || res.StatusCode != 200 {
		log.Fatalf("Login failed: %s, %s\n", err, http.StatusText(res.StatusCode))
	}
	defer res.Body.Close()

	if json.NewDecoder(res.Body).Decode(&sesh) != nil || sesh.AccessToken == "" {
		log.Fatalf("Login response could not be parsed: %s,%s\n", err, res.Body)
	}
}
