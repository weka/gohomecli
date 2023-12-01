package chart

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/chart"
	"github.com/weka/gohomecli/internal/utils"
)

var logger = utils.GetLogger("HelmChart")

func normPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get user home directory")
			os.Exit(255)
		}

		path = filepath.Join(homeDir, path[2:])
	}

	return filepath.Clean(path)
}

func readKubeConfig(kubeConfigPath string) ([]byte, error) {
	if kubeConfigPath == "" {
		kubeConfigPath = os.Getenv("KUBECONFIG")
		if kubeConfigPath == "" {
			kubeConfigPath = "~/.kube/config"
		}
	}
	kubeConfigPath = normPath(kubeConfigPath)

	logger.Debug().Str("kubeConfigPath", kubeConfigPath).Msg("Reading kubeconfig")
	kubeConfig, err := os.ReadFile(kubeConfigPath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read kubeconfig")
		return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	return kubeConfig, nil
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

func runInstallOrUpgrade(cmd *cobra.Command, args []string) error {
	if installCmdOpts.remoteVersion != "" && !installCmdOpts.remoteDownload {
		return fmt.Errorf("%w: --remote-version can only be used with --remote-download", utils.ErrValidationFailed)
	}

	kubeConfig, err := readKubeConfig(installCmdOpts.kubeConfigPath)
	if err != nil {
		return err
	}

	chartConfig, err := readConfiguration(installCmdOpts.jsonConfig)
	if err != nil {
		return err
	}

	var chartLocation *chart.LocationOverride
	if installCmdOpts.remoteDownload {
		chartLocation = &chart.LocationOverride{
			RemoteDownload: true,
			Version:        installCmdOpts.remoteVersion,
		}
	}

	if installCmdOpts.localChart != "" {
		chartLocation = &chart.LocationOverride{
			Path: installCmdOpts.localChart,
		}
	}

	helmOptions := &chart.HelmOptions{
		KubeConfig: kubeConfig,
		Override:   chartLocation,
	}

	return chart.InstallOrUpgrade(cmd.Context(), chartConfig, helmOptions)
}
