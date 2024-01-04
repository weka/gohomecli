package k3s

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog"
	"golang.org/x/mod/semver"

	"github.com/weka/gohomecli/internal/local/bundle"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/utils"
)

const (
	k3sInstallPath          = "/usr/local/bin"
	defaultLocalStoragePath = "/opt/local-path-provisioner"
)

var logger = utils.GetLogger("K3S")

func setupLogger(debug bool) {
	if debug {
		utils.SetGlobalLoggingLevel(utils.DebugLevel)
	}
}

// Wait waits for k3s to be up
func Wait(ctx context.Context) error {
	waitScript := `until [[ $(k3s ctr info) ]]; do sleep 5; done`

	cmd, err := utils.ExecCommand(ctx, "bash", []string{"-"},
		utils.WithStdin(strings.NewReader(waitScript)),
		utils.WithStderrLogger(logger, utils.DebugLevel))
	if err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("kubectl wait: %w", err)
	}
	return nil
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

// setupNetwork checks if provided nodeIP belongs to interface
// if nodeIP is empty it will write first ip from the interface into nodeIP
func setupNetwork(iface string, c *config_v1.Configuration) (err error) {
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
		if c.NodeIP == "" {
			logger.Warn().Msgf("IP not defined, using %q from %q as NodeIP\n", ip.String(), iface)
			c.NodeIP = ip.String()
		}
		// check if provided node ip matched to interface
		if c.NodeIP == ip.String() {
			ipMatch = true
		}
	}

	if !ipMatch {
		return fmt.Errorf("IP address for node %q not belongs to %q", c.NodeIP, iface)
	}

	if c.Host == "" {
		logger.Warn().Msgf("Hostname is not set, using %q as Hostname", c.NodeIP)
		c.Host = c.NodeIP
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

var logRegexp = regexp.MustCompile(`(\[(.+?)\]\s*)?(.+)`)

// k3sLogParser parses log files and uses our logging system
func k3sLogParser(lvl zerolog.Level) func(line []byte) {
	return func(line []byte) {
		// parse log level if present, otherwise log full line
		matches := logRegexp.FindSubmatch(line)
		if matches == nil {
			logger.WithLevel(lvl).Msg(string(line))
			return
		}

		parsedLvl, _ := zerolog.ParseLevel(strings.ToLower(string(matches[2])))
		if parsedLvl != zerolog.NoLevel {
			lvl = parsedLvl
		}

		logger.WithLevel(lvl).Msg(string(matches[3]))
	}
}
