package setup

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/local/bundle"
	"github.com/weka/gohomecli/internal/local/chart"
	"github.com/weka/gohomecli/internal/local/k3s"
	"github.com/weka/gohomecli/internal/local/web"
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

func runSetup(cmd *cobra.Command, args []string) error {
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

	err := k3s.Install(cmd.Context(), k3s.InstallConfig{
		Configuration: setupConfig.Configuration,
		Iface:         setupConfig.Iface,
		Debug:         setupConfig.Debug,
	})
	if err != nil {
		return err
	}

	if len(k3sImportConfig.ImagePaths) == 0 {
		err = k3s.ImportBundleImages(cmd.Context(), k3sImportConfig.FailFast)
	} else {
		err = k3s.ImportImages(cmd.Context(), k3sImportConfig.ImagePaths, k3sImportConfig.FailFast)
	}

	if err != nil {
		return err
	}

	if setupConfig.Chart.remoteVersion != "" && !setupConfig.Chart.remoteDownload {
		return fmt.Errorf("%w: --remote-version can only be used with --remote-download", utils.ErrValidationFailed)
	}

	if setupConfig.Chart.kubeConfigPath != "" {
		setupConfig.Chart.kubeConfigPath, err = normPath(setupConfig.Chart.kubeConfigPath)
		if err != nil {
			return err
		}
	}

	kubeConfig, err := chart.ReadKubeConfig(setupConfig.Chart.kubeConfigPath)
	if err != nil {
		return err
	}

	var chartLocation *chart.LocationOverride
	if setupConfig.Chart.remoteDownload {
		chartLocation = &chart.LocationOverride{
			RemoteDownload: true,
			Version:        setupConfig.Chart.remoteVersion,
		}
	}

	if setupConfig.Chart.localChart != "" {
		chartLocation = &chart.LocationOverride{
			Path: setupConfig.Chart.localChart,
		}
	}

	helmOptions := &chart.HelmOptions{
		KubeConfig: kubeConfig,
		Override:   chartLocation,
		Values:     setupConfig.Chart.values,
		Config:     &setupConfig.Configuration,
	}

	return chart.Install(cmd.Context(), helmOptions)
}
