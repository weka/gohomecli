package k3s

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/weka/gohomecli/internal/install/bundle"

	"github.com/rs/zerolog"

	"github.com/weka/gohomecli/internal/utils"
)

func getCurrentPlatform() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

func unzippedData(imagePath string) ([]byte, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return nil, err
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	mime := http.DetectContentType(buffer)
	if mime == "application/x-gzip" {
		reader, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}

		return io.ReadAll(reader)
	}

	return io.ReadAll(file)
}

func plainLogWriter(level zerolog.Level) io.WriteCloser {
	return utils.NewWriteScanner(func(b []byte) {
		logger.WithLevel(level).Msg(string(b))
	})
}

func ImportImages(ctx context.Context, imagePaths []string, failFast bool) error {
	var importErrors []error
	for _, imagePath := range imagePaths {
		logger := logger.With().Str("imagePath", imagePath).Logger()
		if ctx.Err() != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				logger.Info().Msg("Context canceled")
				return ctx.Err()
			} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				logger.Info().Msg("Context deadline exceeded")
				return ctx.Err()
			} else {
				logger.Warn().Msg("Context error")
				return ctx.Err()
			}
		}

		data, err := unzippedData(imagePath)
		if err != nil {
			logger.Warn().
				Err(err).
				Msg("Failed to read image data")

			if failFast {
				return err
			} else {
				importErrors = append(importErrors, err)
				continue
			}
		}

		reader := bytes.NewBuffer(data)
		cmd := exec.Command(
			"k3s", "ctr", "image", "import",
			"--platform", getCurrentPlatform(),
			"--", "-",
		)
		cmd.Stdin = reader

		stderr, err := cmd.StderrPipe()
		if err != nil {
			logger.Warn().
				Err(err).
				Msg("Failed to capture import command output")

			if failFast {
				return err
			} else {
				importErrors = append(importErrors, err)
				continue
			}
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			logger.Warn().
				Err(err).
				Msg("Failed to capture import command output")

			if failFast {
				return err
			} else {
				importErrors = append(importErrors, err)
				continue
			}
		}

		logger.Debug().Strs("command", cmd.Args).Msg("Running import command")
		err = cmd.Start()
		if err != nil {
			logger.Warn().
				Err(err).
				Msg("Failed run import command")

			if failFast {
				return err
			} else {
				importErrors = append(importErrors, err)
				continue
			}
		}

		go io.Copy(plainLogWriter(zerolog.InfoLevel), stdout)
		go io.Copy(plainLogWriter(zerolog.ErrorLevel), stderr)

		err = cmd.Wait()
		if err != nil {
			logger.Warn().
				Err(err).
				Msg("Failed to import image")

			if failFast {
				return err
			} else {
				importErrors = append(importErrors, err)
				continue
			}
		}
	}

	if len(importErrors) > 0 {
		return errors.Join(importErrors...)
	}

	return nil
}

func ImportBundleImages(ctx context.Context, bundlePathOverride string, failFast bool) error {
	if bundlePathOverride != "" {
		if err := bundle.SetBundlePath(bundlePathOverride); err != nil {
			return err
		}
	}

	imagePaths := []string{}
	err := bundle.Walk("images", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			imagePaths = append(imagePaths, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return ImportImages(ctx, imagePaths, failFast)
}
