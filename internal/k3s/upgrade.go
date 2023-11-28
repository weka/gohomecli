package k3s

import (
	"context"
	"errors"
	"fmt"

	"github.com/weka/gohomecli/internal/bundle"
	"golang.org/x/mod/semver"
)

var ErrNotExist = errors.New("k3s not exists")

type UpgradeConfig struct {
	BundlePath string
}

func Upgrade(ctx context.Context, c UpgradeConfig) error {
	if !hasK3S() {
		return ErrNotExist
	}

	file, version, err := findBundle(c.BundlePath)
	if err != nil {
		return err
	}

	curVersion, err := getK3SVersion(k3sBinary)
	if err != nil {
		return err
	}

	fmt.Printf("Found k3s bundle %q, current version %q\n", version, curVersion)

	if semver.Compare(version, curVersion) < 0 {
		fmt.Println("Downgrading kubernetes cluster is not possible")
		return nil
	}

	fmt.Println("Starting K3S upgrade...")
	fmt.Println("Stopping k3s service")
	if err := serviceCmd("stop").Run(); err != nil {
		return fmt.Errorf("stop K3S service: %w", err)
	}

	fmt.Println("Copying new k3s image...")
	bundle := bundle.Tar(file)

	err = errors.Join(
		bundle.GetFiles(copyK3S, "k3s"),
		bundle.GetFiles(copyAirgapImages, "k3s-airgap*.tar*"),
	)

	if err != nil {
		return err
	}

	fmt.Println("Starting new k3s service...")
	if err := serviceCmd("start").Run(); err != nil {
		return fmt.Errorf("start K3S service: %w", err)
	}

	fmt.Println("Upgrade completed successfully")

	return nil
}
