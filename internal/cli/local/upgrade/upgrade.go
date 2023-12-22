package upgrade

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/install/bundle"
	"github.com/weka/gohomecli/internal/install/k3s"
	"github.com/weka/gohomecli/internal/install/web"
	"github.com/weka/gohomecli/internal/utils"
)

var (
	Cli    hooks.Cli
	logger = utils.GetLogger("upgrade")
)

var config struct {
	Web         bool
	WebBindAddr string
	BundlePath  string
	K3S         k3s.UpgradeConfig
	Chart       struct {
		kubeConfigPath string
		localChart     string
		jsonConfig     string
		remoteDownload bool
		remoteVersion  string
	}
}

var k3sImportConfig struct {
	ImagePaths []string
	FailFast   bool
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

		if web.IsEnabled() {
			upgradeCmd.Flags().BoolVar(&config.Web, "web", false, "start web installer")
			upgradeCmd.Flags().StringVarP(&config.WebBindAddr, "bind-addr", "b", ":8080", "Bind address for web server including port")
		}

		upgradeCmd.Flags().StringVar(&config.BundlePath, "bundle", bundle.BundlePath(), "bundle directory with k3s package")
		upgradeCmd.Flags().BoolVar(&config.K3S.Debug, "debug", false, "enable debug mode")

		upgradeCmd.Flags().MarkHidden("bundle")
		upgradeCmd.Flags().MarkHidden("debug")

		upgradeCmd.Flags().BoolVar(&k3sImportConfig.FailFast, "fail-fast", false, "fail on first error")
		upgradeCmd.Flags().StringSliceVarP(&k3sImportConfig.ImagePaths, "image-path", "f", nil, "images to import (if specified, bundle images are ignored)")

		upgradeCmd.Flags().StringVarP(&config.Chart.kubeConfigPath, "kube-config", "k", "/etc/rancher/k3s/k3s.yaml", "Path to kubeconfig file")
		upgradeCmd.Flags().StringVarP(&config.Chart.localChart, "local-chart", "l", "", "Path to local chart directory/archive")
		upgradeCmd.Flags().StringVarP(&config.Chart.jsonConfig, "json-config", "c", "", "Configuration in JSON format (file or JSON string)")
		upgradeCmd.Flags().BoolVarP(&config.Chart.remoteDownload, "remote-download", "r", false, "Enable downloading chart from remote repository")
		upgradeCmd.Flags().StringVar(&config.Chart.remoteVersion, "remote-version", "", "Version of the chart to download from remote repository")
		upgradeCmd.MarkFlagsMutuallyExclusive("local-chart", "remote-download")
	})
}
