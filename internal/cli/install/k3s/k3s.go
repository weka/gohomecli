package k3s

import (
	"github.com/weka/gohomecli/internal/install/bundle"
	"github.com/weka/gohomecli/internal/install/k3s"

	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/cli/app/hooks"
)

var Cli hooks.Cli

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(k3sCmd)

		k3sCmd.Flags().StringVarP(&k3sInstallConfig.Iface, "iface", "i", "", "interface for k3s network")
		k3sCmd.Flags().StringVarP(&k3sInstallConfig.Hostname, "hostname", "n", k3s.Hostname(), "hostname for cluster")
		k3sCmd.Flags().StringVar(&k3sInstallConfig.NodeIP, "ip", "", "primary IP internal address for wekahome API")
		k3sCmd.Flags().StringSliceVar(&k3sInstallConfig.ExternalIPs, "ips", nil, "additional IP addresses for wekahome API (e.g public ip)")
		k3sCmd.Flags().StringVar(&k3sInstallConfig.BundlePath, "bundle", bundle.BundlePath(), "bundle directory with k3s package")
		k3sCmd.Flags().BoolVar(&k3sInstallConfig.Debug, "debug", false, "enable debug mode")

		k3sCmd.MarkFlagRequired("iface")
		k3sCmd.Flags().MarkHidden("bundle")
		k3sCmd.Flags().MarkHidden("ip")
		k3sCmd.Flags().MarkHidden("debug")

		k3sUpgradeCmd.Flags().StringVar(&k3sUpgradeConfig.BundlePath, "bundle", bundle.BundlePath(), "bundle with k3s to install")
		k3sUpgradeCmd.Flags().BoolVar(&k3sUpgradeConfig.Debug, "debug", false, "enable debug mode")
		k3sUpgradeCmd.Flags().MarkHidden("bundle")
		k3sUpgradeCmd.Flags().MarkHidden("debug")

		k3sImageImportCmd.Flags().StringVar(&k3sImportConfig.BundlePath, "bundle", bundle.BundlePath(), "bundle with images to load")
		k3sImageImportCmd.Flags().BoolVar(&k3sImportConfig.FailFast, "fail-fast", false, "fail on first error")
		k3sImageImportCmd.Flags().StringSliceVarP(&k3sImportConfig.ImagePaths, "image-path", "f", nil, "images to import (if specified, bundle images are ignored)")
		k3sImageImportCmd.Flags().MarkHidden("bundle")

		k3sCmd.AddCommand(k3sUpgradeCmd)
		k3sCmd.AddCommand(k3sImageImportCmd)
	})
}

var (
	k3sInstallConfig k3s.InstallConfig
	k3sUpgradeConfig k3s.UpgradeConfig
	k3sImportConfig  struct {
		BundlePath string
		ImagePaths []string
		FailFast   bool
	}
)

var k3sCmd = &cobra.Command{
	Use:   "k3s",
	Short: "Install k3s based kubernetes cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		return k3s.Install(cmd.Context(), k3sInstallConfig)
	},
}

var k3sUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade k3s cluster to bundled version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return k3s.Upgrade(cmd.Context(), k3sUpgradeConfig)
	},
}

var k3sImageImportCmd = &cobra.Command{
	Use:   "images",
	Short: "Import docker images from bundle",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(k3sImportConfig.ImagePaths) == 0 {
			return k3s.ImportBundleImages(cmd.Context(), k3sImportConfig.BundlePath, k3sImportConfig.FailFast)
		}

		return k3s.ImportImages(cmd.Context(), k3sImportConfig.ImagePaths, k3sImportConfig.FailFast)
	},
}
