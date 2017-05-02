package main

import "net/url"

func authenticate(address, accessToken string) string {
	u, err := url.Parse(address)
	if err != nil {
		panic(err)
	}

	params := u.Query()
	params.Set("access_token", accessToken)
	u.RawQuery = params.Encode()
	return u.String()
}

