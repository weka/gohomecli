package chart

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReadKubeConfig reads the kubeconfig from the given path with fallback to ~/.kube/config
func ReadKubeConfig(kubeConfigPath string) ([]byte, error) {
	if kubeConfigPath == "" {
		kubeConfigPath = os.Getenv("KUBECONFIG")
		if kubeConfigPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("unable to read kubeconfig: %w", err)
			}

			kubeConfigPath = filepath.Join(homeDir, ".kube", "config")
		}
	}

	logger.Debug().Str("kubeConfigPath", kubeConfigPath).Msg("Reading kubeconfig")
	kubeConfig, err := os.ReadFile(kubeConfigPath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read kubeconfig")
		return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	return kubeConfig, nil
}
