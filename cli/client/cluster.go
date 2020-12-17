package client

import (
	"context"
	"fmt"
)

// Cluster API structure
type Cluster struct {
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
}

// GetCluster returns a single cluster
func (client *Client) GetCluster(ctx context.Context, id string) (*Cluster, error) {
	logger.Info().Str("id", id).Msg("Fetching cluster")
	cluster := &Cluster{}
	err := client.getAPIEntity(ctx, "clusters", id, cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (client *Client) GetClusterCustomer(ctx context.Context, cluster *Cluster) (*Customer, error) {
	if len(cluster.CustomerID) == 0 {
		return nil, fmt.Errorf("Cluster %s has no customer", cluster.ID)
	}
	customer, err := client.GetCustomer(ctx, cluster.CustomerID)
	if err != nil {
		return nil, err
	}
	return customer, nil
}
