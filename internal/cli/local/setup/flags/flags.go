package setup_flags

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/local/bundle"
	"github.com/weka/gohomecli/internal/local/web"
	"github.com/weka/gohomecli/internal/utils"
	"k8s.io/utils/strings/slices"
)

var (
	validProxyScheme    = []string{"http", "https", "socks5"}
	validK3SProxyScheme = []string{"http", "https"}
)

type Flags struct {
	Web             bool
	WebBindAddr     string
	Proxy           string
	ProxyKubernetes bool
	BundlePath      string
	JsonConfig      string
	Chart           struct {
		LocalChart     string
		RemoteDownload bool
		RemoteVersion  string
	}

	Debug bool
}

func (c Flags) Validate() error {
	if c.Chart.RemoteVersion != "" && !c.Chart.RemoteDownload {
		return fmt.Errorf("%w: --remote-version can only be used with --remote-download", utils.ErrValidationFailed)
	}

	if c.Proxy != "" {
		addr, err := url.Parse(c.Proxy)
		if err != nil {
			return err
		}

		if !slices.Contains(validProxyScheme, addr.Scheme) {
			return fmt.Errorf("proxy supports only %v, %s is given", validProxyScheme, addr.Scheme)
		}

		if c.ProxyKubernetes && !slices.Contains(validK3SProxyScheme, addr.Scheme) {
			return fmt.Errorf("kubernetes proxy supports only %v, %s is given", validK3SProxyScheme, addr.Scheme)
		}
	}

	return nil
}

func Use(cmd *cobra.Command, config *Flags) {
	if web.IsEnabled() {
		cmd.Flags().BoolVar(&config.Web, "web", false, "start web installer")
		cmd.Flags().StringVarP(&config.WebBindAddr, "bind-addr", "b", ":8080", "Bind address for web server including port")
	}

	cmd.Flags().StringVar(&config.Proxy, "proxy", "", fmt.Sprintf("Use proxy URL for networking (example: http://user:password@addr), supported proxy type: %v", validProxyScheme))
	cmd.Flags().BoolVar(&config.ProxyKubernetes, "proxy-kubernetes", false, fmt.Sprintf("Add proxy support for kubernetes, supported proxy type: %v", validK3SProxyScheme))
	cmd.Flags().StringVarP(&config.JsonConfig, "json-config", "c", "", "Configuration in JSON format")

	cmd.Flags().StringVar(&config.BundlePath, "bundle", bundle.BundlePath(), "bundle directory with k3s package")
	cmd.Flags().BoolVar(&config.Debug, "debug", false, "enable debug mode")

	cmd.Flags().MarkHidden("bundle")
	cmd.Flags().MarkHidden("debug")

	cmd.Flags().StringVarP(&config.Chart.LocalChart, "local-chart", "l", "", "Path to local chart directory/archive")
	cmd.Flags().BoolVarP(&config.Chart.RemoteDownload, "remote-download", "r", false, "Enable downloading chart from remote repository")
	cmd.Flags().StringVar(&config.Chart.RemoteVersion, "remote-version", "", "Version of the chart to download from remote repository")
	cmd.MarkFlagsMutuallyExclusive("local-chart", "remote-download")
}
