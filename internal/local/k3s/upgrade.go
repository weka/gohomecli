package k3s

import (
	"context"
	"errors"
	"fmt"
	"net"

	"golang.org/x/mod/semver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/weka/gohomecli/internal/local/bundle"
	"github.com/weka/gohomecli/internal/local/chart"
)

var ErrNotExist = errors.New("k3s not exists")

func Upgrade(ctx context.Context, c Config) (retErr error) {
	setupLogger(c.Debug)

	if !hasK3S() {
		return ErrNotExist
	}
	if isNmCloudSetupEnabled(ctx) {
		return ErrNMCS
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
	c.IPv4Only, err = IsClusterIPv4Only(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("explore existing cluster")
		return err
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

	logger.Info().Msg("Copying new k3s image...")
	err = bundle.Tar(file).GetFiles(ctx, copyK3S(), copyAirgapImages(), runInstallScript(c))
	if err != nil {
		// restoring backup
		if len(backupFiles) > 0 && !c.Debug {
			err = errors.Join(err, restore(backupFiles))
			if startErr := serviceCmd("start").Run(); startErr != nil {
				err = errors.Join(err, fmt.Errorf("start K3S service: %w", startErr))
			}
		}

		if errors.Is(err, context.Canceled) {
			logger.Warn().Msg("Upgrade was cancelled")
			return err
		}

		return err
	}

	if err := serviceCmd("start").Run(); err != nil {
		return fmt.Errorf("start K3S service: %w", err)
	}

	logger.Info().Msg("K3S upgrade completed")

	return nil
}

func IsClusterIPv4Only(ctx context.Context) (bool, error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", chart.KubeConfigPath)
	if err != nil {
		return false, fmt.Errorf("read Kubeconfig err: %w", err)
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return false, fmt.Errorf("create k8s client err: %w", err)
	}
	node, err := client.CoreV1().Nodes().Get(ctx, "wekahome.local", metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("get node wekahome.local err: %w", err)
	}

	cidrs := node.Spec.PodCIDRs
	for _, cidr := range cidrs {
		logger.Debug().Str("CIDR", cidr).Msg("Node pod CIDR")
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return false, fmt.Errorf("parse CIDR [%s] err: %w", cidr, err)
		}
		if ip.To4() == nil { // cluster support IPv6
			logger.Debug().Str("IP", ip.String()).Msg("cluster supports IPv6")
			return false, nil
		}
	}
	logger.Debug().Msg("cluster doesn't support IPv6")
	return true, nil
}
