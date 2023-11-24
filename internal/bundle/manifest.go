package bundle

import (
	"encoding/json"
	"fmt"
)

const (
	versionFileName = "versions.json"
)

type Manifest struct {
	WekaHome     string            `json:"wekaHome"`
	K3S          string            `json:"k3S"`
	DockerImages map[string]string `json:"dockerImages"`
	FilesDigest  map[string]string `json:"filesDigest"`
}

func GetManifest() (Manifest, error) {
	var manifest Manifest

	versionsBytes, err := ReadFileBytes(versionFileName)
	if err != nil {
		return Manifest{}, fmt.Errorf("failed to read bundle versions: %w", err)
	}

	err = json.Unmarshal(versionsBytes, &manifest)
	if err != nil {
		return Manifest{}, fmt.Errorf("failed to unmarshal bundle versions: %w", err)
	}

	return manifest, nil
}
