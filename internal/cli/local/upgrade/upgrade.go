package upgrade

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/local/bundle"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/local/web"
)

var (
	Cli hooks.Cli
)

var upgradeConfig struct {
	config_v1.Configuration

	JsonConfig string
	ValuesFile string

	Web         bool
	WebBindAddr string
	BundlePath  string
	Images      struct {
		ImportPaths []string
	}

	Chart struct {
		kubeConfigPath string
		localChart     string
		remoteDownload bool
		remoteVersion  string
	}
	Debug bool
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Local Weka Home",
	Long:  `Upgrade Weka Home with K3S bundle`,
	RunE:  runUpgrade,
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(upgradeCmd)

		upgradeCmd.Flags().StringVarP(&upgradeConfig.JsonConfig, "json-config", "c", "", "Configuration in JSON format (file or JSON string)")
		upgradeCmd.Flags().StringVar(&upgradeConfig.ValuesFile, "values", "", "Path to values.yaml (optional)")
		upgradeCmd.MarkFlagsMutuallyExclusive("json-config", "values")

		if web.IsEnabled() {
			upgradeCmd.Flags().BoolVar(&upgradeConfig.Web, "web", false, "start web installer")
			upgradeCmd.Flags().StringVarP(&upgradeConfig.WebBindAddr, "bind-addr", "b", ":8080", "Bind address for web server including port")
		}

		upgradeCmd.Flags().StringVar(&upgradeConfig.BundlePath, "bundle", bundle.BundlePath(), "bundle directory with k3s package")
		upgradeCmd.Flags().BoolVar(&upgradeConfig.Debug, "debug", false, "enable debug mode")

		upgradeCmd.Flags().MarkHidden("bundle")
		upgradeCmd.Flags().MarkHidden("debug")

		upgradeCmd.Flags().StringSliceVarP(&upgradeConfig.Images.ImportPaths, "image-path", "f", nil, "images to import (if specified, bundle images are ignored)")

		upgradeCmd.Flags().StringVarP(&upgradeConfig.Chart.kubeConfigPath, "kube-config", "k", "/etc/rancher/k3s/k3s.yaml", "Path to kubeconfig file")
		upgradeCmd.Flags().StringVarP(&upgradeConfig.Chart.localChart, "local-chart", "l", "", "Path to local chart directory/archive")
		upgradeCmd.Flags().BoolVarP(&upgradeConfig.Chart.remoteDownload, "remote-download", "r", false, "Enable downloading chart from remote repository")
		upgradeCmd.Flags().StringVar(&upgradeConfig.Chart.remoteVersion, "remote-version", "", "Version of the chart to download from remote repository")
		upgradeCmd.MarkFlagsMutuallyExclusive("local-chart", "remote-download")
	})
}
