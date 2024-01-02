package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/local/bundle"
	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/local/config"
	"github.com/weka/gohomecli/internal/local/k3s"
	"github.com/weka/gohomecli/internal/utils"
)

func normPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to expand home directory: %w", err)
		}

		path = filepath.Join(homeDir, path[2:])
	}

	return filepath.Clean(path), nil
}

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
	if len(upgradeConfig.Images.ImportPaths) == 0 {
		err = k3s.ImportBundleImages(cmd.Context(), !upgradeConfig.Debug)
	} else {
		err = k3s.ImportImages(cmd.Context(), upgradeConfig.Images.ImportPaths, !upgradeConfig.Debug)
	}

	if err != nil {
		return err
	}

	if upgradeConfig.Chart.remoteVersion != "" && !upgradeConfig.Chart.remoteDownload {
		return fmt.Errorf("%w: --remote-version can only be used with --remote-download", utils.ErrValidationFailed)
	}

	if upgradeConfig.Chart.kubeConfigPath != "" {
		upgradeConfig.Chart.kubeConfigPath, err = normPath(upgradeConfig.Chart.kubeConfigPath)
		if err != nil {
			return err
		}
	}

	kubeConfig, err := chart.ReadKubeConfig(upgradeConfig.Chart.kubeConfigPath)
	if err != nil {
		return err
	}

	err = config.ReadV1(upgradeConfig.JsonConfig, &upgradeConfig.Configuration)
	if err != nil {
		return err
	}

	var values []byte
	if upgradeConfig.ValuesFile != "" {
		values, err = os.ReadFile(upgradeConfig.ValuesFile)
		if err != nil {
			return fmt.Errorf("reading values.yaml: %w", err)
		}
	}

	var chartLocation *chart.LocationOverride
	if upgradeConfig.Chart.remoteDownload {
		chartLocation = &chart.LocationOverride{
			RemoteDownload: true,
			Version:        upgradeConfig.Chart.remoteVersion,
		}
	}

	if upgradeConfig.Chart.localChart != "" {
		chartLocation = &chart.LocationOverride{
			Path: upgradeConfig.Chart.localChart,
		}
	}

	helmOptions := &chart.HelmOptions{
		KubeConfig: kubeConfig,
		Override:   chartLocation,
		Config:     &upgradeConfig.Configuration,
		Values:     values,
	}

	return chart.Upgrade(cmd.Context(), helmOptions, upgradeConfig.Debug)
}
