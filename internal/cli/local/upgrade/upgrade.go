package upgrade

import (
	"errors"

	"github.com/imdario/mergo"
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	setup_flags "github.com/weka/gohomecli/internal/cli/local/setup/flags"
	"github.com/weka/gohomecli/internal/local/config"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
)

var (
	Cli hooks.Cli
)

type upgrade struct {
	config_v1.Configuration
	setup_flags.Flags
}

func (c upgrade) Validate() error {
	return errors.Join(c.Configuration.Validate(), c.Flags.Validate())
}

var upgradeConfig upgrade

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Local Weka Home",
	Long:  `Upgrade Weka Home with K3S bundle`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		var jsonConfig = config.CLIConfig
		if upgradeConfig.JsonConfig != "" {
			jsonConfig = upgradeConfig.JsonConfig
		}

		// Use cli configuration over config json passed for overwrite
		var cfg config_v1.Configuration

		err = errors.Join(
			config.ReadV1(jsonConfig, &cfg),                // read config into cfg
			mergo.Merge(&upgradeConfig.Configuration, cfg), // merge cfg into upgradeConfig
		)

		if err != nil {
			return err
		}

		if err := readTLS(upgradeConfig.TLSCert, upgradeConfig.TLSKey, &upgradeConfig.Configuration); err != nil {
			return err
		}

		if upgradeConfig.Flags.ProxyURL != "" {
			upgradeConfig.Configuration.Proxy.URL = upgradeConfig.Flags.ProxyURL
		}

		return upgradeConfig.Validate()
	},
	RunE: runUpgrade,
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(upgradeCmd)

		setup_flags.Use(upgradeCmd, &upgradeConfig.Flags)

		upgradeCmd.Flags().StringVar(&upgradeConfig.Host, "host", "", "public host or IP address for LWH (default: interface address)")
		upgradeCmd.Flags().StringVar(&upgradeConfig.IP, "ip", "0.0.0.0", "internal IP address to use for cluster")
	})
}
