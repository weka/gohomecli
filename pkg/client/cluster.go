package client

import (
	"fmt"
	"time"
)

// Cluster API structure
type Cluster struct {
	LicenseDeletedAt time.Time `json:"license_deleted_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedAt        time.Time `json:"created_at"`
	MuteTime         time.Time `json:"mute_time"`
	LicenseSyncTime  time.Time `json:"license_sync_time"`
	LastEvent        time.Time `json:"last_event"`
	LastSeen         time.Time `json:"last_seen"`
	CustomerID       string    `json:"customer_id"`
	ID               string    `json:"id"`
	Version          string    `json:"version"`
	SoftwareRelease  string    `json:"software_release"`
	PublicKey        string    `json:"public_key"`
	Name             string    `json:"name"`
	EventStore       int       `json:"event_store"`
	SkipLicenseCheck bool      `json:"skip_license_check"`
	Muted            bool      `json:"muted"`
}

// GetCluster returns a single cluster
func (client *Client) GetCluster(id string) (*Cluster, error) {
	logger.Info().Str("id", id).Msg("Fetching cluster")
	cluster := &Cluster{}
	err := client.GetAPIEntity("clusters", id, cluster)
	if err != nil {
		return nil, fmt.Errorf("could not fetch cluster %s: %w", id, err)
	}

	return cluster, nil
}

func (client *Client) GetClusterCustomer(cluster *Cluster) (*Customer, error) {
	if len(cluster.CustomerID) == 0 {
		return nil, fmt.Errorf("Cluster %s has no customer", cluster.ID)
	}
	customer, err := client.GetCustomer(cluster.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch customer for cluster %s: %w", cluster.ID, err)
	}

	return customer, nil
}

func (client *Client) QueryClusters(options *RequestOptions) (*PagedQuery, error) {
	query, err := client.QueryEntities("clusters", options)
	if err != nil {
		return nil, err
	}

	return query, nil
}

func GetActiveClustersParams() *QueryParams {
	return (&QueryParams{}).
		Set("seen_within_seconds", 24*60*60).
		Set("muted", "false").
		Set("monitored", "true")
}

func (query *PagedQuery) NextCluster() (*Cluster, error) {
	cluster := &Cluster{}
	ok, err := query.NextEntity(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get next cluster: %w", err)
	}
	if !ok {
		return nil, nil
	}

	return cluster, nil
}
