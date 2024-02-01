package k3s

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/mod/semver"

	"github.com/weka/gohomecli/internal/local/bundle"
)

var ErrNotExist = errors.New("k3s not exists")

func Upgrade(ctx context.Context, c Config) (retErr error) {
	setupLogger(c.Debug)

	if !hasK3S() {
		return ErrNotExist
	}

	logger.Debug().Msgf("Looking for bundle")

	file, manifest, err := findBundle()
	if err != nil {
		return fmt.Errorf("find bundle: %w", err)
	}

	logger.Debug().Msg("Parsing K3S version")
	curVersion, err := getK3SVersion(k3sBinary())
	if err != nil {
		return fmt.Errorf("get k3s version: %w", err)
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

	backupFiles, err := backupK3S()
	if err != nil {
		if !c.Debug {
			return fmt.Errorf("backup k3s: %w", err)
		}
		logger.Warn().Err(err).Msg("Backing up old K3S failed, doing upgrade anyway...")
	}
	defer func() {
		if retErr != nil && len(backupFiles) > 0 && !c.Debug {
			retErr = errors.Join(retErr, restore(backupFiles))
		}
		if err := serviceCmd("start").Run(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("start K3S service: %w", err))
		}
	}()

	logger.Info().Msg("Copying new k3s image...")
	err = bundle.Tar(file).GetFiles(ctx, copyK3S(), copyAirgapImages(), runInstallScript(c))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Warn().Msg("Upgrade was cancelled")
			return err
		}
		return fmt.Errorf("read bundle: %w", err)
	}

	err = setupTLS(ctx, c.Configuration)
	if err != nil && !errors.Is(err, ErrNoTLS) {
		return err
	}

	logger.Info().Msg("Upgrade completed")

	return nil
}
