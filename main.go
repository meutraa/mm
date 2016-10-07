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

type LoginResponse struct {
	ErrorCode    string `json:"errcode"`
	Error        string `json:"error"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	DeviceId     string `json:"device_id"`
}

func main() {
	var host, user, pass string
	flag.StringVar(&host, "host", "", "matrix homeserver")
	flag.StringVar(&user, "user", "", "username")
	flag.StringVar(&pass, "pass", "", "password")
	flag.Parse()

	session := login(host, user, pass)
	fmt.Printf("%s\n", session.AccessToken)
}

func login(host string, user string, pass string) LoginResponse {
	login := Login{"m.login.password", user, pass}
	b, err := json.Marshal(login)
	buf := bytes.NewBuffer(b)

	res, err := http.Post(host+"/_matrix/client/r0/login", "application/json", buf)
	if err != nil {
		log.Fatalf("Login POST request to server failed: %s", err)
	}
	switch res.StatusCode {
	case 200:
		fmt.Println("The user has been authenticated")
	case 400:
		log.Fatalf("Part of the request was invalid. For example, the login type may not be recognised.")
	case 403:
		log.Fatalf("The login attempt failed. For example, the password may have been incorrect.")
	case 429:
		log.Fatalf("This request was rate-limited.")
	default:
		log.Fatalf("HTTP response: %s", http.StatusText(res.StatusCode))
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Login POST response could not read Body: %s", err)
	}
	res.Body.Close()

	var loginRes LoginResponse
	err = json.Unmarshal(body, &loginRes)
	if err != nil {
		log.Fatalf("Login POST response body could not unmarshal: %s", err)
	}
	return loginRes
}
