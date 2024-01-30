package k3s

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"

	"github.com/weka/gohomecli/internal/local/bundle"
)

func ImportImages(ctx context.Context, images map[string]string, failFast bool) error {
	client, err := containerd.New("/run/k3s/containerd/containerd.sock")
	if err != nil {
		return err
	}

	ctx = namespaces.WithNamespace(ctx, "k8s.io")

	platform, err := platforms.Parse(getCurrentPlatform())
	if err != nil {
		return err
	}

	var importErrors []error
	for name, imagePath := range images {
		// check if cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		logger := logger.With().Str("image", name).Logger()

		err := imageReaderFunc(imagePath, func(reader io.Reader) error {
			_, err := client.ImageService().Get(ctx, name)
			if !errors.Is(err, errdefs.ErrNotFound) {
				return err
			}
			if err == nil {
				logger.Debug().Msg("Image already exists")
				return nil
			}

			logger.Info().Msg("Importing image")

			_, err = client.Import(ctx, reader,
				containerd.WithImportPlatform(platforms.Any(platform)),
				containerd.WithIndexName(name),
			)
			return err
		})
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
	var imagePaths = make(map[string]string)

	images, err := bundle.Images()
	if err != nil {
		return err
	}

	for i := range images {
		imagePaths[images[i].Name] = images[i].Location
	}

	return ImportImages(ctx, imagePaths, failFast)
}

func getCurrentPlatform() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

func imageReaderFunc(imagePath string, cb func(io.Reader) error) (err error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return err
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	if mime := http.DetectContentType(buffer); mime == "application/x-gzip" {
		reader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer reader.Close()

		return cb(reader)
	}

	return cb(file)
}
