package k3s

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
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

func validateNetwork(iface string, nodeIPs []string) error {
	var (
		ifaceExists bool
		ips         = make(map[string]bool)
	)
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, i := range ifaces {
		if iface != i.Name {
			continue
		}
		ifaceExists = true

		addr, _ := i.Addrs()
		for _, a := range addr {
			ips[a.(*net.IPNet).IP.String()] = true
		}
	}

	if !ifaceExists {
		return fmt.Errorf("interface %q is not exists", iface)
	}

	var notExists []string
	for _, ip := range nodeIPs {
		if _, found := ips[net.ParseIP(ip).String()]; !found {
			notExists = append(notExists, ip)
		}
	}

	if len(notExists) > 0 {
		return fmt.Errorf("wrong ip addresses provided: %q, available: %v", notExists, ips)
	}

	return nil
}
