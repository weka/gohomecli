package upgrade

import (
	"errors"

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
		var c = config.CLIConfig
		if upgradeConfig.JsonConfig != "" {
			c = upgradeConfig.JsonConfig
		}
		if err := config.ReadV1(c, &upgradeConfig.Configuration); err != nil {
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
	})
}
