package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path"
	"syscall"
	"time"
)

//var client *http.Client
var currentBatch string

func client() *http.Client {
	tr := &http.Transport{DisableKeepAlives: true}
	c := &http.Client{Transport: tr, Timeout: time.Second * 20}
	return c
}

func main() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	var server, username, pass, accPath, cert string
	flag.StringVar(&server, "s", "", "homeserver url <https://matrix.org>")
	flag.StringVar(&username, "u", usr.Username, "not full qualified username <bob>")
	flag.StringVar(&pass, "p", "", "password <pass1234>")
	flag.StringVar(&accPath, "d", usr.HomeDir+"/.mm", "directory path")
	flag.StringVar(&cert, "c", "", "certificate path")
	flag.Parse()

	if server == "" || pass == "" {
		flag.PrintDefaults()
		os.Exit(2)
	}

	/* Self signed certificate */
	/*if "" != cert {
		rootPEM, err := ioutil.ReadFile(cert)
		if err != nil {
			log.Println(err)
		} else if len(rootPEM) > 0 {
			roots := x509.NewCertPool()
			if roots.AppendCertsFromPEM(rootPEM) {
				client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots}}}
			} else {
				log.Println("failed to parse certificate:", cert)
			}
		}
	}*/

	session := login(server, username, pass)

	/* Logout on interrupt signal. */
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	go func() {
		for _ = range c {
			logout(server, session.AccessToken)
			os.Exit(1)
		}
	}()

	accPath = path.Join(accPath, session.HomeServer, session.UserId)
	os.MkdirAll(accPath, 0700)

	/* Start reading existing pipes for sending. */
	watchPipes(accPath, server, session.AccessToken)

	for {
		//sendBufferedMessages(server, session.AccessToken)
		synchronize(server, session, accPath)
	}
}
