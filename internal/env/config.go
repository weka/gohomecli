package env

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	"github.com/weka/gohomecli/internal/utils"

	"github.com/pelletier/go-toml"
)

var logger = utils.GetLogger("Config")

const (
	UnspecifiedSite = "default"
	DefaultSiteName = "default"
	DefaultCloudURL = "https://api.home.weka.io/"
)

var (
	ConfigDir         string
	ConfigFilePath    string
	AliasesFilePath   string
	initialized       = false
	CurrentConfig     *Config
	CurrentSiteConfig *SiteConfig
	SiteName          string
)

func init() {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get user home directory: %s", err)
		os.Exit(1)
	}
	ConfigDir = currentUser.HomeDir + "/.config/home-cli/"
	ConfigFilePath = ConfigDir + "config.toml"
	AliasesFilePath = ConfigDir + "aliases.toml"
}

// SiteConfig holds configuration values for a specific Weka Home site
type SiteConfig struct {
	APIKey   string `toml:"api_key"`
	CloudURL string `toml:"cloud_url"`
}

// Config holds all global CLI configuration values
type Config struct {
	APIKey      string                 `toml:"api_key,omitempty"`
	CloudURL    string                 `toml:"cloud_url,omitempty"`
	DefaultSite string                 `toml:"default_site"`
	Sites       map[string]*SiteConfig `toml:"sites"`
}

func InitConfig(siteNameFromCommandLine string) {
	if initialized {
		return
	}
	if _, err := os.Stat(ConfigDir); os.IsNotExist(err) {
		logger.Warn().
			Str("directory", ConfigDir).
			Str("file", ConfigFilePath).
			Msg("Config directory does not exist, creating directory and default config file")
		createDefaultConfigFileAndExit(true)
	}
	if _, err := os.Stat(ConfigFilePath); os.IsNotExist(err) {
		logger.Warn().
			Str("file", ConfigFilePath).
			Msg("Config file does not exist, creating default config file")
		createDefaultConfigFileAndExit(false)
	}
	CurrentConfig = readCLIConfig()
	CurrentSiteConfig, SiteName = getSiteConfig(CurrentConfig, siteNameFromCommandLine)
	initialized = true
	logger.Debug().Str("site", SiteName).Str("url", CurrentSiteConfig.CloudURL).
		Msg("Site configuration loaded")
}

func createDefaultConfigFileAndExit(createDir bool) {
	if createDir {
		os.MkdirAll(ConfigDir, os.ModePerm)
	}
	writeCLIConfig(&Config{
		DefaultSite: DefaultSiteName,
		Sites: map[string]*SiteConfig{
			DefaultSiteName: &SiteConfig{
				CloudURL: DefaultCloudURL,
			},
		},
	})
	utils.UserWarning(
		"config file not found.\n\n"+
			"Default config file created, please run:\n"+
			"  %s config api-key <your-api-key>\n\n"+
			"Currently using %s. To use another site, please also run:\n"+
			"  %s config cloud-url <url>\n",
		os.Args[0], DefaultCloudURL, os.Args[0])
	os.Exit(1)
}

func readCLIConfig() *Config {
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

func writeCLIConfig(config *Config) {
	logger.Debug().Str("file", ConfigFilePath).Msg("Writing configuration")
	data, e := toml.Marshal(*config)
	if e != nil {
		logger.Panic().Err(e).Msg("Failed to marshal configuration values to TOML format")
	}

	e = ioutil.WriteFile(ConfigFilePath, data, os.ModeExclusive|0644)
	if e != nil {
		logger.Fatal().
			Str("file", ConfigFilePath).
			Err(e).
			Msg("Failed to write config file")
	}
}

func getSiteConfig(config *Config, siteNameFromCommandLine string) (*SiteConfig, string) {
	siteName := siteNameFromCommandLine
	var siteConfig *SiteConfig
	if siteName == "" && (config.APIKey != "" || config.CloudURL != "") {
		// Simple configuration with one site (also backwards compatible with old home-cli config)
		siteName = UnspecifiedSite
		siteConfig = &SiteConfig{APIKey: config.APIKey, CloudURL: config.CloudURL}
		if config.DefaultSite != "" {
			logger.Warn().Msg(
				"Config warning: \"default_site\" is set, but so are the global \"api_key\" and \"cloud_url\"")
		}
	} else {
		// Normal configuration file, with site configurations
		if len(config.Sites) == 0 {
			utils.UserError("Config error: no sites are configured")
		}
		if siteName == "" {
			if config.DefaultSite == "" {
				utils.UserError(
					"Config error: --site was not specified, and \"default_site\" is not set in the config file")
			}
			siteName = config.DefaultSite
		}
		var exists bool
		siteConfig, exists = config.Sites[siteName]
		if !exists {
			utils.UserError(
				"Config error: default site %s has no corresponding [site.%s] configuration in the config file",
				siteName, siteName)
		}
	}
	validateSiteConfig(siteConfig, siteName)
	return siteConfig, siteName
}

func validateSiteConfig(siteConfig *SiteConfig, siteName string) {
	if siteConfig.APIKey == "" {
		utils.UserWarning("config error: \"api_key\" is unset for site %s", siteName)
	}
	if siteConfig.CloudURL == "" {
		utils.UserWarning("config error: \"cloud_url\" is unset for site %s", siteName)
	}
}

// UpdateConfig updates values in the configuration file. To update values specifically
// for the currently active site, use UpdateSiteConfig instead.
func UpdateConfig(update func(config *Config, siteConfig *SiteConfig) error) {
	err := update(CurrentConfig, CurrentSiteConfig)
	if err != nil {
		utils.UserError("failed to update config: " + err.Error())
	}
	writeCLIConfig(CurrentConfig)
}
