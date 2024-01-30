package bundle

import (
	"fmt"
	"path/filepath"
)

func Chart() (string, error) {
	matches, err := filepath.Glob(filepath.Join(BundlePath(), "wekahome-*.tgz"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no chart found in %q", BundlePath())
	}

	return matches[0], nil
}
