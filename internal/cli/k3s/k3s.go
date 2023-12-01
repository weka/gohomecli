package k3s

import (
	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/bundle"
	"github.com/weka/gohomecli/internal/cli/app/options"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/k3s"
	"github.com/weka/gohomecli/internal/utils"
)

var Cli options.Cli

func init() {
	Cli.AddOption(func(appCmd *cobra.Command) {
		appCmd.AddGroup(K3SGroup)
		appCmd.AddCommand(k3sCmd)

		k3sInstallCmd.Flags().StringVarP(&k3sInstallConfig.Iface, "iface", "i", "", "interface for k3s network")
		k3sInstallCmd.Flags().StringVarP(&k3sInstallConfig.Hostname, "hostname", "n", k3s.Hostname(), "hostname for cluster")
		k3sInstallCmd.Flags().StringVar(&k3sInstallConfig.NodeIP, "ip", "", "primary IP internal address for wekahome API")
		k3sInstallCmd.Flags().StringSliceVar(&k3sInstallConfig.ExternalIPs, "ips", nil, "additional IP addresses for wekahome API (e.g public ip)")
		k3sInstallCmd.Flags().StringVar(&k3sInstallConfig.BundlePath, "bundle", bundle.BundlePath(), "bundle directory with k3s package")
		k3sInstallCmd.Flags().BoolVar(&k3sInstallConfig.Debug, "debug", false, "enable debug mode")

		k3sInstallCmd.MarkFlagRequired("iface")
		k3sInstallCmd.Flags().MarkHidden("bundle")
		k3sInstallCmd.Flags().MarkHidden("ip")
		k3sInstallCmd.Flags().MarkHidden("debug")

		k3sUpgradeCmd.Flags().StringVar(&k3sUpgradeConfig.BundlePath, "bundle", bundle.BundlePath(), "bundle with k3s to install")
		k3sUpgradeCmd.Flags().BoolVar(&k3sUpgradeConfig.Debug, "debug", false, "enable debug mode")

		k3sUpgradeCmd.Flags().MarkHidden("bundle")
		k3sUpgradeCmd.Flags().MarkHidden("debug")

		k3sCmd.AddCommand(k3sInstallCmd)
		k3sCmd.AddCommand(k3sUpgradeCmd)
	})
}

var (
	k3sInstallConfig k3s.InstallConfig
	k3sUpgradeConfig k3s.UpgradeConfig
)

var K3SGroup = &cobra.Group{
	ID:    "k3s",
	Title: "K3S management",
}

var k3sCmd = &cobra.Command{
	Use:     "k3s",
	Short:   "k3s management commands",
	GroupID: "k3s",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !env.VerboseLogging {
			utils.SetGlobalLoggingLevel(utils.InfoLevel)
		}
	},
}

var k3sInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "install cluster from scratch",
	RunE: func(cmd *cobra.Command, args []string) error {
		return k3s.Install(cmd.Context(), k3sInstallConfig)
	},
}

var k3sUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "upgrade cluster to bundled version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return k3s.Upgrade(cmd.Context(), k3sUpgradeConfig)
	},
}
