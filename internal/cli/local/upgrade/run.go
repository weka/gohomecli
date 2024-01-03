package upgrade

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/local/bundle"
	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/local/k3s"
)

func runUpgrade(cmd *cobra.Command, args []string) error {
	if upgradeConfig.BundlePath != bundle.BundlePath() {
		if err := bundle.SetBundlePath(upgradeConfig.BundlePath); err != nil {
			return err
		}
	}

	err := k3s.Upgrade(cmd.Context(), k3s.UpgradeConfig{Debug: upgradeConfig.Debug})
	if err != nil {
		return err
	}

	time.Sleep(5 * time.Second) // wait for k3s to be ready

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

	return chart.Upgrade(cmd.Context(), helmOptions, upgradeConfig.Debug)
}
