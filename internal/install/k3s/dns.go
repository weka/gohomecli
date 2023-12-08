package k3s

import (
	"bufio"
	"errors"
	"os"
	"regexp"
)

const (
	k3sResolvConfPath = "/etc/k3s-resolv.conf"
	fakeNameserver    = "nameserver 127.0.0.1:9999"
)

var resolvRegexp = regexp.MustCompile(`%s*nameserver.*`)

// resolvConfigOverriden is WORKAROUND for an issue in CoreDNS with AirGap environment
func resolvConfOverriden() (bool, error) {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn().Msg("Nameserver is not found, fixing...")
			return true, createk3sResolvConf()
		}
		return false, err
	}
	defer f.Close()

	scan := bufio.NewScanner(f)

	for scan.Scan() {
		if resolvRegexp.Match(scan.Bytes()) {
			logger.Debug().Msg("Nameserver found, no fix needed")
			return false, nil
		}
	}

	logger.Warn().Msg("Nameserver is not found, fixing...")
	return true, createk3sResolvConf()
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
