package chart

import (
	"context"
	"errors"
	"fmt"
)

func Upgrade(ctx context.Context, opts *HelmOptions, debug bool) error {
	client, err := NewHelmClient(ctx, opts)
	if err != nil {
		return fmt.Errorf("helm client: %w", err)
	}

	spec, err := chartSpec(client, opts)
	if err != nil {
		return fmt.Errorf("failed to prepare chart spec: %w", err)
	}

	logger.Info().
		Str("namespace", spec.Namespace).
		Str("chart", spec.ChartName).
		Str("release", spec.ReleaseName).
		Msg("Upgrading chart")

	warnings, cancel, err := watchWarningEvents(ctx, spec.Namespace, opts.KubeConfig)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to watch events")
	}

	release, err := client.UpgradeChart(ctx, spec, nil)

	if cancel != nil {
		cancel()
	}

	if err != nil {
		// if it's not canceled, print warnings
		if !errors.Is(err, context.Canceled) && len(warnings) > 0 {
			logger.Info().Msg("Received next warnings:")
			for warn := range warnings {
				logger.Warn().Str("name", warn.Name).Msg(warn.Message)
			}
		}

		if !debug {
			logger.Warn().Msg("Rolling back release")
			if err := client.RollbackRelease(spec); err != nil {
				logger.Error().Err(err).Msg("Rollback failed")
			}
		}

		if isTimeoutErr(err) {
			return errors.Join(ErrTimeout, err)
		}

		logger.Error().Err(err).Msg("Upgrade failed")
		return err
	}

	logger.Info().Msg(release.Info.Notes)

	return nil
}
