package k3s

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
)

const k3sResolvConfPath = "/etc/k3s-resolv.conf"
const fakeNameserver = "nameserver 127.0.0.1:9999"

var resolvRegexp = regexp.MustCompile(`%s*nameserver.*`)

// resolvConfigOverriden is WORKAROUND for an issue in CoreDNS with AirGap environment
func resolvConfOverriden() (bool, error) {
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

func createk3sResolvConf() error {
	f, err := os.OpenFile(k3sResolvConfPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fakeNameserver)
	return err
}
