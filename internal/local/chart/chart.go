package chart

import (
	"fmt"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/weka/gohomecli/internal/local/bundle"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
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
	KubeConfig        []byte                   // path or content of kubeconfig file
	Override          *LocationOverride        // override chart package location
	KubeContext       string                   // kubeconfig context to use
	NamespaceOverride string                   // override namespace for release
	Values            []byte                   // raw values.yaml
	Config            *config_v1.Configuration // json config
}

func chartSpec(client helmclient.Client, opts *HelmOptions) (*helmclient.ChartSpec, error) {
	namespace := ReleaseNamespace
	if opts.NamespaceOverride != "" {
		namespace = opts.NamespaceOverride
	}

	logger.Debug().
		Interface("locationOverride", opts.Override).
		Msg("Determining chart location")
	chartLocation, err := getChartLocation(client, opts)
	if err != nil {
		return nil, err
	}

	logger.Debug().
		Interface("configuration", opts.Config).
		Msg("Generating chart values")

	values, err := generateValuesV3(opts.Config)
	if err != nil {
		return nil, err
	}
	opts.Values, err = yaml.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed serializing values yaml: %w", err)
	}

	logger.Debug().Msgf("Helm values.yaml:\n%s", string(opts.Values))

	chartVersion := "" // any available
	if opts.Override != nil {
		chartVersion = opts.Override.Version
	}
	return &helmclient.ChartSpec{
		ReleaseName:     ReleaseName,
		ChartName:       chartLocation,
		Version:         chartVersion,
		Namespace:       namespace,
		ValuesYaml:      string(opts.Values),
		CreateNamespace: true,
		ResetValues:     true,
		Wait:            true,
		WaitForJobs:     true,
		Timeout:         time.Minute * 5,
	}, nil
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
		return bundle.Chart()
	}

	return "", ErrUnableToFindChart
}
