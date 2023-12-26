package chart

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/weka/gohomecli/internal/install/bundle"
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

	SMTPHost        *string `json:"smtpHost"`        // smtp server host
	SMTPPort        *int    `json:"smtpPort"`        // smtp server port
	SMTPUser        *string `json:"smtpUser"`        // smtp server user
	SMTPPassword    *string `json:"smtpPassword"`    // smtp server password
	SMTPInsecure    *bool   `json:"smtpInsecure"`    // smtp insecure connection
	SMTPSender      *string `json:"smtpSender"`      // smtp sender name
	SMTPSenderEmail *string `json:"smtpSenderEmail"` // smtp sender email

	DiagnosticsRetentionDays *int `json:"diagnosticsRetentionDays"` // diagnostics retention days
	EventsRetentionDays      *int `json:"eventsRetentionDays"`      // events retention days

	ForwardingEnabled                   bool    `json:"forwardingEnabled"`                   // forwarding enabled
	ForwardingUrl                       *string `json:"forwardingUrl"`                       // forwarding url override
	ForwardingEnableEvents              *bool   `json:"forwardingEnableEvents"`              // forwarding enable events
	ForwardingEnableUsageReports        *bool   `json:"forwardingEnableUsageReports"`        // forwarding enable usage reports
	ForwardingEnableAnalytics           *bool   `json:"forwardingEnableAnalytics"`           // forwarding enable analytics
	ForwardingEnableDiagnostics         *bool   `json:"forwardingEnableDiagnostics"`         // forwarding enable diagnostics
	ForwardingEnableStats               *bool   `json:"forwardingEnableStats"`               // forwarding enable stats
	ForwardingEnableClusterRegistration *bool   `json:"forwardingEnableClusterRegistration"` // forwarding enable cluster registration

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
