package k3s

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/bundle"
	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/k3s"
	"github.com/weka/gohomecli/internal/utils"
)

var Cli hooks.Cli

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
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

		k3sImageImportCmd.Flags().StringVar(&k3sImportConfig.BundlePath, "bundle", bundle.BundlePath(), "bundle with images to load")
		k3sImageImportCmd.Flags().BoolVar(&k3sImportConfig.FailFast, "fail-fast", false, "fail on first error")
		k3sImageImportCmd.Flags().BoolVar(&k3sImportConfig.FromBundle, "from-bundle", false, "import images from bundle")
		k3sImageImportCmd.Flags().StringSliceVarP(&k3sImportConfig.ImagePaths, "image-path", "f", nil, "images to import")
		k3sImageImportCmd.Flags().MarkHidden("bundle")
		k3sImageImportCmd.MarkFlagsMutuallyExclusive("from-bundle", "image-path")

		k3sCmd.AddCommand(k3sInstallCmd)
		k3sCmd.AddCommand(k3sUpgradeCmd)
		k3sCmd.AddCommand(k3sImageImportCmd)
	})
}

var (
	k3sInstallConfig k3s.InstallConfig
	k3sUpgradeConfig k3s.UpgradeConfig
	k3sImportConfig  struct {
		BundlePath string
		FromBundle bool
		ImagePaths []string
		FailFast   bool
	}
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

var k3sImageImportCmd = &cobra.Command{
	Use:   "import-images",
	Short: "import images from bundle",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !k3sImportConfig.FromBundle && len(k3sImportConfig.ImagePaths) == 0 {
			return fmt.Errorf("%w: either --from-bundle or --image-path must be specified", utils.ErrValidationFailed)
		}

		if k3sImportConfig.FromBundle {
			return k3s.ImportBundleImages(cmd.Context(), k3sImportConfig.BundlePath, k3sImportConfig.FailFast)
		}

		return k3s.ImportImages(cmd.Context(), k3sImportConfig.ImagePaths, k3sImportConfig.FailFast)
	},
}
