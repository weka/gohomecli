package k3s

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/weka/gohomecli/internal/install/bundle"
	"github.com/weka/gohomecli/internal/utils"
)

var (
	ErrExists               = errors.New("k3s already installed")
	defaultLocalStoragePath = "/opt/local-path-provisioner"
	k3sBundleRegexp         = regexp.MustCompile(`k3s.*\.(tar(\.gz)?)|(tgz)`)
)

type InstallConfig struct {
	Iface       string   // interface for k3s network to work on, required
	Hostname    string   // host name of the cluster, optional, default from env
	NodeIP      string   // node ip to bind on as primary internal ip
	ExternalIPs []string // list of external ip addresses, optional
	TLS         TLSConfig
	Debug       bool
}

func (c InstallConfig) k3sInstallArgs() string {
	k3sArgs := []string{
		fmt.Sprintf("--node-ip=%s", c.NodeIP),
		fmt.Sprintf("--flannel-iface=%s", c.Iface),
		fmt.Sprintf("--default-local-storage-path=%s", defaultLocalStoragePath),
	}

	if len(c.ExternalIPs) > 0 {
		k3sArgs = append(k3sArgs, fmt.Sprintf("--node-external-ip=%s", strings.Join(c.ExternalIPs, ",")))
	}

	return strings.Join(k3sArgs, " ")
}

// Install runs K3S installation process
func Install(ctx context.Context, c InstallConfig) error {
	setupLogger(c.Debug)

	if hasK3S() && !c.Debug {
		return ErrExists
	}

	if err := setupNetwork(c.Iface, &c.NodeIP); err != nil {
		return err
	}

	name, manifest, err := findBundle()
	if err != nil {
		return err
	}

	logger.Info().Msgf("Installing K3S %q\n", manifest.K3S)

	bundle := bundle.Tar(name)

	err = bundle.GetFiles(ctx, copyK3S(), copyAirgapImages(), runInstallScript(c))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info().Msg("Setup was cancelled")
			cleanup(false)
			return nil
		}

		cleanup(c.Debug)
		return err
	}

	err = setupTLS(ctx, c.TLS)
	if err != nil && !errors.Is(err, ErrNoTLS) {
		cleanup(c.Debug)
		return err
	}

	return nil
}

// cleanup runs k3s-uninstall and removes copied files
// if debug flag is not enabled
func cleanup(debug bool) {
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

func runInstallScript(c InstallConfig) bundle.TarCallback {
	return bundle.TarCallback{
		FileName: "install.sh",

		Callback: func(ctx context.Context, fi fs.FileInfo, r io.Reader) error {
			logger.Info().Msg("Starting k3s install")

			if c.Hostname != "" {
				os.Setenv("K3S_HOSTNAME", c.Hostname)
				os.Setenv("K3S_NODE_NAME", c.Hostname)
			}
			overriden, err := resolvConfOverriden()
			if err != nil {
				return err
			}
			if overriden {
				os.Setenv("K3S_RESOLV_CONF", k3sResolvConfPath)
			}
			os.Setenv("INSTALL_K3S_BIN_DIR", k3sInstallPath)
			os.Setenv("INSTALL_K3S_SKIP_DOWNLOAD", "true")
			os.Setenv("INSTALL_K3S_SELINUX_WARN", "true")
			os.Setenv("INSTALL_K3S_SKIP_SELINUX_RPM", "true")
			os.Setenv("INSTALL_K3S_EXEC", c.k3sInstallArgs())

			cmd, err := utils.ExecCommand(ctx, "bash", []string{"-"},
				utils.WithStdin(r),
				utils.WithStdoutReader(k3sLogParser(utils.InfoLevel)),
				utils.WithStderrReader(k3sLogParser(utils.InfoLevel)),
			)
			if err != nil {
				return err
			}

			if err = cmd.Wait(); err != nil {
				return fmt.Errorf("install.sh: %w", errors.Join(err, ctx.Err()))
			}

			logger.Info().Msg("Install completed")

			return nil
		},
	}
}
