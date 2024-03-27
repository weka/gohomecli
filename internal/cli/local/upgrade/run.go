package upgrade

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/local/bundle"
	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/local/config"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/local/k3s"
)

func runUpgrade(cmd *cobra.Command, args []string) error {
	if upgradeConfig.BundlePath != bundle.BundlePath() {
		if err := bundle.SetBundlePath(upgradeConfig.BundlePath); err != nil {
			return err
		}
	}

	err := k3s.Upgrade(cmd.Context(), k3s.Config{
		Configuration:   &upgradeConfig.Configuration,
		Iface:           upgradeConfig.Iface,
		ProxyKubernetes: upgradeConfig.ProxyKubernetes,
		Debug:           upgradeConfig.Debug,
	})
	if err != nil {
		return err
	}

	k3s.Wait(cmd.Context())

	// in debug mode we don't do fail-fast
	err = k3s.ImportBundleImages(cmd.Context(), !upgradeConfig.Debug)

	if err != nil {
		return err
	}

	kubeConfig, err := chart.ReadKubeConfig(chart.KubeConfigPath)
	if err != nil {
		return err
	}

	var chartLocation *chart.LocationOverride
	if upgradeConfig.Chart.RemoteDownload {
		chartLocation = &chart.LocationOverride{
			RemoteDownload: true,
			Version:        upgradeConfig.Chart.RemoteVersion,
		}
	}

	if upgradeConfig.Chart.LocalChart != "" {
		chartLocation = &chart.LocationOverride{
			Path: upgradeConfig.Chart.LocalChart,
		}
	}

	helmOptions := &chart.HelmOptions{
		KubeConfig: kubeConfig,
		Override:   chartLocation,
		Config:     &upgradeConfig.Configuration,
	}

	err = chart.Upgrade(cmd.Context(), helmOptions, upgradeConfig.Debug)
	if err != nil {
		return fmt.Errorf("chart upgrade: %w", err)
	}

	return config.SaveV1(config.CLIConfig, upgradeConfig.Configuration)
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
