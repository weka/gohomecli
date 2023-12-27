package upgrade

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/weka/gohomecli/internal/local/bundle"
	"github.com/weka/gohomecli/internal/local/chart"
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

func runUpgrade(cmd *cobra.Command, args []string) error {
	if config.BundlePath != bundle.BundlePath() {
		if err := bundle.SetBundlePath(config.BundlePath); err != nil {
			return err
		}
	}

	err := k3s.Upgrade(cmd.Context(), k3s.UpgradeConfig{Debug: config.Debug})
	if err != nil {
		return err
	}

	time.Sleep(5 * time.Second) // wait for k3s to be ready

	// in debug mode we don't do fail-fast
	if len(config.Images.ImportPaths) == 0 {
		err = k3s.ImportBundleImages(cmd.Context(), !config.Debug)
	} else {
		err = k3s.ImportImages(cmd.Context(), config.Images.ImportPaths, !config.Debug)
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

	var values []byte
	if config.Chart.valuesFile != "" {
		values, err = os.ReadFile(config.Chart.valuesFile)
		if err != nil {
			return fmt.Errorf("reading values.yaml: %w", err)
		}
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
		Config:     chartConfig,
		Values:     values,
	}

	return chart.Upgrade(cmd.Context(), helmOptions, config.Debug)
}
