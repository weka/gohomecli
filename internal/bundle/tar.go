package bundle

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

var ErrWrongBundle = errors.New("tar: invalid bundle provided")

type Tar string

// GetFiles reads files from tar archive and pass them to callback
// fileNames can be direct match or glob like file_*.go, callback is applied to any file matched
func (t Tar) GetFiles(cb func(fs.FileInfo, io.Reader) error, fileNames ...string) error {
	var reader io.ReadCloser

	logger.Debug().Msgf("opening %q", string(t))

	f, err := os.Open(string(t))
	if err != nil {
		return fmt.Errorf("os open: %w", err)
	}
	defer f.Close()

	reader = f
	if t.isGZipped() {
		logger.Debug().Msgf("%q is gzipped, using gunzip", string(t))

		gz, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("gzip: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	tarReader := tar.NewReader(reader)

	var found bool

	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Debug().Msgf("tar read: EOF")
				break
			}
			logger.Err(err).Msg("tar read file header error, skipping")
			continue
		}

		var match bool
		for _, fileName := range fileNames {
			match, err = path.Match(fileName, header.FileInfo().Name())
			if err != nil {
				return err
			}
			if match {
				logger.Debug().Msgf("Found %q in archive matches %q", header.FileInfo().Name(), fileName)
				break
			}
		}

		if !match {
			continue
		}

		found = true

		if err := cb(header.FileInfo(), tarReader); err != nil {
			return err
		}
	}

	if !found {
		return errors.Join(ErrWrongBundlePath, fmt.Errorf("missing files: %s", fileNames))
	}

	return nil
}

func (t Tar) isGZipped() bool {
	return strings.HasSuffix(string(t), ".gz") || strings.HasSuffix(string(t), ".tgz")
}
