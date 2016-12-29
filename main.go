package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
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

var client = &http.Client{}

func main() {
	usr, _ := user.Current()
	var server, username, pass, accPath, cert string
	flag.StringVar(&server, "s", "", "homeserver url <https://matrix.org>")
	flag.StringVar(&username, "u", "", "not full qualified username <bob>")
	flag.StringVar(&pass, "p", "", "password <pass1234>")
	flag.StringVar(&accPath, "d", usr.HomeDir+"/mm", "directory path")
	flag.StringVar(&cert, "c", "", "certificate path")
	flag.Parse()

	host, err := url.Parse(server)
	if server == "" || username == "" || pass == "" || err != nil {
		flag.PrintDefaults()
		os.Exit(2)
	}
	if host.Scheme == "" {
		host.Scheme = "https"
	}

	/* Self signed certificate */
	rootPEM, _ := ioutil.ReadFile(cert)
	if string(rootPEM) != "" {
		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM(rootPEM)
		if ok {
			client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots}}}
		} else {
			log.Println("failed to parse certificate:", cert)
		}
	}

	/* Account login and setup. */
	var sesh session
	/* Empty get request to give time to ACK http2 settings. */
	client.Get(host.String())
	b, _ := json.Marshal(auth{"m.login.password", username, pass})
	resp, err := client.Post(host.String()+"/_matrix/client/r0/login", "application/json", bytes.NewBuffer(b))
	body, _ := readBody(resp, err)
	json.Unmarshal(body, &sesh)
	if sesh.Token == "" {
		os.Exit(1)
	}

	/* Revoke access_token on exit. */
	defer client.Post(host.String()+"/_matrix/client/r0/logout?access_token="+sesh.Token, "application/json", nil)

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
	for {
		sesh.CurrentBatch = sync(host.String(), sesh, accPath)
	}
}

func readPipe(pipe string, host string, token string) {
	roomID := path.Base(path.Dir(pipe))
	for {
		str, err := ioutil.ReadFile(pipe)
		if err != nil {
			log.Println("Could not read message:", pipe, roomID, err)
			continue
		}

		/* Send a message. */
		data, _ := json.Marshal(message{string(str), "m.text"})
		url := host + "/_matrix/client/r0/" + "rooms/" + roomID + "/send/m.room.message/" + strconv.FormatInt(time.Now().UnixNano(), 10) + "?" + "access_token=" + token
		req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(data))
		readBody(client.Do(req))
	}
}

func sync(host string, sesh session, accPath string) string {
	url := host + "/_matrix/client/r0/sync?"
	if sesh.CurrentBatch != "" {
		url += "since=" + sesh.CurrentBatch + "&timeout=30000&"
	}
	url += "access_token=" + sesh.Token

	body, err := readBody(client.Get(url))
	if err != nil {
		time.Sleep(time.Second * 10)
		return sesh.CurrentBatch
	}

	var d data
	if json.Unmarshal(body, &d) != nil || d.NextBatch == "" {
		return sesh.CurrentBatch
	}

	for roomID, room := range d.Rooms.Join {
		roomPath := path.Join(accPath, roomID)
		os.Mkdir(roomPath, 0700)

		pipe := path.Join(roomPath, "in")
		_, stat := os.Stat(pipe)
		if os.IsNotExist(stat) {
			syscall.Mkfifo(pipe, syscall.S_IFIFO|0600)
			go readPipe(pipe, host, sesh.Token)
		}

		var lastID string
		for _, e := range room.Timeline.Events {
			lastID = e.EventId
			if e.Type != "m.room.message" {
				continue
			}
			file := path.Join(roomPath, e.Sender, e.EventId)

			saveEvent(file, e, host)
		}
		/* Send a read receipt. */
		if lastID != "" {
			url := host + "/_matrix/client/r0/rooms/" + roomID + "/receipt/m.read/" + lastID + "?access_token=" + sesh.Token
			readBody(client.Post(url, "application/json", nil))
		}
	}
	return d.NextBatch
}

func saveEvent(file string, e event, host string) {
	os.Mkdir(path.Dir(file), 0700)
	_, stat := os.Stat(file)
	if os.IsExist(stat) {
		return
	}
	s := e.Content.Body
	switch e.Content.Type {
	case "m.image", "m.video", "m.file", "m.audio":
		f := e.Content.FileInfo
		s = host + "/_matrix/media/r0/download/" + strings.TrimPrefix(e.Content.Url, "mxc://") + " (" + f.MimeType
		if e.Content.Type == "m.image" || e.Content.Type == "m.video" {
			s += " " + strconv.Itoa(f.Height) + "x" + strconv.Itoa(f.Width)
		}
		s += " " + strconv.Itoa(f.Size>>10) + "KiB)"
	case "m.location":
		s += " " + e.Content.GeoUri
	}
	ioutil.WriteFile(file, []byte(s+"\n"), 0644)

	t := time.Unix((e.Timestamp/1000)-5, 0)
	os.Chtimes(file, t, t)
	fmt.Println(file)
}

func readBody(res *http.Response, err error) ([]byte, error) {
	if err != nil {
		log.Println(err)
		return []byte(""), err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil || res.StatusCode != 200 {
		log.Println(err, res.StatusCode, http.StatusText(res.StatusCode))
	}
	return body, err
}
