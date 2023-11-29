package k3s

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"

	"golang.org/x/mod/semver"
)

func serviceCmd(action string) *exec.Cmd {
	var cmd *exec.Cmd
	if hasSystemd() {
		cmd = exec.Command("systemctl", action, "k3s")
	} else {
		cmd = exec.Command("service", "k3s", action)
	}
	return cmd
}

func hasK3S() bool {
	_, err := os.Stat(k3sBinary)
	if err == nil {
		return true
	}
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("os.Stat: ", err)
		os.Exit(255)
	}
	// check k3s in PATH
	err = exec.Command("k3s").Run()
	return err == nil
}

func hasSystemd() bool {
	if err := exec.Command("systemctl", "status").Run(); err != nil {
		return false
	}
	return true
}

func Hostname() string {
	var hostname = os.Getenv("HOSTNAME")
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
			fmt.Printf("IP not defined, using %q from %q as NodeIP\n", ip.String(), iface)
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

func findBundle(dir string) (filename string, version string, err error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", "", err
	}

	var matches []string

	for _, file := range files {
		if k3sBundleRegexp.MatchString(file.Name()) {
			matches = append(matches, file.Name())
			version = k3sBundleRegexp.FindStringSubmatch(file.Name())[1]
		}
	}

	if len(matches) == 0 {
		return "", "", fmt.Errorf("k3s-*.(tar(.gz))|(tgz) bundle is not found")
	}

	if len(matches) > 1 {
		return "", "", fmt.Errorf("ambigious bundle, found: %q", matches)
	}

	if !semver.IsValid(version) {
		return "", "", fmt.Errorf("unable to parse version %q", version)
	}

	return path.Join(dir, matches[0]), semver.Canonical(version), nil
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
	line, err := bufio.NewReader(rc).ReadString('\n')
	if err != nil {
		return "", err
	}
	if err := cmd.Wait(); err != nil {
		return "", err
	}

	version := strings.Split(line, " ")[2]
	if !semver.IsValid(version) {
		return "", fmt.Errorf("invalid k3s version: %q", version)
	}

	return semver.Canonical(version), nil
}
