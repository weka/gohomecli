package client

import (
	"context"
	"encoding/json"
)

// Cluster API structure
type Cluster struct {
	Type       string `json:"type"`
	Attributes struct {
		ID               string      `json:"id"`
		Name             string      `json:"name"`
		CreatedAt        string      `json:"created_at"`
		CustomerID       string      `json:"customer_id"`
		EventStore       int         `json:"event_store"`
		LastEvent        string      `json:"last_event"`
		LastSeen         string      `json:"last_seen"`
		LicenseDeletedAt int         `json:"license_deleted_at"`
		LicenseSyncTime  string      `json:"license_sync_time"`
		Muted            bool        `json:"muted"`
		MuteTime         int         `json:"mute_time"`
		PublicKey        string      `json:"public_key"`
		SkipLicenseCheck bool        `json:"skip_license_check"`
		SoftwareRelease  string      `json:"software_release"`
		StatsStore       interface{} `json:"stats_store"`
		UpdatedAt        string      `json:"updated_at"`
		Version          string      `json:"version"`
	} `json:"attributes"`
	Relationships json.RawMessage `json:"relationships"`
	// Relationships struct {
	// 	Customer struct {
	// 		Data struct {
	// 			Type string `json:"type"`
	// 			ID   string `json:"id"`
	// 		} `json:"data"`
	// 	} `json:"customer"`
	// } `json:"relationships"`
	ID string `json:"id"`
}

// GetCluster returns a single cluster
func (c *Client) GetCluster(ctx context.Context, id string) (*Cluster, error) {
	logger.Info().Str("id", id).Msg("Fetching cluster")
	cluster := &Cluster{}
	err := c.getAPIEntity(ctx, "clusters", id, cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}
