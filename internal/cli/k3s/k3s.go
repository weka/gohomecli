package cli

import (
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/cli/app"
	"github.com/weka/gohomecli/internal/k3s"
)

func init() {
	// by default homecli is located in /opt/wekahome/{release}/bin
	// and bundle in /opt/wekahome/{release}/
	bundlePath, _ := os.Executable()
	bundlePath = path.Join(path.Dir(bundlePath), "..")

	k3sInstallCmd.Flags().StringVarP(&k3sInstallConfig.Iface, "iface", "i", "", "iterface for k3s network")
	k3sInstallCmd.Flags().StringVarP(&k3sInstallConfig.Hostname, "hostname", "n", k3s.Hostname(), "hostname for cluster")
	k3sInstallCmd.Flags().StringVar(&k3sInstallConfig.NodeIP, "ip", "", "primary IP internal address for wekahome API")
	k3sInstallCmd.Flags().StringSliceVar(&k3sInstallConfig.ExternalIPs, "ips", nil, "additional IP addresses for wekahome API")
	k3sInstallCmd.Flags().StringVar(&k3sInstallConfig.BundlePath, "bundle", bundlePath, "bundle directory with k3s package")

	k3sInstallCmd.MarkFlagRequired("iface")

	k3sUpgradeCmd.Flags().StringVar(&k3sUpgradeConfig.BundlePath, "bundle", bundlePath, "bundle with k3s to install")

	app.AppCmd.AddGroup(&cobra.Group{
		ID:    "k3s",
		Title: "K3S management",
	})

	app.AppCmd.AddCommand(k3sCmd)
	k3sCmd.AddCommand(k3sInstallCmd)
	k3sCmd.AddCommand(k3sUpgradeCmd)
}

var (
	k3sInstallConfig k3s.InstallConfig
	k3sUpgradeConfig k3s.UpgradeConfig
)

var k3sCmd = &cobra.Command{
	Use:     "k3s",
	Short:   "k3s management commands",
	GroupID: "k3s",
}

var k3sInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "install cluster from scratch",
	PreRun: func(cmd *cobra.Command, args []string) {
		ctx, _ := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
		cmd.SetContext(ctx)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return k3s.Install(cmd.Context(), k3sInstallConfig)
	},
}

var k3sUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "upgrade cluster to bundled version",
	PreRun: func(cmd *cobra.Command, args []string) {
		ctx, _ := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
		cmd.SetContext(ctx)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return k3s.Upgrade(cmd.Context(), k3sUpgradeConfig)
	},
}
