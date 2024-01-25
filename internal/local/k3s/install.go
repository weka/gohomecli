package k3s

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/weka/gohomecli/internal/local/bundle"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/utils"
)

const dataDir = "/opt/wekahome/data/"

var (
	ErrExists = errors.New("k3s already installed")

	DefaultLocalStoragePath = filepath.Join(dataDir, "local-storage")

	k3sBundleRegexp = regexp.MustCompile(`k3s.*\.(tar(\.gz)?)|(tgz)`)
	k3sImagesPath   = "/var/lib/rancher/k3s/agent/images/"
)

type InstallConfig struct {
	*config_v1.Configuration

	Iface     string // interface for k3s network to work on
	IfaceAddr string // ip addr for k3s to use as node ip

	ProxyKubernetes bool // use proxy for k3s

	Debug bool
}

func (c InstallConfig) k3sInstallArgs() []string {
	k3sArgs := []string{
		fmt.Sprintf("--flannel-iface=%s", c.Iface),
		fmt.Sprintf("--node-ip=%s", c.IfaceAddr), // node ip needs to have ip address (not 0.0.0.0)
		fmt.Sprintf("--kubelet-arg=address=%s", c.IP),
		fmt.Sprintf("--bind-address=%s", c.IP),
		fmt.Sprintf("--default-local-storage-path=%s", DefaultLocalStoragePath),
	}

	k3sArgs = append(k3sArgs, c.Configuration.K3SArgs...)

	return k3sArgs
}

// Install runs K3S installation process
func Install(ctx context.Context, c InstallConfig) error {
	setupLogger(c.Debug)

	if hasK3S() && !c.Debug {
		return ErrExists
	}

	name, manifest, err := findBundle()
	if err != nil {
		return err
	}

	logger.Info().Msgf("Installing K3S %q\n", manifest.K3S)

	if err := setupNetwork(&c); err != nil {
		return err
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

	err = setupTLS(ctx, c.Configuration)
	if err != nil && !errors.Is(err, ErrNoTLS) {
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

			if c.Host != "" {
				logger.Debug().Str("hostname", c.Host).Msg("Using hostname")
				os.Setenv("K3S_HOSTNAME", c.Host)
				os.Setenv("K3S_NODE_NAME", c.Host)
			}

			overriden, err := resolvConfOverriden()
			if err != nil {
				return err
			}
			if overriden {
				logger.Debug().Str("resolvconf", k3sResolvConfPath).Msg("Resolv.conf is overriden")
				os.Setenv("K3S_RESOLV_CONF", k3sResolvConfPath)
			}

			logger.Debug().
				Str("installPath", k3sInstallPath).
				Msg("Setting env vars")

			os.Setenv("INSTALL_K3S_BIN_DIR", k3sInstallPath)
			os.Setenv("INSTALL_K3S_SKIP_DOWNLOAD", "true")
			os.Setenv("INSTALL_K3S_SELINUX_WARN", "true")
			os.Setenv("INSTALL_K3S_SKIP_SELINUX_RPM", "true")

			if c.Proxy.URL != "" && c.ProxyKubernetes {
				proxyURL, err := url.Parse(c.Proxy.URL)
				if err != nil {
					return fmt.Errorf("url parse: %w", err)
				}

				logger.Info().
					Str("proxy", utils.URLSafe(proxyURL).String()).
					Msg("Using proxy")

				var noProxy = []string{
					"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
					fmt.Sprintf("%s/32", c.IP),
					fmt.Sprintf("%s/32", c.IfaceAddr),
				}

				os.Setenv("NO_PROXY", strings.Join(noProxy, ","))

				switch proxyURL.Scheme {
				case "http":
					os.Setenv("HTTPS_PROXY", proxyURL.String())
					os.Setenv("HTTP_PROXY", proxyURL.String())
				case "https":
					os.Setenv("HTTPS_PROXY", proxyURL.String())
				default:
					logger.Warn().
						Str("url", proxyURL.String()).
						Msgf("Proxy scheme %s is not supported with K3S", proxyURL.Scheme)
				}
			}

			cmd, err := utils.ExecCommand(ctx, "sh", append([]string{"-s", "-", "server"}, c.k3sInstallArgs()...),
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
