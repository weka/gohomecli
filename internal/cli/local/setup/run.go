package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/install/bundle"
	"github.com/weka/gohomecli/internal/install/chart"
	"github.com/weka/gohomecli/internal/install/k3s"
	"github.com/weka/gohomecli/internal/install/web"
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

func readConfiguration(jsonConfig string) (*chart.Configuration, error) {
	if jsonConfig == "" {
		return &chart.Configuration{}, nil
	}

	var jsonConfigBytes []byte
	if _, err := os.Stat(jsonConfig); err == nil {
		logger.Debug().Str("path", jsonConfig).Msg("Reading JSON config from file")
		jsonConfigBytes, err = os.ReadFile(jsonConfig)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to read JSON config from file")
			return nil, fmt.Errorf("failed to read JSON config from file: %w", err)
		}
	} else {
		logger.Debug().Msg("Using JSON object from command line")
		jsonConfigBytes = []byte(jsonConfig)
	}

	logger.Debug().Msg("Parsing JSON config")
	config := &chart.Configuration{}
	err := json.Unmarshal(jsonConfigBytes, &config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse JSON config")
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	return config, nil
}

func runSetup(cmd *cobra.Command, args []string) error {
	if config.BundlePath != bundle.BundlePath() {
		if err := bundle.SetBundlePath(config.BundlePath); err != nil {
			return err
		}
	}

	if config.Web {
		logger.Info().Str("bindAddr", config.WebBindAddr).Msg("Starting web server")
		err := web.ServeConfigurer(cmd.Context(), config.WebBindAddr)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}

	err := k3s.Install(cmd.Context(), config.K3S)
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

	if config.Chart.remoteVersion != "" && !config.Chart.remoteDownload {
		return fmt.Errorf("%w: --remote-version can only be used with --remote-download", utils.ErrValidationFailed)
	}

	if config.Chart.kubeConfigPath != "" {
		config.Chart.kubeConfigPath, err = normPath(config.Chart.kubeConfigPath)
		if err != nil {
			return err
		}
	}

	kubeConfig, err := chart.ReadKubeConfig(config.Chart.kubeConfigPath)
	if err != nil {
		return err
	}

	chartConfig, err := readConfiguration(config.Chart.jsonConfig)
	if err != nil {
		return err
	}

	var chartLocation *chart.LocationOverride
	if config.Chart.remoteDownload {
		chartLocation = &chart.LocationOverride{
			RemoteDownload: true,
			Version:        config.Chart.remoteVersion,
		}
	}

	if config.Chart.localChart != "" {
		chartLocation = &chart.LocationOverride{
			Path: config.Chart.localChart,
		}
	}

	helmOptions := &chart.HelmOptions{
		KubeConfig: kubeConfig,
		Override:   chartLocation,
	}

	return chart.InstallOrUpgrade(cmd.Context(), chartConfig, helmOptions)
}
