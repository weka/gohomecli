package client

import (
	"encoding/json"
	"fmt"
	"time"
)

// Cluster API structure
type Event struct {
	ID             string          `json:"id"`
	CloudID        string          `json:"cloud_id"`
	ClusterID      string          `json:"cluster_id"`
	EventType      string          `json:"type"`
	Category       string          `json:"category"`
	Entity         string          `json:"entity"`
	Fields         json.RawMessage `json:"eventFields"` // map[string]interface{}
	NodeID         string          `json:"nid"`
	Permission     string          `json:"permission"`
	Severity       string          `json:"severity"`
	Time           time.Time       `json:"timestamp"`
	IngestTime     time.Time       `json:"ingest_time"`
	OrganizationID int64           `json:"org_id"`
	Processed      bool            `json:"processed"`
}

// GetCluster returns a single event
func (client *Client) GetEvent(clusterID string, eventID string) (*Event, error) {
	logger.Info().Str("clusterID", clusterID).Str("eventID", eventID).Msg("Fetching event")
	event := &Event{}
	err := client.Get(fmt.Sprintf("events/%s", eventID), event, &RequestOptions{Prefix: "api"})
	if err != nil {
		return nil, fmt.Errorf("could not fetch event %s: %s", eventID, err)
	}
	return event, nil
}

func (client *Client) QueryEvents(clusterID string) (*PagedQuery, error) {
	query, err := client.QueryEntities(
		fmt.Sprintf("%s/events/list", clusterID),
		&RequestOptions{Prefix: "api", NoMetadata: true})
	if err != nil {
		return nil, err
	}
	return query, nil
}

func (query *PagedQuery) NextEvent() (*Event, error) {
	event := &Event{}
	ok, err := query.NextEntity(event)
	if err != nil {
		return nil, fmt.Errorf("failed to get next event: %s", err)
	}
	if !ok {
		return nil, nil
	}
	return event, nil
}
