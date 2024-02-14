package k3s

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"
	"golang.org/x/mod/semver"

	"github.com/weka/gohomecli/internal/local/bundle"
	config_v1 "github.com/weka/gohomecli/internal/local/config/v1"
	"github.com/weka/gohomecli/internal/utils"
)

const (
	k3sInstallPath = "/opt/k3s/bin"
	dataDir        = "/opt/wekahome/data/"
)

var logger = utils.GetLogger("K3S")

var (
	ErrExists = errors.New("k3s already installed")
	ErrNMCS   = errors.New("nm-cloud-setup is enabled, please run systemctl disable nm-cloud-setup.service nm-cloud-setup.timer and reboot")

	DefaultLocalStoragePath = filepath.Join(dataDir, "local-storage")

	k3sBundleRegexp = regexp.MustCompile(`k3s.*\.(tar(\.gz)?)|(tgz)`)
	k3sImagesPath   = "/var/lib/rancher/k3s/agent/images/"
)

type Config struct {
	*config_v1.Configuration

	Iface           string // interface for k3s network to work on
	ProxyKubernetes bool   // use proxy for k3s
	Debug           bool

	ifaceAddr string // ip addr for k3s to use as node ip (taken from iface or IP)
}

func (c Config) k3sInstallArgs() []string {
	k3sArgs := []string{
		fmt.Sprintf("--flannel-iface=%s", c.Iface),
		fmt.Sprintf("--node-ip=%s", c.ifaceAddr), // node ip needs to have ip address (not 0.0.0.0)
		fmt.Sprintf("--kubelet-arg=address=%s", c.IP),
		fmt.Sprintf("--bind-address=%s", c.IP),
		fmt.Sprintf("--default-local-storage-path=%s", DefaultLocalStoragePath),
	}

	k3sArgs = append(k3sArgs, c.Configuration.K3SArgs...)

	return k3sArgs
}

func setupLogger(debug bool) {
	if debug {
		utils.SetLoggingLevel(utils.DebugLevel)
	}
}

// Wait waits for k3s to be up
func Wait(ctx context.Context) error {
	logger.Info().Msg("Waiting for k3s to be up...")

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
func setupNetwork(c *Config) (err error) {
	if c.IP == "127.0.0.1" {
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

	c.ifaceAddr = c.IP

	err = upsertIfaceAddrHost(netIF, &c.ifaceAddr, &c.Host)
	if err != nil {
		return err
	}

	return nil
}

var internalIfaces = []string{
	"cni",
	"veth",
	"flannel",
	"docker",
}

// getInterface checks if iface name is valid and running
func getInterface(iface string) (net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return net.Interface{}, err
	}

	for _, i := range ifaces {
		// skip internal k3s flannel ifaces
		internal := slices.ContainsFunc(internalIfaces, func(s string) bool {
			return strings.HasPrefix(i.Name, s)
		})

		// if it's internal flannel iface, loopback or not running - skip it
		if internal || i.Flags&net.FlagLoopback == net.FlagLoopback || i.Flags&net.FlagRunning != net.FlagRunning {
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

	var addrFound bool

	for _, a := range addr {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.To4() == nil {
			logger.Debug().Str("addr", a.String()).Msg("Not an IPv4")
			continue
		}

		if *ifaceAddr == ipnet.IP.To4().String() {
			logger.Debug().Str("addr", ipnet.IP.To4().String()).Msg("IP match to interface")
			addrFound = true
			break
		}

		if *ifaceAddr == "0.0.0.0" || *ifaceAddr == "" {
			logger.Debug().Str("addr", ipnet.IP.To4().String()).Msg("Using interface IP for NodeIP")
			// use first ip found from interface
			*ifaceAddr = ipnet.IP.To4().String()
			addrFound = true
			break
		}
	}

	if !addrFound {
		return fmt.Errorf("IP address %q is not valid", *ifaceAddr)
	}

	if *hostname == "" {
		// set IP to hostname
		logger.Warn().
			Str("hostname", *ifaceAddr).
			Msgf("Hostname is not set, using IP")

		*hostname = *ifaceAddr
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

func k3sInstall(ctx context.Context, c Config, fi fs.FileInfo, r io.Reader) error {
	logger.Info().Msg("Running k3s install script")

	if err := setupNetwork(&c); err != nil {
		return err
	}

	if c.Host != "" {
		logger.Debug().Str("hostname", c.Host).Msg("Using hostname")
		os.Setenv("K3S_HOSTNAME", c.Host)
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
	os.Setenv("K3S_NODE_NAME", "wekahome.local")

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
			fmt.Sprintf("%s/32", c.ifaceAddr),
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
}

var logRegexp = regexp.MustCompile(`(\[(.+?)\]\s*)?(.+)`)

// k3sLogParser parses log files and uses our logging system
func k3sLogParser(lvl zerolog.Level) func(lines chan []byte) {
	return func(lines chan []byte) {
		for line := range lines {
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
}

func isNmCloudSetupEnabled(ctx context.Context) bool {
	logger.Info().Msgf("Checking if nm-cloud-setup is enabled")

	var enabled bool

	cmd, err := utils.ExecCommand(ctx, "systemctl", []string{"is-active", "nm-cloud-setup"},
		utils.WithStderrLogger(logger, utils.DebugLevel),
		utils.WithStdoutReader(func(lines chan []byte) {
			for line := range lines {
				logger.Debug().Str("output", string(line)).Msg("nm-cloud-setup status")
				if string(line) == "enabled" {
					enabled = true
				}
			}
		}))

	err = errors.Join(err, cmd.Wait())
	if err != nil && cmd.ProcessState.ExitCode() != 3 { // 3 means no systemd unit exists
		logger.Debug().Err(err).Msg("systemctl exit status")
	}

	return enabled
}
