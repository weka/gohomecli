package chart

import (
	"context"
	"fmt"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	"github.com/weka/gohomecli/internal/bundle"
	"github.com/weka/gohomecli/internal/utils"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/repo"
)

const (
	releaseName    = "wekahome"
	repositoryURL  = "https://weka.github.io/gohome"
	repositoryName = "wekahome"
	chartName      = "wekahome"
)

var logger = utils.GetLogger("HelmChart")

type HelmOptions struct {
	KubeConfig    []byte
	KubeContext   string
	Namespace     string
	AllowDownload bool
	PathOverride  string
}

func InstallOrUpgrade(
	ctx context.Context,
	cfg *Configuration,
	opts *HelmOptions,
) error {
	logger.Debug().
		Str("namespace", opts.Namespace).
		Msg("Configuring helm client")
	// kubeContext override isn't working - https://github.com/mittwald/go-helm-client/issues/127
	client, err := helmclient.NewClientFromKubeConf(&helmclient.KubeConfClientOptions{
		Options:     &helmclient.Options{Namespace: opts.Namespace},
		KubeContext: opts.KubeContext,
		KubeConfig:  opts.KubeConfig,
	})
	if err != nil {
		return fmt.Errorf("failed configuring helm client: %w", err)
	}

	logger.Debug().
		Bool("allowDownload", opts.AllowDownload).
		Str("pathOverride", opts.PathOverride).
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

	chartSpec := &helmclient.ChartSpec{
		ReleaseName:     releaseName,
		ChartName:       chartLocation,
		Namespace:       opts.Namespace,
		ValuesYaml:      string(valuesYaml),
		CreateNamespace: true,
		ResetValues:     true,
		Wait:            true,
		WaitForJobs:     true,
		Timeout:         time.Minute * 5,
	}

	logger.Debug().
		Str("namespace", opts.Namespace).
		Str("chart", chartSpec.ChartName).
		Str("release", chartSpec.ReleaseName).
		Msg("Installing/upgrading chart")

	fmt.Print()

	_, err = client.InstallOrUpgradeChart(ctx, chartSpec, nil)
	if err != nil {
		return fmt.Errorf("failed installing/upgrading chart: %w", err)
	}

	return nil
}

func getChartLocation(client helmclient.Client, opts *HelmOptions) (string, error) {
	var chartLocation string

	if opts.PathOverride != "" {
		chartLocation = opts.PathOverride
	} else if bundle.IsBundled() {
		chartLocation = bundle.GetPath("chart.tgz")
	} else if opts.AllowDownload {
		err := client.AddOrUpdateChartRepo(repo.Entry{
			Name: repositoryName,
			URL:  repositoryURL,
		})
		if err != nil {
			return "", fmt.Errorf("failed adding chart repo: %w", err)
		}

		chartLocation = fmt.Sprintf("%s/%s", repositoryName, chartName)
	} else {
		return "", fmt.Errorf("unable to determine chart location")
	}

	return chartLocation, nil
}
