package bundle

import (
	"encoding/json"
	"fmt"
)

const (
	versionFileName = "versions.json"
)

type Versions struct {
	WekaHome     string            `json:"wekaHome"`
	K3S          string            `json:"k3S"`
	DockerImages map[string]string `json:"dockerImages"`
}

func GetVersions() (Versions, error) {
	var versions Versions

	versionsBytes, err := ReadFileBytes(versionFileName)
	if err != nil {
		return Versions{}, fmt.Errorf("failed to read bundle versions: %w", err)
	}

	err = json.Unmarshal(versionsBytes, &versions)
	if err != nil {
		return Versions{}, fmt.Errorf("failed to unmarshal bundle versions: %w", err)
	}

	return versions, nil
}
