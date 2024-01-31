package chart

import (
	"context"
	"errors"
	"fmt"
	"strings"

	helmclient "github.com/mittwald/go-helm-client"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/weka/gohomecli/internal/utils"
)

var ErrTimeout = errors.New("timeout")

func Install(ctx context.Context, opts *HelmOptions) error {
	namespace := ReleaseNamespace
	if opts.NamespaceOverride != "" {
		namespace = opts.NamespaceOverride
	}

	go watchEvents(ctx, opts)

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
			Output: utils.NewWritterFunc(func(b []byte) {
				logger.Info().Msg(string(b))
			}),
		},
		KubeContext: opts.KubeContext,
		KubeConfig:  opts.KubeConfig,
	})
	if err != nil {
		return fmt.Errorf("failed configuring helm client: %w", err)
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

	release, err := client.InstallChart(ctx, spec, nil)
	if err != nil {
		if isTimeoutErr(err) {
			return fmt.Errorf("failed installing chart: %w", ErrTimeout)
		}

		return fmt.Errorf("failed installing chart: %w", err)
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
	case strings.Contains(err.Error(), "would exceed context deadline"):
		// there is no dedicated error in kubernetes rate limiter
		// so we need to check the error message
		return true
	}

	return false
}
