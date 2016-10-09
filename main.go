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
		return
	}

	sesh := login(host, username, pass)
	accPath = path.Join(accPath, sesh.Homeserver, sesh.UserId)
	mkdir(accPath)
	var walker = func(path string, info os.FileInfo, err error) error {
		if info.Name() == "in" {
			go readPipe(path, host, sesh.AccessToken)
		}
		return nil
	}
	filepath.Walk(accPath, walker)

	for ; ; time.Sleep(5 * time.Second) {
		sync(host, sesh, accPath)
	}
	logout(host, sesh.AccessToken)
}

func apistr(host string, call string, token string) string {
	return host + "/_matrix/client/r0/" + call + "access_token=" + token
}

func sendMessage(host string, token string, roomId string, text string) {
	b, _ := json.Marshal(message{text, "m.text"})
	var client http.Client
	req, _ := http.NewRequest("PUT", apistr(host, "rooms/"+roomId+
		"/send/m.room.message/"+strconv.FormatInt(time.Now().UnixNano(), 10)+"?", token),
		bytes.NewBuffer(b))
	res, err := client.Do(req)
	body := check(res, err)
	if len(body) == 0 {
		fmt.Println("Unable to send message:", roomId, ":", text)
	}
}

func mkdir(dir string) bool {
	_, stat := os.Stat(dir)
	created := os.IsNotExist(stat)
	if os.IsNotExist(stat) {
		os.MkdirAll(dir, 0700)
	}
	return created
}

func readPipe(pipe string, host string, token string) {
	roomId := path.Base(path.Dir(pipe))
	for {
		str, err := ioutil.ReadFile(pipe)
		if nil != err || len(str) == 0 {
			fmt.Println("Could not read message:", roomId, err)
			continue
		}
		sendMessage(host, token, roomId, string(str))
	}
}

func addfifo(roomPath string, host string, token string) {
	pipe := path.Join(roomPath, "in")
	syscall.Mkfifo(pipe, syscall.S_IFIFO|0600)
	go readPipe(pipe, host, token)
}

func sync(host string, sesh session, accPath string) {
	str := "sync?"
	if sesh.CurrentBatch != "" {
		str += "since=" + sesh.CurrentBatch + "&"
	}
	res, err := http.Get(apistr(host, str, sesh.AccessToken))
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
		roomPath := path.Join(accPath, id)
		if mkdir(roomPath) {
			/* Is there a better way to get the room name/member? */
			addfifo(roomPath, host, sesh.AccessToken)
			var name string
			for _, ev := range room.State.Events {
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
		for i, ev := range room.Timeline.Events {
			if i == len(room.Timeline.Events)-1 {
				pes, per := http.Post(apistr(host, "rooms/"+id+
					"/receipt/m.read/"+ev.EventId+"?", sesh.AccessToken),
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

func logout(host string, token string) {
	res, err := http.Post(apistr(host, "logout?", token), "application/json", nil)
	check(res, err)
}

func login(host string, username string, pass string) session {
	b, _ := json.Marshal(auth{"m.login.password", username, pass})
	res, err := http.Post(host+"/_matrix/client/r0/login", "application/json", bytes.NewBuffer(b))

	body := check(res, err)
	if len(body) == 0 {
		log.Fatalln("Login failed")
	}
	var sesh session
	err = json.Unmarshal(body, &sesh)
	if err != nil || sesh.AccessToken == "" {
		log.Fatalf("Login response not decoded: %s, %s\n", err, body)
	}
	return sesh
}
