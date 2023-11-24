package k3s

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"

	"github.com/weka/gohomecli/internal/bundle"
)

var (
	ErrExists = errors.New("k3s already installed")
)

type Config struct {
	BundlePath string   // path to bundle with k3s and images
	Iface      string   // interface for k3s network to work on, required
	Hostname   string   // host name of the cluster, optional, default from env
	NodeIP     []string // list of node ip addresses, optional
	ExternalIP []string // list of external ip addresses, optional
}

func Install(ctx context.Context, c Config) error {
	if hasK3S() {
		return ErrExists
	}

	bundle := bundle.Tar(c.BundlePath)

	err := errors.Join(
		bundle.GetFiles(copyK3S, "k3s"),
		bundle.GetFiles(copyAirgapImages, "airgap-*"),
		bundle.GetFiles(runInstallScript(c), "install.sh"),
	)

	if err != nil {
		// TODO: cleanup
		return err
	}

	return nil
}

func Upgrade(ctx context.Context) error {
	fmt.Println("Starting K3S upgrade...")

	if hasSystemd() {
		fmt.Println("Stopping K3S service...")
		cmd := exec.Command("systemctl", "stop", "k3s")
		if err := cmd.Run(); err != nil {
			fmt.Println("Failed to stop K3S service: ", err)
		}
	}

	return nil
}

func Uninstall() error {
	return nil
}

func hasK3S() bool {
	_, err := os.Stat("/usr/local/bin/k3s")
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		fmt.Println("os.Open: ", err)
		os.Exit(1)
		return false
	}
	return true
}

func hasKubectl() bool {
	return false
}

func hasSystemd() bool {
	cmd := exec.Command("systemctl", "status")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func copyK3S(_ fs.FileInfo, r io.Reader) error {
	f, err := os.OpenFile("/usr/local/bin/k3s", os.O_CREATE|os.O_WRONLY, fs.FileMode(0755))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	if err != nil && !errors.Is(err, io.EOF) {
		err = errors.Join(err, f.Close())
		err = errors.Join(err, os.Remove("/usr/local/bin/k3s"))
		return err
	}
	return nil
}

func copyAirgapImages(info fs.FileInfo, r io.Reader) error {
	os.MkdirAll("/var/lib/rancher/k3s/agent/images/", 0644)

	f, err := os.OpenFile("/var/lib/rancher/k3s/agent/images/"+info.Name(), os.O_CREATE|os.O_WRONLY, fs.FileMode(0644))
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if err != nil && !errors.Is(err, io.EOF) {
		err = errors.Join(err, f.Close())
		err = errors.Join(err, os.Remove("/var/lib/rancher/k3s/agent/images/"+info.Name()))
		return err
	}
	return nil
}

func runInstallScript(c Config) func(fs.FileInfo, io.Reader) error {
	return func(fi fs.FileInfo, r io.Reader) error {
		os.Setenv("K3S_HOSTNAME", "$HOSTNAME")
		os.Setenv("K3S_NODE_NAME", "$K3S_HOSTNAME")
		os.Setenv("INSTALL_K3S_SKIP_DOWNLOAD", "true")
		os.Setenv("INSTALL_K3S_SELINUX_WARN", "true")
		os.Setenv("INSTALL_K3S_SKIP_SELINUX_RPM", "true")
		os.Setenv("INSTALL_K3S_EXEC", fmt.Sprintf("--with-node-id --flannel-iface=%s --default-local-storage-path=/opt/local-path-provisioner", c.Iface))

		cmd := exec.Command("sh", "-")
		cmd.Stdin = r

		stdout, _ := cmd.StdoutPipe()
		go io.Copy(os.Stdout, stdout)
		return cmd.Run()
	}
}
