package k3s

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/mod/semver"

	"github.com/weka/gohomecli/internal/bundle"
	"github.com/weka/gohomecli/internal/utils"
)

var ErrNotExist = errors.New("k3s not exists")

type UpgradeConfig struct {
	BundlePath string
	Debug      bool
}

func Upgrade(ctx context.Context, c UpgradeConfig) error {
	if !hasK3S() {
		return ErrNotExist
	}

	if c.Debug {
		logger.Level(utils.DebugLevel)
	}

	if c.BundlePath != "" {
		err := bundle.SetBundlePath(c.BundlePath)
		if err != nil {
			return err
		}
	}

	file, manifest, err := findBundle()
	if err != nil {
		return err
	}

	curVersion, err := getK3SVersion(k3sBinary())
	if err != nil {
		return err
	}

	logger.Info().Msgf("Found k3s bundle %q, current version %q\n", manifest.K3S, curVersion)

	if semver.Compare(manifest.K3S, curVersion) == -1 && !c.Debug {
		logger.Error().Msg("Downgrading kubernetes cluster is not possible")
		return nil
	}

	logger.Info().Msg("Starting K3S upgrade...")
	if err := serviceCmd("stop").Run(); err != nil {
		return fmt.Errorf("stop K3S service: %w", err)
	}

	logger.Info().Msg("Copying new k3s image...")
	bundle := bundle.Tar(file)

	err = errors.Join(
		bundle.GetFiles(copyK3S, "k3s"),
		bundle.GetFiles(copyAirgapImages, "k3s-airgap*.tar*"),
	)

	if err != nil {
		return err
	}

	if err := serviceCmd("start").Run(); err != nil {
		return fmt.Errorf("start K3S service: %w", err)
	}

	logger.Info().Msg("Upgrade completed successfully")

	return nil
}
