package cli

import (
	"io/ioutil"
	"os"
	"os/user"

	"github.com/weka/gohomecli/cli/logging"

	"github.com/pelletier/go-toml"
)

var logger = logging.GetLogger("Config")

// ConfigFilePath is the full path of the CLI configuration file (TOML)
var ConfigFilePath string

var initialized = false
var CurrentConfig *Config
var CurrentSiteConfig *SiteConfig
var SiteName string

func init() {
	currentUser, e := user.Current()
	if e != nil {
		panic(e)
	}
	ConfigFilePath = currentUser.HomeDir + "/.config/home-cli/config.toml"
}

func InitConfig(siteNameFromCommandLine string) {
	if initialized {
		return
	}
	CurrentConfig = ReadCLIConfig()
	CurrentSiteConfig, SiteName = GetSiteConfig(CurrentConfig, siteNameFromCommandLine)
	initialized = true
	logger.Debug().Str("site", SiteName).Str("url", CurrentSiteConfig.CloudURL).
		Msg("Site configuration loaded")
}

// SiteConfig holds configuration values for a specific Weka Home site
type SiteConfig struct {
	APIKey   string `toml:"api_key"`
	CloudURL string `toml:"cloud_url"`
}

// Config holds all global CLI configuration values
type Config struct {
	APIKey      string                `toml:"api_key"`
	CloudURL    string                `toml:"cloud_url"`
	DefaultSite string                `toml:"default_site"`
	Sites       map[string]SiteConfig `toml:"sites"`
}

// ReadCLIConfig reads the configuration file and returns a Config instance
func ReadCLIConfig() *Config {
	logger.Debug().Str("file", ConfigFilePath).Msg("Reading configuration")
	data, e := ioutil.ReadFile(ConfigFilePath)
	if e != nil {
		logger.Fatal().
			Str("file", ConfigFilePath).
			Err(e).
			Msg("Failed to read config file")
	}
	config := &Config{}
	e = toml.Unmarshal(data, config)
	if e != nil {
		logger.Fatal().
			Str("file", ConfigFilePath).
			Err(e).
			Msg("Failed to parse config file contents")
	}
	return config
}

// WriteCLIConfig writes the given Config instance to the configuration file
func WriteCLIConfig(config *Config) {
	logger.Debug().Str("file", ConfigFilePath).Msg("Writing configuration")
	data, e := toml.Marshal(*config)
	if e != nil {
		panic(e)
	}
	e = ioutil.WriteFile(ConfigFilePath, data, os.ModeExclusive)
	if e != nil {
		panic(e)
	}
}

// GetSiteConfig returns configuration values, and site name, for a specific Weka Home site
func GetSiteConfig(config *Config, siteNameFromCommandLine string) (*SiteConfig, string) {
	siteName := siteNameFromCommandLine
	var siteConfig SiteConfig
	if siteName == "" && (config.APIKey != "" || config.CloudURL != "") {
		// Simple configuration with one site (also backwards compatible with old home-cli config)
		siteName = "<unspecified>"
		siteConfig = SiteConfig{APIKey: config.APIKey, CloudURL: config.CloudURL}
		if config.DefaultSite != "" {
			logger.Warn().Msg(
				"Config warning: \"default_site\" is set, but so are the global \"api_key\" and \"cloud_url\"")
		}
	} else {
		// Normal configuration file, with site configurations
		if len(config.Sites) == 0 {
			logger.Fatal().Msg("Config error: no sites are configured")
		}
		if siteName == "" {
			if config.DefaultSite == "" {
				logger.Fatal().Msg(
					"Config error: --site was not specified, and \"default_site\" is not set in the config file")
			}
			siteName = config.DefaultSite
		}
		var exists bool
		siteConfig, exists = config.Sites[siteName]
		if !exists {
			logger.Fatal().Msgf(
				"Config error: default site %s has no corresponding [site.%s] configuration in the config file",
				siteName, siteName)
		}
	}
	validateSiteConfig(&siteConfig, siteName)
	return &siteConfig, siteName
}

func validateSiteConfig(siteConfig *SiteConfig, siteName string) {
	if siteConfig.APIKey == "" {
		logger.Fatal().Msgf("Config error: \"api_key\" is unset for site %s", siteName)
	}
	if siteConfig.CloudURL == "" {
		logger.Fatal().Msgf("Config error: \"cloud_url\" is unset for site %s", siteName)
	}
}
