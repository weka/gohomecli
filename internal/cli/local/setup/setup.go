package setup

import (
	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/cli/app/hooks"
	"github.com/weka/gohomecli/internal/local/bundle"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/local/k3s"
	"github.com/weka/gohomecli/internal/local/web"
	"github.com/weka/gohomecli/internal/utils"
)

var (
	Cli    hooks.Cli
	logger = utils.GetLogger("setup")
)

var setupConfig struct {
	config_v1.Configuration

	Web         bool
	WebBindAddr string
	BundlePath  string
	JsonConfig  string
	ValuesFile  string
	Iface       string
	Debug       bool

	Chart struct {
		kubeConfigPath string
		localChart     string
		remoteDownload bool
		remoteVersion  string
	}
}

var k3sImportConfig struct {
	ImagePaths []string
	FailFast   bool
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install Local Weka Home",
	Long:  `Install Weka Home Helm chart with K3S bundle`,
	RunE:  runSetup,
}

func init() {
	Cli.AddHook(func(appCmd *cobra.Command) {
		appCmd.AddCommand(setupCmd)

		setupCmd.Flags().StringVarP(&setupConfig.JsonConfig, "json-config", "c", "", "Configuration in JSON format (file or JSON string)")
		setupCmd.Flags().StringVar(&setupConfig.ValuesFile, "values", "", "Path to values.yaml (optional)")
		setupCmd.MarkFlagsMutuallyExclusive("json-config", "values")

		if web.IsEnabled() {
			setupCmd.Flags().BoolVar(&setupConfig.Web, "web", false, "start web installer")
			setupCmd.Flags().StringVarP(&setupConfig.WebBindAddr, "bind-addr", "b", ":8080", "Bind address for web server including port")
		}

		setupCmd.Flags().StringVar(&setupConfig.BundlePath, "bundle", bundle.BundlePath(), "bundle directory with k3s package")

		setupCmd.Flags().StringVarP(&setupConfig.Iface, "iface", "i", "", "interface for k3s network")
		setupCmd.Flags().StringVarP(setupConfig.Host, "hostname", "n", k3s.Hostname(), "hostname for cluster")
		setupCmd.Flags().StringVar(&setupConfig.NodeIP, "ip", "", "primary IP internal address for wekahome API")
		setupCmd.Flags().StringSliceVar(&setupConfig.ExternalIPs, "ips", nil, "additional IP addresses for wekahome API (e.g public ip)")
		setupCmd.Flags().BoolVar(&setupConfig.Debug, "debug", false, "enable debug mode")
		setupCmd.Flags().StringVar(setupConfig.TLS.Cert, "cert", "", "TLS certificate")
		setupCmd.Flags().StringVar(setupConfig.TLS.Key, "key", "", "TLS secret key")

		setupCmd.MarkFlagRequired("iface")
		setupCmd.Flags().MarkHidden("bundle")
		setupCmd.Flags().MarkHidden("ip")
		setupCmd.Flags().MarkHidden("debug")

		setupCmd.Flags().BoolVar(&k3sImportConfig.FailFast, "fail-fast", false, "fail on first error")
		setupCmd.Flags().StringSliceVarP(&k3sImportConfig.ImagePaths, "image-path", "f", nil, "images to import (if specified, bundle images are ignored)")

		setupCmd.Flags().StringVarP(&setupConfig.Chart.kubeConfigPath, "kube-config", "k", "/etc/rancher/k3s/k3s.yaml", "Path to kubeconfig file")
		setupCmd.Flags().StringVarP(&setupConfig.Chart.localChart, "local-chart", "l", "", "Path to local chart directory/archive")
		setupCmd.Flags().BoolVarP(&setupConfig.Chart.remoteDownload, "remote-download", "r", false, "Enable downloading chart from remote repository")
		setupCmd.Flags().StringVar(&setupConfig.Chart.remoteVersion, "remote-version", "", "Version of the chart to download from remote repository")
		setupCmd.MarkFlagsMutuallyExclusive("local-chart", "remote-download")
	})
}
