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
	"github.com/weka/gohomecli/internal/utils"
)

const (
	k3sInstallPath          = "/usr/local/bin"
	defaultLocalStoragePath = "/opt/local-path-provisioner"
)

var logger = utils.GetLogger("K3S")

func setupLogger(debug bool) {
	if debug {
		utils.SetLoggingLevel(utils.DebugLevel)
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
func setupNetwork(c *InstallConfig) (err error) {
	if c.BindIP == "127.0.0.1" {
		return fmt.Errorf("unable to bind to 127.0.0.1")
	}

	netIF, err := getInterface(c.Iface)
	if err != nil {
		return err
	}

	if c.Iface == "" {
		logger.Debug().
			Str("iface", netIF.Name).
			Msg("Iface is not set, using from lookup")
		// use random iface
		c.Iface = netIF.Name
	}

	c.IfaceAddr = c.BindIP

	err = upsertIfaceAddrHost(netIF, &c.IfaceAddr, &c.Host)
	if err != nil {
		return err
	}

	return nil
}

// getInterface checks if iface name is valid and running
func getInterface(iface string) (net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return net.Interface{}, err
	}

	for _, i := range ifaces {
		// if it's loopback or not running - skip it
		if i.Flags&net.FlagLoopback == net.FlagLoopback || i.Flags&net.FlagRunning != net.FlagRunning {
			logger.Debug().
				Str("iface", i.Name).
				Bool("loopback", i.Flags&net.FlagLoopback == net.FlagLoopback).
				Bool("running", i.Flags&net.FlagRunning == net.FlagRunning).
				Msg("Skipping interface")
			continue
		}

		if iface == i.Name || iface == "" {
			logger.Info().
				Str("iface", i.Name).
				Bool("loopback", i.Flags&net.FlagLoopback == net.FlagLoopback).
				Bool("running", i.Flags&net.FlagRunning == net.FlagRunning).
				Msg("Found interface for networking")

			return i, nil
		}
	}

	return net.Interface{}, fmt.Errorf("network interface %q is not running, does not exists or it's loopback interface", iface)
}

// upsertIfaceAddrHost sets any IP from iface or returns error if provided IP not match to the interface
func upsertIfaceAddrHost(iface net.Interface, ifaceAddr *string, hostname *string) error {
	addr, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("network addr: %w", err)
	}

	for _, a := range addr {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.To4() == nil {
			logger.Debug().Str("addr", a.String()).Msg("Not an IPv4")
			continue
		}
		if *ifaceAddr == ipnet.IP.To4().String() {
			logger.Debug().Str("addr", ipnet.IP.To4().String()).Msg("IP match to interface")
			return nil
		}

		if *hostname == "" {
			// set IP to hostname
			logger.Warn().
				Str("hostname", ipnet.IP.To4().String()).
				Msgf("Hostname is not set, using IP")

			*hostname = ipnet.IP.To4().String()
		}

		if *ifaceAddr == "0.0.0.0" || *ifaceAddr == "" {
			logger.Debug().Str("addr", ipnet.IP.To4().String()).Msg("Using interface IP for NodeIP")
			// use first ip found from interface
			*ifaceAddr = ipnet.IP.To4().String()
			return nil
		}
	}

	return fmt.Errorf("IP address for node %q is not exists", *ifaceAddr)
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
