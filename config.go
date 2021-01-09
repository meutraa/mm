package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Server      string
	Username    string
	Password    string
	Certificate string
	Directory   string
	Login       struct {
		UserID      string
		DeviceID    string
		AccessToken string
	}
}

func xdgConfigDir(home string) string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if "" == dir {
		dir = path.Join(home, ".config", "mm")
	} else {
		dir = path.Join(dir, "mm")
	}
	return dir
}

func xdgDataDir(home string) string {
	dir := os.Getenv("XDG_DATA_HOME")
	if "" == dir {
		dir = path.Join(home, ".local", "share", "mm")
	} else {
		dir = path.Join(dir, "mm")
	}
	return dir
}

// ParseConfig will create or read
func (c *Client) ParseConfig() error {
	usr, err := user.Current()
	if nil != err {
		return errors.Wrap(err, "unable to get current user")
	}

	dataDir := xdgDataDir(usr.HomeDir)
	configPath := xdgConfigDir(usr.HomeDir)

	var configFile string
	flag.StringVar(&configFile, "c", path.Join(configPath, "config.yml"), "mm configuration file")
	flag.Parse()

	c.ConfigFile = configFile
	c.Config = &Config{
		Server:    "https://matrix.org",
		Username:  usr.Username,
		Directory: dataDir,
	}

	// Check if the file exists or not
	info, err := os.Stat(configFile)
	if nil != err {
		if os.IsNotExist(err) {
			// Try to create the config file with default values
			if err := c.SaveConfig(); nil != err {
				return errors.Wrap(err, "unable to save default config")
			}
			return errors.New("default config file written to " + configFile + "\nupdate file and rerun")
		}
		return errors.Wrap(err, "unable to stat config file")
	}

	if info.IsDir() {
		return errors.New("config file is a directory, not a yaml file")
	}

	d, err := ioutil.ReadFile(configFile)
	if nil != err {
		return errors.Wrap(err, "unable to read default config file")
	}
	if err := yaml.Unmarshal(d, &c.Config); nil != err {
		return errors.Wrap(err, "unable to unmarshal config file")
	}

	return nil
}

func (c *Client) SaveConfig() error {
	d, err := yaml.Marshal(c.Config)
	if nil != err {
		return errors.Wrap(err, "unable to marshal default config")
	}

	if err := ensureDir(c.ConfigFile); nil != err {
		return err
	}

	if err := ioutil.WriteFile(c.ConfigFile, d, 0600); nil != err {
		return errors.Wrap(err, "unable to write default config file")
	}
	return nil
}
