package cli

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

const (
	defaultVersionSelected = "latest"
)

var (
	logger = utils.GetLogger("HelmChart")
)

type FlatConfiguration struct {
	Host string `json:"host"`
}

func (c *FlatConfiguration) ToChartConfiguration() *chart.Configuration {
	return &chart.Configuration{
		Host: c.Host,
	}
}

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
	var flatConfig FlatConfiguration
	err := json.Unmarshal(jsonConfigBytes, &flatConfig)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse JSON config")
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	return flatConfig.ToChartConfiguration(), nil
}

// Note: there's an issue that -r/--remote-download can't accept values without =
func buildChartInstallCmd() *cobra.Command {
	cliOpts := struct {
		kubeConfigPath string
		localChart     string
		jsonConfig     string
		remoteDownload string
	}{}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Weka Home Helm chart",
		Long:  `Install Weka Home Helm chart on already deployed Kubernetes cluster`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeConfig, err := readKubeConfig(cliOpts.kubeConfigPath)
			if err != nil {
				return err
			}

			chartConfig, err := readConfiguration(cliOpts.jsonConfig)
			if err != nil {
				return err
			}

			var chartLocation *chart.LocationOverride
			if cliOpts.remoteDownload == defaultVersionSelected {
				chartLocation = &chart.LocationOverride{
					RemoteDownload: true,
				}
			} else if cliOpts.remoteDownload != "" {
				chartLocation = &chart.LocationOverride{
					RemoteDownload: true,
					Version:        cliOpts.remoteDownload,
				}
			}

			if cliOpts.localChart != "" {
				chartLocation = &chart.LocationOverride{
					Path: cliOpts.localChart,
				}
			}

			helmOptions := &chart.HelmOptions{
				KubeConfig: kubeConfig,
				Override:   chartLocation,
			}

			return chart.InstallOrUpgrade(cmd.Context(), chartConfig, helmOptions)
		},
	}

	cmd.Flags().StringVarP(&cliOpts.kubeConfigPath, "kube-config", "k", "", "Path to kubeconfig file")
	cmd.Flags().StringVarP(&cliOpts.localChart, "local-chart", "l", "", "Path to local chart directory/archive")
	cmd.Flags().StringVarP(&cliOpts.jsonConfig, "json-config", "c", "", "Configuration in JSON format (file or JSON string)")

	// --remote-download should be available without any options
	cmd.Flags().StringVarP(&cliOpts.remoteDownload, "remote-download", "r", "", "Enable downloading chart from remote repository, optionally specify version")
	cmd.Flags().Lookup("remote-download").NoOptDefVal = defaultVersionSelected

	cmd.MarkFlagsMutuallyExclusive("local-chart", "remote-download")

	return cmd
}
