package cleanup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/weka/gohomecli/internal/utils"
)

var logger = utils.GetLogger("cleanup")

var localStoragePath = "/opt/local-path-provisioner"

func SetLocalStoragePath(path string) {
	localStoragePath = path
}

func LocalStorage(ctx context.Context) error {
	used, err := getUsed(ctx)
	if err != nil {
		return err
	}

	localPath, err := filepath.EvalSymlinks(localStoragePath)
	if err != nil {
		return err
	}

	available, err := filepath.Glob(filepath.Join(localPath, "*"))
	if err != nil {
		return err
	}

	var toRemove []string
	for _, dir := range available {
		if !used[dir] {
			toRemove = append(toRemove, dir)
		}
	}

	if len(toRemove) == 0 {
		logger.Info().Msg("Nothing to cleanup")
		return nil
	}

	for _, dir := range toRemove {
		logger.Warn().Msg(dir)
	}

	logger.Warn().Msg("Next directories will be removed in 10 seconds. Press CTRL+C to cancel")

	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		return fmt.Errorf("cancelled")
	case <-timeout.Done():
		break
	}

	for _, dir := range toRemove {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
		logger.Info().Msgf("Persistent volume %q was deleted", filepath.Base(dir))
	}

	logger.Info().Msg("Local storage was cleaned up")

	return nil
}

func getUsed(ctx context.Context) (map[string]bool, error) {
	logger.Debug().Str("path", localStoragePath).Msg("Getting used volumes")

	kubeCmd := exec.CommandContext(ctx, "kubectl", "get", "pv", "-o", "jsonpath='{.items[*].spec.hostPath.path}'", "-A")
	stdout, err := kubeCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := kubeCmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	errWriter := utils.NewWriteScanner(func(b []byte) {
		logger.Warn().Msg(string(b))
	})

	go io.Copy(errWriter, stderr)

	if err := kubeCmd.Start(); err != nil {
		return nil, err
	}

	out, err := io.ReadAll(stdout)
	if err != nil {
		return nil, err
	}

	if err = kubeCmd.Wait(); err != nil {
		return nil, err
	}

	var isUsed = map[string]bool{}

	for _, dir := range strings.Split(string(out), " ") {
		isUsed[dir] = true
	}
	return isUsed, nil
}
