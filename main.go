package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Login struct {
	Type     string `json:"type"`
	User     string `json:"user"`
	Password string `json:"password"`
}

var session Session

type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	DeviceId     string `json:"device_id"`
	TokenId      int
}

func main() {
	var host, user, pass string
	flag.StringVar(&host, "host", "", "matrix homeserver")
	flag.StringVar(&user, "user", "", "username")
	flag.StringVar(&pass, "pass", "", "password")
	flag.Parse()

	session = login(host, user, pass)
	fmt.Printf("%s\n", session.AccessToken)
	sync(host)

	logout(host)
}

func sync(host string) {
	res, _ := http.Get(host + "/_matrix/client/r0/sync" + "?access_token=" + session.AccessToken)
	body, _ := ioutil.ReadAll(res.Body)
	fmt.Printf("%s\n", body)
	res.Body.Close()
}

func logout(host string) {
	res, _ := http.Post(host+"/_matrix/client/r0/logout"+"?access_token="+session.AccessToken, "application/json", nil)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		body, _ := ioutil.ReadAll(res.Body)
		fmt.Printf("Logout unsuccessful: %s\n", body)
	}
}

func login(host string, user string, pass string) Session {
	b, _ := json.Marshal(Login{"m.login.password", user, pass})

	res, err := http.Post(host+"/_matrix/client/r0/login", "application/json", bytes.NewBuffer(b))
	if err != nil || res.StatusCode != 200 {
		log.Fatalf("Login request to server failed: %s, %s\n", err, http.StatusText(res.StatusCode))
	}

	fmt.Println("The user has been authenticated")

	body, _ := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	var ses Session
	err = json.Unmarshal(body, &ses)
	if err != nil || ses.AccessToken == "" {
		log.Fatalf("Login response could not be parsed: %s,%s\n", err, body)
	}
	return ses
}
