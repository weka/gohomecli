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

	"github.com/weka/gohomecli/internal/bundle"
)

const k3sImagesPath = "/var/lib/rancher/k3s/agent/images/"
const defaultLocalStoragePath = "/opt/local-path-provisioner"

var (
	ErrExists = errors.New("k3s already installed")
)

var (
	k3SInstallDir string
	k3sBinary     string

	k3sBundleRegexp = regexp.MustCompile(`k3s-(.*)\.(tar(\.gz)?)|(tgz)`)
)

func init() {
	// by default install to current directory
	exe, _ := os.Executable()
	k3SInstallDir = path.Dir(exe)
	k3sBinary = path.Join(k3SInstallDir, "k3s")
}

type InstallConfig struct {
	Iface       string   // interface for k3s network to work on, required
	BundlePath  string   // path to bundle with k3s and images
	Hostname    string   // host name of the cluster, optional, default from env
	NodeIP      string   // node ip to bind on as primary internal ip
	ExternalIPs []string // list of external ip addresses, optional
}

// Install runs K3S installation process
func Install(ctx context.Context, c InstallConfig) error {
	if hasK3S() {
		return ErrExists
	}

	if err := setupNetwork(c.Iface, &c.NodeIP); err != nil {
		return err
	}

	name, version, err := findBundle(c.BundlePath)
	if err != nil {
		return err
	}

	bundle := bundle.Tar(name)

	fmt.Printf("Installing K3S %q\n", version)

	err = errors.Join(
		bundle.GetFiles(copyK3S, "k3s"),
		bundle.GetFiles(copyAirgapImages, "k3s-airgap-*.tar*"),
		bundle.GetFiles(runInstallScript(ctx, c), "install.sh"),
	)

	if err != nil {
		cleanup()
		return err
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		cleanup()
		return ctx.Err()
	}

	return nil
}

func cleanup() error {
	exec.Command("k3s-uninstall.sh").Run()
	os.RemoveAll(k3sImagesPath)
	os.Remove(k3sBinary)
	os.Remove(k3sResolvConfPath)
	return nil
}

func copyK3S(_ fs.FileInfo, r io.Reader) error {
	f, err := os.OpenFile(k3sBinary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fs.FileMode(0755))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	if err != nil && !errors.Is(err, io.EOF) {
		err = errors.Join(err, f.Close())
		err = errors.Join(err, os.Remove(k3sBinary))
		return err
	}
	return nil
}

func copyAirgapImages(info fs.FileInfo, r io.Reader) error {
	os.MkdirAll(k3sImagesPath, 0644)

	f, err := os.OpenFile(path.Join(k3sImagesPath, info.Name()), os.O_CREATE|os.O_WRONLY, fs.FileMode(0644))
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if err != nil && !errors.Is(err, io.EOF) {
		err = errors.Join(err, f.Close())
		err = errors.Join(err, os.Remove(path.Join(k3sImagesPath, info.Name())))
		return err
	}
	return nil
}

func runInstallScript(ctx context.Context, c InstallConfig) func(fs.FileInfo, io.Reader) error {
	return func(fi fs.FileInfo, r io.Reader) error {
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
		os.Setenv("INSTALL_K3S_BIN_DIR", k3SInstallDir)
		os.Setenv("INSTALL_K3S_SKIP_DOWNLOAD", "true")
		os.Setenv("INSTALL_K3S_SELINUX_WARN", "true")
		os.Setenv("INSTALL_K3S_SKIP_SELINUX_RPM", "true")

		k3sArgs := []string{
			// binding node to localhost only
			fmt.Sprintf("--node-ip=%s", c.NodeIP),
			fmt.Sprintf("--flannel-iface=%s", c.Iface),
			fmt.Sprintf("--default-local-storage-path=%s", defaultLocalStoragePath),
		}
		if len(c.ExternalIPs) > 0 {
			k3sArgs = append(k3sArgs, fmt.Sprintf("--node-external-ip=%s", strings.Join(c.ExternalIPs, ",")))
		}

		os.Setenv("INSTALL_K3S_EXEC", strings.Join(k3sArgs, " "))

		cmd := exec.CommandContext(ctx, "sh", "-")
		cmd.Stdin = r

		stdout, _ := cmd.StdoutPipe()
		go io.Copy(os.Stdout, stdout)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("install.sh: %w", err)
		}

		return nil
	}
}
