package setup

import (
	"errors"
	"fmt"

	"github.com/imdario/mergo"
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	setup_flags "github.com/weka/gohomecli/internal/cli/local/setup/flags"
	"github.com/weka/gohomecli/internal/local/config"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/local/k3s"
	"github.com/weka/gohomecli/internal/utils"
)

var (
	Cli hooks.Cli
)

var setupConfig struct {
	config_v1.Configuration

	setup_flags.Flags

	Iface      string
	JsonConfig string
}

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

		if setupConfig.Chart.RemoteVersion != "" && !setupConfig.Chart.RemoteDownload {
			return fmt.Errorf("%w: --remote-version can only be used with --remote-download", utils.ErrValidationFailed)
		}

		// if was specified from args
		if setupConfig.TLS.Cert != "" {
			var b = true
			setupConfig.TLS.Enabled = &b
		}

		return nil
	},
	RunE: runSetup,
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(setupCmd)

		setup_flags.Use(setupCmd, &setupConfig.Flags)
		setupCmd.Flags().StringVarP(&setupConfig.JsonConfig, "json-config", "c", "", "Configuration in JSON format (file or JSON string)")

		setupCmd.Flags().StringVarP(&setupConfig.Iface, "iface", "i", "", "interface for k3s network")
		setupCmd.Flags().StringVarP(&setupConfig.Host, "hostname", "n", k3s.Hostname(), "hostname for cluster")
		setupCmd.Flags().StringVar(&setupConfig.NodeIP, "ip", "", "primary IP internal address for wekahome API")
		setupCmd.Flags().StringSliceVar(&setupConfig.ExternalIPs, "ips", nil, "additional IP addresses for wekahome API (e.g public ip)")
		setupCmd.Flags().StringVar(&setupConfig.TLS.Cert, "cert", "", "TLS certificate")
		setupCmd.Flags().StringVar(&setupConfig.TLS.Key, "key", "", "TLS secret key")

		setupCmd.MarkFlagRequired("iface")
		setupCmd.Flags().MarkHidden("ip")
	})
}
