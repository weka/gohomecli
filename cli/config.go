package cli

import (
	"io/ioutil"
	"os"
	"os/user"

	toml "github.com/pelletier/go-toml"
)

// ConfigFilePath is the full path of the CLI configuration file (TOML)
var ConfigFilePath string

func init() {
	currentUser, e := user.Current()
	if e != nil {
		panic(e)
	}
	ConfigFilePath = currentUser.HomeDir + "/.config/home-cli/config.toml"
}

// Config holds all global CLI configuration values
type Config struct {
	APIKey   string `toml:"api_key"`
	CloudURL string `toml:"cloud_url"`
}

// ReadCLIConfig reads the configuration file and returns a Config instance
func ReadCLIConfig() *Config {
	data, e := ioutil.ReadFile(ConfigFilePath)
	if e != nil {
		panic(e)
	}
	config := &Config{}
	toml.Unmarshal(data, config)
	return config
}

// WriteCLIConfig writes the given Config instance to the configuration file
func WriteCLIConfig(config *Config) {
	data, e := toml.Marshal(*config)
	if e != nil {
		panic(e)
	}
	e = ioutil.WriteFile(ConfigFilePath, data, os.ModeExclusive)
	if e != nil {
		panic(e)
	}
}
