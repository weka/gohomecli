package client

import (
	"fmt"
	"time"
)

// Cluster API structure
type Cluster struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	CreatedAt        time.Time `json:"created_at"`
	CustomerID       string    `json:"customer_id"`
	EventStore       int       `json:"event_store"`
	LastEvent        time.Time `json:"last_event"`
	LastSeen         time.Time `json:"last_seen"`
	LicenseDeletedAt time.Time `json:"license_deleted_at"`
	LicenseSyncTime  time.Time `json:"license_sync_time"`
	Muted            bool      `json:"muted"`
	MuteTime         time.Time `json:"mute_time"`
	PublicKey        string    `json:"public_key"`
	SkipLicenseCheck bool      `json:"skip_license_check"`
	SoftwareRelease  string    `json:"software_release"`
	UpdatedAt        time.Time `json:"updated_at"`
	Version          string    `json:"version"`
}

// GetCluster returns a single cluster
func (client *Client) GetCluster(id string) (*Cluster, error) {
	logger.Info().Str("id", id).Msg("Fetching cluster")
	cluster := &Cluster{}
	err := client.GetAPIEntity("clusters", id, cluster)
	if err != nil {
		return nil, fmt.Errorf("could not fetch cluster %s: %s", id, err)
	}
	return cluster, nil
}

func (client *Client) GetClusterCustomer(cluster *Cluster) (*Customer, error) {
	if len(cluster.CustomerID) == 0 {
		return nil, fmt.Errorf("Cluster %s has no customer", cluster.ID)
	}
	customer, err := client.GetCustomer(cluster.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch customer for cluster %s: %s", cluster.ID, err)
	}
	return customer, nil
}

func (client *Client) QueryClusters() (func(interface{}) (bool, error), error) {
	params := map[string]interface{}{"seen_within_seconds": 60 * 60}
	return client.QueryEntities("clusters", params)
}
