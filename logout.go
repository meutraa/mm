package main

import (
	"log"
)

var logoutAddress = "/_matrix/client/r0/logout"

func logout(host, accessToken string) {
	log.Println("revoking access token:", accessToken)

	path := authenticate(host + logoutAddress, accessToken)
	res := PostJSON(path, "")
	if 200 != res.StatusCode {
		panic(res.Status)
	}
}

