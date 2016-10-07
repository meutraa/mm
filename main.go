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

var session Session
var host, user, pass string

type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	DeviceId     string `json:"device_id"`
}

func apistr(str string) string {
	return host + "/_matrix/client/r0/" + str + "access_token=" + session.AccessToken
}

func main() {
	flag.StringVar(&host, "host", "", "matrix homeserver")
	flag.StringVar(&user, "user", "", "username")
	flag.StringVar(&pass, "pass", "", "password")
	flag.Parse()

	login()
	sync()
	logout()
}

func sync() {
	res, _ := http.Get(apistr("sync?"))
	body, _ := ioutil.ReadAll(res.Body)
	fmt.Printf("%s\n", body)
	res.Body.Close()
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
		bytes.NewBuffer([]byte("{\"type\":\"m.login.password\",\"user\":\""+user+"\",\"password\":\""+pass+"\"}")))
	if err != nil || res.StatusCode != 200 {
		log.Fatalf("Login failed: %s, %s\n", err, http.StatusText(res.StatusCode))
	}

	fmt.Println("The user has been authenticated")

	body, _ := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if json.Unmarshal(body, &session) != nil || session.AccessToken == "" {
		log.Fatalf("Login response could not be parsed: %s,%s\n", err, body)
	}
}
