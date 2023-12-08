package chart

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/weka/gohomecli/internal/install/bundle"

	helmclient "github.com/mittwald/go-helm-client"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/weka/gohomecli/internal/utils"
)

const (
	ReleaseName      = "wekahome"
	ReleaseNamespace = "home-weka-io"
	RepositoryURL    = "https://weka.github.io/gohome"
	RepositoryName   = "wekahome"
	ChartName        = "wekahome"
)

var ErrUnableToFindChart = fmt.Errorf("unable to determine chart location")

var logger = utils.GetLogger("HelmChart")

type LocationOverride struct {
	Path           string // path to chart package
	RemoteDownload bool   // download from remote repository
	Version        string // version of the chart to download from remote repository
}

type HelmOptions struct {
	KubeConfig        []byte            // path or content of kubeconfig file
	Override          *LocationOverride // override chart package location
	KubeContext       string            // kubeconfig context to use
	NamespaceOverride string            // override namespace for release
}

func InstallOrUpgrade(
	ctx context.Context,
	cfg *Configuration,
	opts *HelmOptions,
) error {
	namespace := ReleaseNamespace
	if opts.NamespaceOverride != "" {
		namespace = opts.NamespaceOverride
	}

	logger.Info().
		Str("namespace", namespace).
		Str("kubeContext", opts.KubeContext).
		Msg("Configuring helm client")

	// kubeContext override isn't working - https://github.com/mittwald/go-helm-client/issues/127
	client, err := helmclient.NewClientFromKubeConf(&helmclient.KubeConfClientOptions{
		Options: &helmclient.Options{
			Namespace: namespace,
			DebugLog: func(format string, v ...interface{}) {
				logger.Debug().Msgf(format, v...)
			},
			Output: utils.NewWriteScanner(func(b []byte) {
				logger.Info().Msg(string(b))
			}),
		},
		KubeContext: opts.KubeContext,
		KubeConfig:  opts.KubeConfig,
	})
	if err != nil {
		return fmt.Errorf("failed configuring helm client: %w", err)
	}

	logger.Debug().
		Interface("locationOverride", opts.Override).
		Msg("Determining chart location")
	chartLocation, err := getChartLocation(client, opts)
	if err != nil {
		return err
	}

	logger.Debug().
		Interface("configuration", cfg).
		Msg("Generating chart values")

	values, err := generateValuesV3(cfg)
	if err != nil {
		return err
	}

	valuesYaml, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed serializing values yaml: %w", err)
	}
	logger.Debug().Msgf("Generated values:\n %s", string(valuesYaml))

	chartVersion := "" // any available
	if opts.Override != nil {
		chartVersion = opts.Override.Version
	}
	chartSpec := &helmclient.ChartSpec{
		ReleaseName:     ReleaseName,
		ChartName:       chartLocation,
		Version:         chartVersion,
		Namespace:       namespace,
		ValuesYaml:      string(valuesYaml),
		CreateNamespace: true,
		ResetValues:     true,
		Wait:            true,
		WaitForJobs:     true,
		Timeout:         time.Minute * 5,
	}

	logger.Info().
		Str("namespace", namespace).
		Str("chart", chartSpec.ChartName).
		Str("release", chartSpec.ReleaseName).
		Msg("Installing/upgrading chart")

	release, err := client.InstallOrUpgradeChart(ctx, chartSpec, nil)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info().Msg("Chart installation was cancelled")
			return nil
		}
		return fmt.Errorf("failed installing/upgrading chart: %w", err)
	}

	logger.Info().Msg(release.Info.Notes)

	return nil
}

func findBundledChart() (string, error) {
	path := ""

	err := bundle.Walk("", func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path != "" {
			return nil
		}

		matched, err := filepath.Match("wekahome-*.tgz", info.Name())
		if err != nil {
			return err
		}

		if matched {
			path = name
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("unable to find wekahome chart in bundle")
	}

	return path, nil
}

func getChartLocation(client helmclient.Client, opts *HelmOptions) (string, error) {
	var chartLocation string

	if opts.Override != nil && opts.Override.RemoteDownload {
		err := client.AddOrUpdateChartRepo(repo.Entry{
			Name: RepositoryName,
			URL:  RepositoryURL,
		})
		if err != nil {
			return "", fmt.Errorf("failed adding chart repo: %w", err)
		}

		chartLocation = fmt.Sprintf("%s/%s", RepositoryName, ChartName)
		return chartLocation, nil
	}

	if opts.Override != nil && opts.Override.Path != "" {
		return opts.Override.Path, nil
	}

	if bundle.IsBundled() {
		return findBundledChart()
	}

	return "", ErrUnableToFindChart
}
