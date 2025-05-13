package setup

import (
	"errors"

	"github.com/imdario/mergo"
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	setup_flags "github.com/weka/gohomecli/internal/cli/local/setup/flags"
	"github.com/weka/gohomecli/internal/local/config"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
)

var Cli hooks.Cli

type setup struct {
	setup_flags.Flags
	config_v1.Configuration
}

func (s setup) Validate() error {
	return errors.Join(s.Configuration.Validate(), s.Flags.Validate())
}

var setupConfig setup

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install Local Weka Home",
	Long:  `Install Weka Home Helm chart with K3S bundle`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if setupConfig.JsonConfig != "" {
			// Use cli configuration over config json passed for overwrite
			var c config_v1.Configuration

			err := errors.Join(
				config.ReadV1(setupConfig.JsonConfig, &c),
				mergo.Merge(&setupConfig.Configuration, c),
			)
			if err != nil {
				return err
			}
		}

		if err := readTLS(setupConfig.TLSCert, setupConfig.TLSKey, &setupConfig.Configuration); err != nil {
			return err
		}

		if setupConfig.ProxyURL != "" {
			setupConfig.Proxy.URL = setupConfig.ProxyURL
		}

		return setupConfig.Validate()
	},
	RunE: runSetup,
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(setupCmd)

		setup_flags.Use(setupCmd, &setupConfig.Flags)

		setupCmd.Flags().
			StringVar(&setupConfig.Host, "host", "", "public host or IP address for LWH (default: interface address)")
		setupCmd.Flags().StringVar(&setupConfig.IPv4, "ip", "0.0.0.0", "internal IP address to use for cluster")
		setupCmd.Flags().StringVar(&setupConfig.IPv6, "ip6", "::", "IP6 address to use for cluster")
	})
}
