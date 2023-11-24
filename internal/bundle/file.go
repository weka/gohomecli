package bundle

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	markerFileName = ".bundle"
)

var (
	execDir string
)

func executableDirectory() string {
	if execDir != "" {
		return execDir
	}

	// Get the absolute path of the executable
	executablePath, err := os.Executable()
	if err != nil {
		panic(fmt.Sprintf("Failed determining executable path: %v\n", err))
	}

	// Resolve the absolute path (eliminate any symbolic links)
	absolutePath, err := filepath.EvalSymlinks(executablePath)
	if err != nil {
		panic(fmt.Sprintf("Failed determining executable path: %v\n", err))
	}

	execDir = filepath.Dir(absolutePath)
	return execDir
}

func IsInsideBundle() bool {
	markerPath := filepath.Join(executableDirectory(), markerFileName)
	_, err := os.Stat(markerPath)
	return err == nil
}

func GetPath(path string) string {
	return filepath.Join(executableDirectory(), path)
}

func ReadFileBytes(path string) ([]byte, error) {
	return os.ReadFile(GetPath(path))
}

func ReadFile(path string) (io.Reader, error) {
	return os.Open(GetPath(path))
}

func Walk(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(GetPath(root), walkFn)
}
