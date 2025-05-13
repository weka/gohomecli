package chart

import (
	"errors"
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

var ErrUnableToFindChart = errors.New("unable to determine chart location")

var logger = utils.GetLogger("HelmChart")

type LocationOverride struct {
	Path           string
	Version        string
	RemoteDownload bool
}

type HelmOptions struct {
	Override          *LocationOverride
	Config            *config_v1.Configuration
	KubeContext       string
	NamespaceOverride string
	KubeConfig        []byte
	Values            []byte
}

func crdSpec(client helmclient.Client, opts *HelmOptions) (*helmclient.ChartSpec, error) {
	namespace := ReleaseNamespace
	if opts.NamespaceOverride != "" {
		namespace = opts.NamespaceOverride
	}

	logger.Debug().
		Interface("locationOverride", opts.Override).
		Msg("Determining chart crd location")

	chartLocation, err := getChartCrdLocation(client, opts)
	if err != nil {
		return nil, err
	}

	return &helmclient.ChartSpec{
		ReleaseName:     ReleaseName + "-crds",
		ChartName:       chartLocation,
		Namespace:       namespace,
		CreateNamespace: true,
		UpgradeCRDs:     true,
		ResetValues:     true,
		Wait:            true,
		Timeout:         time.Minute * 5,
	}, nil
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
		Interface("configuration", opts.Config.LoggingSafe()).
		Msg("Generating chart values")

	values, err := generateValuesV3(opts.Config)
	if err != nil {
		return nil, err
	}
	opts.Values, err = yaml.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed serializing values yaml: %w", err)
	}

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
		UpgradeCRDs:     true,
		Timeout:         time.Minute * 5,
	}, nil
}

func getChartCrdLocation(client helmclient.Client, opts *HelmOptions) (string, error) {
	var chartLocation string

	if opts.Override != nil && opts.Override.RemoteDownload {
		err := client.AddOrUpdateChartRepo(repo.Entry{
			Name: RepositoryName,
			URL:  RepositoryURL,
		})
		if err != nil {
			return "", fmt.Errorf("failed adding chart repo: %w", err)
		}

		chartLocation = fmt.Sprintf("%s/%s-crds", RepositoryName, ChartName)

		return chartLocation, nil
	}

	if bundle.IsBundled() {
		return bundle.ChartCrd()
	}

	return "", ErrUnableToFindChart
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
