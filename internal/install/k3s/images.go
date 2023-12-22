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
	"runtime"

	"github.com/weka/gohomecli/internal/install/bundle"
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

func ImportImages(ctx context.Context, imagePaths []string, failFast bool) error {
	var importErrors []error
	for _, imagePath := range imagePaths {
		logger := logger.With().Str("imagePath", imagePath).Logger()

		// check if cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
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

		cmd, err := utils.ExecCommand(ctx, "k3s", []string{
			"ctr", "image", "import",
			"--platform", getCurrentPlatform(),
			"--", "-"},
			utils.WithStdin(bytes.NewBuffer(data)),
			utils.WithStdoutLogger(logger, utils.InfoLevel),
			utils.WithStderrLogger(logger, utils.WarnLevel),
		)
		logger.Debug().Strs("command", cmd.Args).Msg("Running import command")

		if err != nil {
			logger.Warn().Err(err).Msg("Failed run import command")

			if failFast {
				return err
			} else {
				importErrors = append(importErrors, err)
				continue
			}
		}

		err = cmd.Wait()
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to import image")

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

func ImportBundleImages(ctx context.Context, failFast bool) error {
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
