package main

import (
        "net/http"
        "encoding/json"
        "bytes"
        "io/ioutil"
	"time"
)

var mimeJson = "application/json"

func PostJSON(address string, data interface{}) *http.Response {
        out, err := json.Marshal(data)
        if err != nil {
                panic(err)
        }

	client := http.Client{}
	client.Timeout = time.Second * 15
        res, err := client.Post(address, mimeJson, bytes.NewBuffer(out))
        if err != nil {
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

	client := http.Client{}
	client.Timeout = time.Second * 15
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	return res
}

func ReadJSON(r *http.Response, data interface{}) {
        body, err := ioutil.ReadAll(r.Body)
        defer r.Body.Close()
        if err != nil {
                panic(err)
        }

        err = json.Unmarshal(body, data)
        if err != nil {
                panic(err)
        }
}
