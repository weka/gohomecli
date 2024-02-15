package k3s

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"

	"github.com/weka/gohomecli/internal/local/bundle"
)

// Install runs K3S installation process
func Install(ctx context.Context, c Config) error {
	setupLogger(c.Debug)

	if hasK3S() && !c.Debug {
		return ErrExists
	}

	if isNmCloudSetupEnabled(ctx) {
		return ErrNMCS
	}

	name, manifest, err := findBundle()
	if err != nil {
		return err
	}

	logger.Info().Msgf("Installing K3S %q\n", manifest.K3S)

	switch {
	case isFirewallActive(ctx, FirewallTypeFirewalld):
		if err := addFirewallRules(ctx, FirewallTypeFirewalld); err != nil {
			logger.Warn().Err(err).Msg("Failed to add firewalld rules")
		}
	case isFirewallActive(ctx, FirewallTypeUFW):
		if err := addFirewallRules(ctx, FirewallTypeFirewalld); err != nil {
			logger.Warn().Err(err).Msg("Failed to add UFW rules")
		}
	default:
		logger.Warn().Msg("No supported firewall found, skipping firewall rules setup. Please make sure to open required ports manually")
	}

	bundle := bundle.Tar(name)

	err = bundle.GetFiles(ctx, copyK3S(), copyAirgapImages(), runInstallScript(c))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info().Msg("Setup was cancelled")
			return nil
		}
		return err
	}

	return nil
}

// Cleanup runs k3s-uninstall and removes copied files
// if debug flag is not enabled
func Cleanup(debug bool) {
	if !debug {
		logger.Info().Msg("Cleaning up installation")

		exec.Command("k3s-uninstall.sh").Run()
		os.RemoveAll(k3sImagesPath)
		os.Remove(k3sBinary())
		os.Remove(k3sResolvConfPath)
	}
}

func copyK3S() bundle.TarCallback {
	return bundle.TarCallback{
		FileName: "k3s",

		Callback: func(ctx context.Context, _ fs.FileInfo, r io.Reader) (err error) {
			logger.Info().Msg("Copying k3s binary")

			_ = os.MkdirAll(k3sInstallPath, 0o755)

			f, err := os.OpenFile(k3sBinary(), os.O_CREATE|os.O_WRONLY, fs.FileMode(0o755))
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, r)
			if err != nil {
				f.Close()
				os.Remove(k3sBinary())
				return err
			}

			return nil
		},
	}
}

func copyAirgapImages() bundle.TarCallback {
	return bundle.TarCallback{
		FileName: "k3s-airgap-*.tar*",

		Callback: func(ctx context.Context, info fs.FileInfo, r io.Reader) (err error) {
			logger.Info().Msg("Copying airgap images")

			os.MkdirAll(k3sImagesPath, 0o644)

			var f *os.File
			f, err = os.OpenFile(path.Join(k3sImagesPath, info.Name()), os.O_CREATE|os.O_WRONLY, fs.FileMode(0o644))
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, r)
			if err != nil {
				err = errors.Join(err, os.Remove(path.Join(k3sImagesPath, info.Name())))
				return err
			}

			return nil
		},
	}
}

func runInstallScript(c Config) bundle.TarCallback {
	return bundle.TarCallback{
		FileName: "install.sh",
		Callback: func(ctx context.Context, fi fs.FileInfo, r io.Reader) error {
			return k3sInstall(ctx, c, fi, r)
		},
	}
}
