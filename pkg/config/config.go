package config

import (
	"flag"
	"io/ioutil"
	"os"
	"os/user"

	"github.com/adrg/xdg"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	configFileName = "mm/config.yml"
)

// Config is all the state we need
type Config struct {
	file string
	Server      string
	Username    string
	Password    string
	Certificate string
	Directory   string
	Login       struct {
		UserID      string
		DeviceID    string
		AccessToken string
		// PrivateKey  []byte
		// PublicKey   []byte
	}
}

// Load will read from disk, or create on disk if it does not exist
func Load() (*Config, error) {
	usr, err := user.Current()
	if nil != err {
		return nil, errors.Wrap(err, "unable to get current user")
	}

	configPath, err := xdg.ConfigFile(configFileName)
	if nil != err {
		return nil, errors.Wrap(err, "unable to create config file")
	}

	var configFile string
	flag.StringVar(&configFile, "c", configPath, "mm configuration file")
	flag.Parse()

	cfg := &Config{
		file: configFile,
		Server:    "https://matrix.org",
		Username:  usr.Username,
		Directory: xdg.DataHome,
	}

	// Check if the file exists or not
	if _, err = os.Stat(cfg.file); nil != err {
		if !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "unable to stat config file")
		}

			// Try to create the config file with default values
			if err := cfg.Save(); nil != err {
				return nil, errors.Wrap(err, "unable to save default config")
			}
			return nil, errors.New("default config file written to " + cfg.file + "\nupdate file and rerun")
	}

	b, err := ioutil.ReadFile(cfg.file)
	if nil != err {
		return nil, errors.Wrap(err, "unable to read default config file")
	}
	if err := yaml.Unmarshal(b, cfg); nil != err {
		return nil, errors.Wrap(err, "unable to unmarshal config file")
	}

	return cfg, nil
}

// Save will save the in memory config to disk
func (c *Config) Save() error {
	b, err := yaml.Marshal(c)
	if nil != err {
		return errors.Wrap(err, "unable to marshal config")
	}

	return ioutil.WriteFile(c.file, b, 0600)
}
