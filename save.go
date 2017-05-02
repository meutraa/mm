package main

import (
	"os"
	"path"
	"strings"
	"strconv"
	"io/ioutil"
	"time"
	"fmt"
	"log"
)

func (e event) Save(file string, host string) {
	/* If this event exists, skip. */
	_, err := os.Stat(file)
	if err == nil {
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

	e.Write(file, s)
	fmt.Println(file)
}

func (e event) Write(file string, body string) {
	if err := os.MkdirAll(path.Dir(file), 0700); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(file, []byte(body+"\n"), 0600); err != nil {
		panic(err)
	}
	t := time.Unix((e.Timestamp/1000)-5, 0)
	if err := os.Chtimes(file, t, t); err != nil {
		log.Println(err)
	}
}
