package bundle

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	markerFileName = ".bundle"
)

var ErrWrongBundlePath = errors.New("wrong bundle directory")

func executableDirectory() string {
	// Get the absolute path of the executable
	executablePath, err := os.Executable()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed determining executable path")
	}

	// Resolve the absolute path (eliminate any symbolic links)
	absolutePath, err := filepath.EvalSymlinks(executablePath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed determining executable path")
	}

	return filepath.Dir(absolutePath)
}

var bundlePath string

// by default homecli is located in /opt/wekahome/{release}/bin
// and bundle in /opt/wekahome/{release}/
func BundlePath() string {
	if bundlePath == "" {
		bundlePath = filepath.Join(executableDirectory(), "..")
	}

	realPath, err := filepath.EvalSymlinks(bundlePath)
	if err != nil {
		fmt.Printf("Bad bundle path %q", realPath)
		os.Exit(1)
	}

	return realPath
}

func BundleBinDir() string {
	return filepath.Join(BundlePath(), "bin")
}

// SetBundlePath allows to override default bundle directory
func SetBundlePath(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("%w: %q", ErrWrongBundle, path)
	}

	bundlePath = path
	if !IsBundled() {
		return fmt.Errorf("%w: %q not exists", ErrWrongBundle, markerFileName)
	}

	return nil
}

func IsBundled() bool {
	markerPath := GetPath(markerFileName)
	_, err := os.Stat(markerPath)
	return err == nil
}

func GetPath(path string) string {
	return filepath.Join(BundlePath(), path)
}

func ReadFileBytes(path string) ([]byte, error) {
	return os.ReadFile(GetPath(path))
}

func ReadFile(path string) (io.Reader, error) {
	return os.Open(GetPath(path))
}

type BundleImage struct {
	Name     string
	Location string
}

// Images returns list of images in bundle
func Images() ([]BundleImage, error) {
	manifest, err := GetManifest()
	if err != nil {
		return nil, err
	}

	var images []BundleImage

	for path, img := range manifest.DockerImages {
		images = append(images, BundleImage{
			Name:     img,
			Location: filepath.Join(BundlePath(), path),
		})
	}

	return images, nil
}
