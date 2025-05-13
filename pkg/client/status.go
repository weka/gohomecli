package client

import (
	"fmt"
)

// ServerStatus is the status of the server
type ServerStatus struct {
	Version string `json:"version"`
	Active  bool   `json:"active"`
}

// GetServerStatus returns the status of the server
func (client *Client) GetServerStatus() (*ServerStatus, error) {
	logger.Info().Msg("Fetching server status")
	status := &ServerStatus{}
	err := client.Get("status", status, nil)
	if err != nil {
		return nil, fmt.Errorf("could not fetch server status: %w", err)
	}

	return status, nil
}

// GetDBStatus returns the status of the database
func (client *Client) GetDBStatus() ([]byte, error) {
	result := &rawResponse{}
	err := client.Get("db/status", result, nil)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}
