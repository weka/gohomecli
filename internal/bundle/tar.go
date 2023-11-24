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

	f, err := os.Open(string(t))
	if err != nil {
		return fmt.Errorf("os open: %w", err)
	}
	defer f.Close()

	reader = f
	if t.isGZipped() {
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
				break
			}
			fmt.Println("tar read: ", err.Error())
			continue
		}

		var match bool
		for _, fileName := range fileNames {
			match, err = path.Match(fileName, header.Name)
			if err != nil {
				return err
			}
			if match {
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
		return ErrWrongBundle
	}

	return nil
}

func (t Tar) isGZipped() bool {
	return strings.HasSuffix(string(t), ".gz") || strings.HasSuffix(string(t), ".tgz")
}
