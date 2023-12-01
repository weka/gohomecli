package k3s

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/weka/gohomecli/internal/bundle"
	"github.com/weka/gohomecli/internal/utils"
)

const k3sInstallPath = "/usr/local/bin"

var logger = utils.GetLogger("K3S")

func setupLogger(debug bool) {
	if debug {
		utils.SetGlobalLoggingLevel(utils.DebugLevel)
	}
}

func k3sBinary() string {
	return filepath.Join(k3sInstallPath, "k3s")
}

func serviceCmd(action string) *exec.Cmd {
	logger.Info().Msgf("Running %s for k3s service", action)
	var cmd *exec.Cmd
	if hasSystemd() {
		cmd = exec.Command("systemctl", action, "k3s")
	} else {
		cmd = exec.Command("service", "k3s", action)
	}
	return cmd
}

func hasK3S() bool {
	_, err := os.Stat(k3sBinary())
	if err == nil {
		return true
	}
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		logger.Err(err).Msg("os.Stat error")
		// something very rare like file-system error: immediately exit
		os.Exit(255)
	}

	return true
}

func hasSystemd() bool {
	if err := exec.Command("systemctl", "status").Run(); err != nil {
		return false
	}
	return true
}

func Hostname() string {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		f, _ := os.Open("/etc/hostname")
		hostname, _ = bufio.NewReader(f).ReadString('\n')
		f.Close()
	}

	return hostname
}

// setupNetwork checks if provided nodeIP belongs to interface
// if nodeIP is empty it will write first ip from the interface into nodeIP
func setupNetwork(iface string, nodeIP *string) error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	var nodeIface *net.Interface
	for _, i := range ifaces {
		if iface == i.Name && i.Flags&net.FlagLoopback == 0 {
			nodeIface = &i
			break
		}
	}

	if nodeIface == nil {
		return fmt.Errorf("interface %q is not exists or it's loopback interface", iface)
	}

	addr, _ := nodeIface.Addrs()
	var ipMatch bool
	for _, a := range addr {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.To4() == nil {
			continue
		}

		ip := ipnet.IP.To4()
		// setup first ip as default one
		if *nodeIP == "" {
			logger.Warn().Msgf("IP not defined, using %q from %q as NodeIP\n", ip.String(), iface)
			*nodeIP = ip.String()
		}
		// check if provided node ip matched to interface
		if *nodeIP == ip.String() {
			ipMatch = true
		}
	}

	if !ipMatch {
		return fmt.Errorf("IP address for node %q not belongs to %q", *nodeIP, iface)
	}

	return nil
}

func findBundle() (filename string, manifest bundle.Manifest, err error) {
	manifest, err = bundle.GetManifest()
	if err != nil {
		return
	}

	var files []fs.DirEntry
	files, err = os.ReadDir(bundle.BundlePath())
	if err != nil {
		return
	}
	for _, file := range files {
		if k3sBundleRegexp.MatchString(file.Name()) {
			filename = path.Join(bundle.BundlePath(), file.Name())
			return
		}
	}

	err = fmt.Errorf("k3s bundle is not found")
	return
}

func getK3SVersion(binary string) (string, error) {
	cmd := exec.Command(binary, "-v")
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	var line string
	for {
		line, err = bufio.NewReader(rc).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			logger.Error().Err(err).Msg("get k3s version")
			break
		}
		if strings.HasPrefix(line, "k3s version") || errors.Is(err, io.EOF) {
			break
		}
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	version := strings.Split(line, " ")
	if len(version) < 3 || !semver.IsValid(version[2]) {
		return "", fmt.Errorf("invalid k3s version: %q", line)
	}

	return version[2], nil
}
