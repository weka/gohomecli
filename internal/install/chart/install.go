package chart

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/weka/gohomecli/internal/install/bundle"
	"github.com/weka/gohomecli/internal/utils"
)

func Install(ctx context.Context, cfg *Configuration, opts *HelmOptions) error {
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

	spec, err := chartSpec(client, cfg, opts)
	if err != nil {
		return fmt.Errorf("failed to prepare chart spec: %w", err)
	}

	logger.Info().
		Str("namespace", spec.Namespace).
		Str("chart", spec.ChartName).
		Str("release", spec.ReleaseName).
		Msg("Installing chart")

	release, err := client.InstallChart(ctx, spec, nil)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info().Msg("Chart installation was cancelled")
			return nil
		}
		return fmt.Errorf("failed installing chart: %w", err)
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
