package client

import (
	"fmt"
)

type ServerStatus struct {
	Active  bool   `json:"active"`
	Version string `json:"version"`
}

// GetCluster returns a single cluster
func (client *Client) GetServerStatus() (*ServerStatus, error) {
	logger.Info().Msg("Fetching server status")
	status := &ServerStatus{}
	err := client.GetRaw("status", status)
	if err != nil {
		return nil, fmt.Errorf("could not fetch server status: %s", err)
	}
	return status, nil
}

func (client *Client) GetDBStatus() ([]byte, error) {
	result := &genericRawResponse{}
	err := client.GetRaw("db/status", result)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}
