package chart

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/weka/gohomecli/internal/chart"
	"github.com/weka/gohomecli/internal/utils"
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

func buildChartInstallCmd() *cobra.Command {
	cliOpts := struct {
		kubeConfigPath string
		localChart     string
		jsonConfig     string
		remoteDownload bool
		remoteVersion  string
	}{}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Weka Home Helm chart",
		Long:  `Install Weka Home Helm chart on already deployed Kubernetes cluster`,
		Args:  cobra.NoArgs,
		PreRun: func(cmd *cobra.Command, args []string) {
			ctx, _ := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
			cmd.SetContext(ctx)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cliOpts.remoteVersion != "" && !cliOpts.remoteDownload {
				return fmt.Errorf("--remote-version can only be used with --remote-download")
			}

			kubeConfig, err := readKubeConfig(cliOpts.kubeConfigPath)
			if err != nil {
				return err
			}

			chartConfig, err := readConfiguration(cliOpts.jsonConfig)
			if err != nil {
				return err
			}

			var chartLocation *chart.LocationOverride
			if cliOpts.remoteDownload {
				chartLocation = &chart.LocationOverride{
					RemoteDownload: true,
					Version:        cliOpts.remoteVersion,
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

	cmd.Flags().BoolVarP(&cliOpts.remoteDownload, "remote-download", "r", false, "Enable downloading chart from remote repository")
	cmd.Flags().StringVar(&cliOpts.remoteVersion, "remote-version", "", "Version of the chart to download from remote repository")

	cmd.MarkFlagsMutuallyExclusive("local-chart", "remote-download")

	return cmd
}
