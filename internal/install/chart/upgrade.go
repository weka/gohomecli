package chart

import (
	"context"
	"fmt"

	helmclient "github.com/mittwald/go-helm-client"
	"github.com/weka/gohomecli/internal/utils"
)

func Upgrade(ctx context.Context, cfg *Configuration, opts *HelmOptions) error {
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
		Msg("Upgrading chart")

	upgradeOpts := &helmclient.GenericHelmOptions{
		RollBack: client,
	}
	// in debug mode we don't do rollback
	if !cfg.Debug {
		upgradeOpts = nil
	}

	release, err := client.UpgradeChart(ctx, spec, upgradeOpts)
	if err != nil {
		logger.Error().Err(err).Msg("Upgrade failed")
		return err
	}

	logger.Info().Msg(release.Info.Notes)

	return nil
}
