package setup

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/local/bundle"
	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/local/config"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/local/k3s"
	"github.com/weka/gohomecli/internal/local/web"
	"github.com/weka/gohomecli/internal/utils"
)

var logger = utils.GetLogger("setup")

func runSetup(cmd *cobra.Command, args []string) (err error) {
	if setupConfig.BundlePath != bundle.BundlePath() {
		if err := bundle.SetBundlePath(setupConfig.BundlePath); err != nil {
			return err
		}
	}

	if setupConfig.Web {
		logger.Info().Str("bindAddr", setupConfig.WebBindAddr).Msg("Starting web server")
		err := web.ServeConfigurer(cmd.Context(), setupConfig.WebBindAddr)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}

	// cleanup on error during installing
	defer func() {
		var skipErrors = []error{
			k3s.ErrExists,
			chart.ErrTimeout,
			context.DeadlineExceeded,
		}

		if err != nil {
			for _, skipErr := range skipErrors {
				if errors.Is(err, skipErr) {
					return
				}
			}

			k3s.Cleanup(setupConfig.Debug)
		}
	}()

	err = k3s.Install(cmd.Context(), k3s.InstallConfig{
		Configuration:   &setupConfig.Configuration,
		Iface:           setupConfig.Iface,
		ProxyKubernetes: setupConfig.ProxyKubernetes,
		Debug:           setupConfig.Debug,
	})
	if err != nil {
		return err
	}

	// wait before continue
	k3s.Wait(cmd.Context())

	// in debug mode we don't do fail-fast
	err = k3s.ImportBundleImages(cmd.Context(), !setupConfig.Debug)
	if err != nil {
		return err
	}

	kubeConfig, err := chart.ReadKubeConfig(chart.KubeConfigPath)
	if err != nil {
		return err
	}

	var chartLocation *chart.LocationOverride
	if setupConfig.Chart.RemoteDownload {
		chartLocation = &chart.LocationOverride{
			RemoteDownload: true,
			Version:        setupConfig.Chart.RemoteVersion,
		}
	}

	if setupConfig.Chart.LocalChart != "" {
		chartLocation = &chart.LocationOverride{
			Path: setupConfig.Chart.LocalChart,
		}
	}

	helmOptions := &chart.HelmOptions{
		KubeConfig: kubeConfig,
		Override:   chartLocation,
		Config:     &setupConfig.Configuration,
	}

	err = chart.Install(cmd.Context(), helmOptions)
	if err != nil {
		return fmt.Errorf("chart install: %w", err)
	}

	return config.SaveV1(config.CLIConfig, setupConfig.Configuration)
}

func readTLS(certFile, keyFile string, config *config_v1.Configuration) error {
	if certFile == "" || keyFile == "" {
		return nil
	}

	cert, err := os.ReadFile(certFile)
	if err != nil {
		return err
	}
	config.TLS.Cert = string(cert)

	key, err := os.ReadFile(keyFile)
	if err != nil {
		return err
	}

	config.TLS.Key = string(key)

	return nil
}
