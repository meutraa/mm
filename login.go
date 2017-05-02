package main

type LoginResponse struct {
	AccessToken        string `json:"access_token"`
	HomeServer   string `json:"home_server"`
	UserId       string `json:"user_id"`
}

type LoginRequest struct {
	Type string `json:"type"`
	User string `json:"user"`
	Password string `json:"password"`
}

var (
	typePassword = "m.login.password"
	loginAddress = "/_matrix/client/r0/login"
)

func login(host, user, pass string) (credentials LoginResponse) {
	res := PostJSON(host + loginAddress, LoginRequest{typePassword, user, pass})
	if 200 != res.StatusCode {
		panic(res.Status)
	}

	ReadJSON(res, &credentials)
	return
}

