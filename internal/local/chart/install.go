package chart

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/wait"
)

var ErrTimeout = errors.New("timeout")

func Install(ctx context.Context, opts *HelmOptions) error {
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
		Msg("Installing chart")

	warnings, cancel, err := watchWarningEvents(ctx, spec.Namespace, opts.KubeConfig)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to watch events")
	}

	release, err := client.InstallChart(ctx, spec, nil)

	// stop watching for events
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

		if isTimeoutErr(err) {
			return errors.Join(ErrTimeout, err)
		}

		return err
	}

	logger.Info().Msg(release.Info.Notes)

	return nil
}

// isTimeoutErr returns true if err looks like timeout error
func isTimeoutErr(err error) bool {
	switch {
	case wait.Interrupted(err):
		// kubernetes client returns interrupted error
		// for both context.Cancelled and timeout error, we need only the one
		if errors.Is(err, context.Canceled) {
			return false
		}
		return true
	case strings.Contains(err.Error(), "context deadline"):
		// there is no dedicated error in kubernetes rate limiter
		// so we need to check the error message
		return true
	}

	return false
}
