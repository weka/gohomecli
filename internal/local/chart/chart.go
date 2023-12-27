package chart

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/weka/gohomecli/internal/local/bundle"
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

// Configuration flat options for the chart, pointers are used to distinguish between empty and unset values
type Configuration struct {
	Host    *string `json:"host"`    // ingress host
	TLS     *bool   `json:"tls"`     // ingress tls enabled
	TLSCert *string `json:"tlsCert"` // ingress tls cert
	TLSKey  *string `json:"tlsKey"`  // ingress tls key

	SMTP struct {
		Host        *string `json:"host"`        // smtp server host
		Port        *int    `json:"port"`        // smtp server port
		User        *string `json:"user"`        // smtp server user
		Password    *string `json:"password"`    // smtp server password
		Insecure    *bool   `json:"insecure"`    // smtp insecure connection
		Sender      *string `json:"sender"`      // smtp sender name
		SenderEmail *string `json:"senderEmail"` // smtp sender email
	} `json:"smtp"`

	RetentionDays struct {
		Diagnostics *int `json:"diagnostics"` // diagnostics retention days
		Events      *int `json:"events"`      // events retention days
	} `json:"retention_days"`

	Forwarding struct {
		Enabled                   bool    `json:"enabled"`                   // forwarding enabled
		Url                       *string `json:"url"`                       // forwarding url override
		EnableEvents              *bool   `json:"enableEvents"`              // forwarding enable events
		EnableUsageReports        *bool   `json:"enableUsageReports"`        // forwarding enable usage reports
		EnableAnalytics           *bool   `json:"enableAnalytics"`           // forwarding enable analytics
		EnableDiagnostics         *bool   `json:"enableDiagnostics"`         // forwarding enable diagnostics
		EnableStats               *bool   `json:"enableStats"`               // forwarding enable stats
		EnableClusterRegistration *bool   `json:"enableClusterRegistration"` // forwarding enable cluster registration
	} `json:"forwarding"`

	Autoscaling     bool `json:"autoscaling"`        // enable services autoscaling
	WekaNodesServed *int `json:"wekaNodesMonitored"` // number of weka nodes to monitor, controls load preset
}

func chartSpec(client helmclient.Client, cfg *Configuration, opts *HelmOptions) (*helmclient.ChartSpec, error) {
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
		Interface("configuration", cfg).
		Msg("Generating chart values")

	values, err := generateValuesV3(cfg)
	if err != nil {
		return nil, err
	}

	valuesYaml, err := yaml.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed serializing values yaml: %w", err)
	}
	logger.Debug().Msgf("Generated values:\n %s", string(valuesYaml))

	chartVersion := "" // any available
	if opts.Override != nil {
		chartVersion = opts.Override.Version
	}
	return &helmclient.ChartSpec{
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
	}, nil
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

	if err != nil || path == "" {
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
