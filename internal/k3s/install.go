package k3s

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"regexp"

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

	k3sBundleRegexp = regexp.MustCompile(`k3s-.*\.(tar(\.gz)?)|(tgz)`)
)

func init() {
	exe, _ := os.Executable()
	k3SInstallDir = path.Dir(exe)
	k3sBinary = path.Join(k3SInstallDir, "k3s")
}

type InstallConfig struct {
	Iface      string   // interface for k3s network to work on, required
	BundlePath string   // path to bundle with k3s and images
	Hostname   string   // host name of the cluster, optional, default from env
	NodeIP     []string // list of node ip addresses, optional
	ExternalIP []string // list of external ip addresses, optional
	dnsFixed   bool
}

func Install(ctx context.Context, c InstallConfig) error {
	if hasK3S() {
		return ErrExists
	}

	if err := validateNetwork(c.Iface, c.NodeIP); err != nil {
		return err
	}

	fixed, err := dnsFixed()
	if err != nil {
		return err
	}
	c.dnsFixed = fixed

	name, err := findBundle(c.BundlePath)
	if err != nil {
		return err
	}

	bundle := bundle.Tar(name)

	err = errors.Join(
		bundle.GetFiles(copyK3S, "k3s"),
		bundle.GetFiles(copyAirgapImages, "k3s-airgap-*.tar"),
		bundle.GetFiles(runInstallScript(c), "install.sh"),
	)

	if err != nil {
		cleanup()
		return err
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

func runInstallScript(c InstallConfig) func(fs.FileInfo, io.Reader) error {
	return func(fi fs.FileInfo, r io.Reader) error {
		if c.Hostname != "" {
			os.Setenv("K3S_HOSTNAME", c.Hostname)
			os.Setenv("K3S_NODE_NAME", c.Hostname)
		}
		if c.dnsFixed {
			os.Setenv("K3S_RESOLV_CONF", k3sResolvConfPath)
		}

		os.Setenv("INSTALL_K3S_BIN_DIR", k3SInstallDir)
		os.Setenv("INSTALL_K3S_SKIP_DOWNLOAD", "true")
		os.Setenv("INSTALL_K3S_SELINUX_WARN", "true")
		os.Setenv("INSTALL_K3S_SKIP_SELINUX_RPM", "true")
		os.Setenv("INSTALL_K3S_EXEC", fmt.Sprintf("--with-node-id --flannel-iface=%s --default-local-storage-path=%s", c.Iface, defaultLocalStoragePath))

		cmd := exec.Command("sh", "-")
		cmd.Stdin = r

		stdout, _ := cmd.StdoutPipe()
		go io.Copy(os.Stdout, stdout)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("install.sh: %w", err)
		}

		return nil
	}
}

var resolvRegexp = regexp.MustCompile(`%s*nameserver.*`)

func dnsFixed() (bool, error) {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Nameserver is not found, fixing...")
			return true, createk3sResolvConf()
		}
	}
	defer f.Close()

	var nameServerFound bool

	scan := bufio.NewScanner(f)

	for scan.Scan() {
		if resolvRegexp.Match(scan.Bytes()) {
			fmt.Println("Nameserver found, no fix needed")
			nameServerFound = true
			break
		}
	}

	if !nameServerFound {
		fmt.Println("Nameserver is not found, fixing...")
		return true, createk3sResolvConf()
	}

	return false, nil
}

const k3sResolvConfPath = "/etc/k3s-resolv.conf"

func createk3sResolvConf() error {
	f, err := os.OpenFile(k3sResolvConfPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("nameserver 127.0.0.1:9999")
	return err
}

func findBundle(path string) (string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	var matches []string
	for _, file := range files {
		if k3sBundleRegexp.MatchString(file.Name()) {
			matches = append(matches, file.Name())
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("k3s-*.(tar(.gz))|(tgz) bundle is not found")
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("ambigious bundle, found: %q", matches)
	}

	return matches[0], nil
}
