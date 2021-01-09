package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

type Client struct {
	Matrix      *mautrix.Client
	Config      *Config
	ConfigFile  string
	AccountRoot string
}

// Initialize configures the matrix client
func (c *Client) Initialize() error {
	if err := c.ParseConfig(); nil != err {
		return errors.Wrap(err, "unable to parse config")
	}

	cli, err := mautrix.NewClient(
		c.Config.Server,
		id.UserID(c.Config.Login.UserID),
		c.Config.Login.AccessToken,
	)
	if nil != err {
		return errors.Wrap(err, "unable to create a new matrix client")
	}
	c.Matrix = cli

	// Read certificate and create the HTTP client
	if err := c.CreateHTTPClient(); nil != err {
		return errors.Wrap(err, "unable to create http client")
	}

	return nil
}

// CreateHTTPClient will read and parse a custom certificate if set,
// and set the HTTP client in the Matrix client
func (c *Client) CreateHTTPClient() error {
	tr := &http.Transport{
		DisableKeepAlives: true,
		IdleConnTimeout:   10 * time.Second,
	}

	/* Self signed certificate */
	if c.Config.Certificate != "" {
		rootPEM, err := ioutil.ReadFile(c.Config.Certificate)
		if nil != err {
			return errors.Wrap(err, "unable to read certificate file")
		}
		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM(rootPEM)
		if !ok {
			return errors.Wrap(err, "failed to parse certificate")
		}
		tr.TLSClientConfig = &tls.Config{
			RootCAs: roots,
		}
	}

	c.Matrix.Client = &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}
	return nil
}

// Login will attempt to use account credentials to get a new access token
// and save the token to disk
func (c *Client) Login() error {
	resp, err := c.Matrix.Login(&mautrix.ReqLogin{
		Type: "m.login.password",
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: c.Config.Username,
		},
		Password:         c.Config.Password,
		StoreCredentials: true,
	})
	if nil != err {
		return errors.Wrap(err, "unable to login")
	}

	c.Config.Login.UserID = resp.UserID.String()
	c.Config.Login.DeviceID = resp.DeviceID.String()
	c.Config.Login.AccessToken = resp.AccessToken
	c.SaveConfig()
	return nil
}
