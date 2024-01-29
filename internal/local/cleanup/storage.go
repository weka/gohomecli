package cleanup

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/weka/gohomecli/internal/local/k3s"
	"github.com/weka/gohomecli/internal/utils"
)

var logger = utils.GetLogger("cleanup")

func LocalStorage(ctx context.Context) error {
	used, err := getVolumesInUse(ctx)
	if err != nil {
		return err
	}
	logger.Debug().Interface("used", used).Msg("Volumes in use")

	localPath, err := filepath.EvalSymlinks(k3s.DefaultLocalStoragePath)
	if err != nil {
		return err
	}

	available, err := filepath.Glob(filepath.Join(localPath, "*"))
	if err != nil {
		return err
	}

	logger.Debug().Interface("available", available).Msg("Available volumes")

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

	logger.Warn().Msgf("Next %d PV will be removed:", len(toRemove))
	for i := range toRemove {
		logger.Info().Msgf("%d. %s", i+1, filepath.Base(toRemove[i]))
	}
	logger.Info().Msg("Enter Yes (case sensitive) to confirm:\n")

	if err = getConfirmation(ctx); err != nil {
		return err
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

func getVolumesInUse(ctx context.Context) (map[string]bool, error) {
	logger.Debug().Str("path", k3s.DefaultLocalStoragePath).Msg("Getting used volumes")

	var isUsed = map[string]bool{}

	kubeCmd, err := utils.ExecCommand(ctx, "kubectl",
		[]string{"get", "pv", "-o", "jsonpath={.items[*].spec.hostPath.path}", "-A"},
		utils.WithStderrLogger(logger, zerolog.WarnLevel),
		utils.WithStdoutReader(func(lines chan []byte) {
			for line := range lines {
				for _, dir := range strings.Split(string(line), " ") {
					logger.Debug().Str("output", dir).Msg("PV")
					isUsed[dir] = true
				}
			}
		}),
	)

	if err = errors.Join(err, kubeCmd.Wait()); err != nil {
		return nil, err
	}

	return isUsed, nil
}

func getConfirmation(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	lineCh := make(chan []byte)

	go func() {
		line, _, err := reader.ReadLine()
		if err != nil {
			logger.Debug().Err(err).Msg("Failed to read input")
		}
		lineCh <- line
		close(lineCh)
	}()

	select {
	case line := <-lineCh:
		if string(line) != "Yes" {
			return errors.New("cleanup was canceled")
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
