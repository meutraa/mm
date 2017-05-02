package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

var mimeJson = "application/json"

func PostJSON(address string, data interface{}) *http.Response {
	out, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	res, err := client().Post(address, mimeJson, bytes.NewBuffer(out))
	if nil != err {
		panic(err)
	}
	return res
}

func PutJSON(address string, data interface{}) *http.Response {
	out, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("PUT", address, bytes.NewBuffer(out))
	if err != nil {
		panic(err)
	}

	res, err := client().Do(req)
	if err != nil {
		panic(err)
	}
	return res
}

func ReadJSON(r *http.Response, data interface{}) {
	body, err := ioutil.ReadAll(r.Body)
	if nil != body {
		defer r.Body.Close()
	}
	if nil != err {
		panic(err)
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		panic(err)
	}
}
