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

type ClusterQueryResults struct {
	Data []struct {
		Attributes Cluster
	}
	Meta struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	} `json:"meta"`
}

func (client *Client) QueryClusters() (func() (*Cluster, error), error) {
	var (
		results            ClusterQueryResults
		index              int
		maxIndex           int
		page               int  = 0
		morePagesAvailable bool = true
	)
	nextPage := func() error {
		page++
		results = ClusterQueryResults{}
		err := client.GetRaw(
			fmt.Sprintf("clusters?seen_within_seconds=%d&page=%d", 60*60*24, page),
			&results)
		if err != nil {
			return err
		}
		morePagesAvailable = len(results.Data) == results.Meta.PageSize
		index = -1
		maxIndex = len(results.Data) - 1
		//logger.Debug().
		//	Int("page", page).
		//	Bool("morePagesAvailable", morePagesAvailable).
		//	Int("maxIndex", maxIndex).
		//	Int("len", len(results.Data)).
		//	Int("PageSize", results.Meta.PageSize).
		//	Send()
		return nil
	}
	err := nextPage()
	if err != nil {
		return nil, err
	}
	nextEntity := func() (*Cluster, error) {
		if page == 0 || index == maxIndex {
			if !morePagesAvailable {
				return nil, nil
			}
			err = nextPage()
			if err != nil {
				return nil, err
			}
		}
		index++
		return &results.Data[index].Attributes, nil
	}
	return nextEntity, nil
}
