package chart

import (
	"context"
	"errors"
	"fmt"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	"github.com/weka/gohomecli/internal/bundle"
	"github.com/weka/gohomecli/internal/utils"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/repo"
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
	Path           string
	RemoteDownload bool
	Version        string
}

type HelmOptions struct {
	KubeConfig        []byte
	Override          *LocationOverride
	KubeContext       string
	NamespaceOverride string
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

	logger.Debug().
		Str("namespace", namespace).
		Str("kubeContext", opts.KubeContext).
		Msg("Configuring helm client")
	// kubeContext override isn't working - https://github.com/mittwald/go-helm-client/issues/127
	client, err := helmclient.NewClientFromKubeConf(&helmclient.KubeConfClientOptions{
		Options:     &helmclient.Options{Namespace: namespace},
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

	chartVersion := "" // latest available
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

	logger.Debug().
		Str("namespace", namespace).
		Str("chart", chartSpec.ChartName).
		Str("release", chartSpec.ReleaseName).
		Msg("Installing/upgrading chart")

	_, err = client.InstallOrUpgradeChart(ctx, chartSpec, nil)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("failed installing/upgrading chart: %w", err)
	}

	return nil
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
		return bundle.GetPath("chart.tgz"), nil
	}

	return "", ErrUnableToFindChart
}