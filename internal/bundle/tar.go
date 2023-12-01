package bundle

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ErrWrongBundle = errors.New("tar: invalid bundle provided")

type Tar string

type TarCallback struct {
	FileName string
	Callback func(context.Context, fs.FileInfo, io.Reader) error
}

// GetFiles reads files from tar archive and pass them to callback
// fileNames can be direct match or glob like file_*.go, callback is applied to any file matched
func (t Tar) GetFiles(ctx context.Context, calls ...TarCallback) (err error) {
	logger.Debug().Msgf("opening %q", string(t))

	var f *os.File
	f, err = os.Open(string(t))
	if err != nil {
		return fmt.Errorf("os open: %w", err)
	}
	defer f.Close()

	var reader io.ReadCloser = f

	if t.isGZipped() {
		logger.Debug().Msgf("%q is gzipped, using gunzip", string(t))

		gz, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("gzip: %w", err)
		}
		reader = gz
	}

	for _, call := range calls {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// reset file to start so we can reuse it
		f.Seek(0, 0)
		if t.isGZipped() {
			reader.(*gzip.Reader).Reset(f)
		}

		tarReader := tarReaderCtx{tar.NewReader(reader), ctx}

		logger.Debug().Msgf("Looking up for %q in bundle", call.FileName)

		var found bool
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			header, err := tarReader.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return err
			}

			match, err := filepath.Match(call.FileName, header.FileInfo().Name())
			if err != nil {
				return err
			}

			if match {
				found = true
				logger.Debug().Msgf("Found %q", header.FileInfo().Name())
				if err := call.Callback(ctx, header.FileInfo(), tarReader); err != nil {
					return err
				}
			}
		}

		if !found {
			return errors.Join(ErrWrongBundlePath, fmt.Errorf("no required files found for %q", call.FileName))
		}
	}

	return nil
}

func (t Tar) isGZipped() bool {
	return strings.HasSuffix(string(t), ".gz") || strings.HasSuffix(string(t), ".tgz")
}

type tarReaderCtx struct {
	*tar.Reader
	ctx context.Context
}

func (t tarReaderCtx) Read(p []byte) (n int, err error) {
	select {
	case <-t.ctx.Done():
		return 0, t.ctx.Err()
	default:
		return t.Reader.Read(p)
	}
}
