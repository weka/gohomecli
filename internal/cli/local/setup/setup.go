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

var (
	Cli hooks.Cli
)

type setup struct {
	config_v1.Configuration
	setup_flags.Flags

	Iface      string
	JsonConfig string

	TLSCert string // TLS certificate file
	TLSKey  string // TLS key file
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

		return setupConfig.Validate()
	},
	RunE: runSetup,
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(setupCmd)

		setup_flags.Use(setupCmd, &setupConfig.Flags)

		setupCmd.Flags().StringVar(&setupConfig.Host, "host", "", "public host or IP address for LWH (default: interface address)")

		setupCmd.Flags().StringVar(&setupConfig.IP, "ip", "0.0.0.0", "internal IP address to use for cluster")
		setupCmd.Flags().StringVar(&setupConfig.Iface, "iface", "", "interface to use for internal networking")

		setupCmd.Flags().StringVar(&setupConfig.TLSCert, "tls-cert", "", "TLS certificate file")
		setupCmd.Flags().StringVar(&setupConfig.TLSKey, "tls-key", "", "TLS secret key file")
	})
}
