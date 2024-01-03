package upgrade

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	setup_flags "github.com/weka/gohomecli/internal/cli/local/setup/flags"
	"github.com/weka/gohomecli/internal/local/config"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
)

var (
	Cli hooks.Cli
)

var upgradeConfig struct {
	config_v1.Configuration

	setup_flags.Flags
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Local Weka Home",
	Long:  `Upgrade Weka Home with K3S bundle`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		if err := upgradeConfig.Validate(); err != nil {
			return err
		}
		return config.ReadV1(config.LHWConfig, &upgradeConfig.Configuration)
	},
	RunE: runUpgrade,
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(upgradeCmd)

		setup_flags.Use(upgradeCmd, &upgradeConfig.Flags)
	})
}
