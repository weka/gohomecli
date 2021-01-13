package client

import (
	"encoding/json"
	"fmt"
	"github.com/weka/gohomecli/internal/utils"
	"time"
)

// Cluster API structure
type Event struct {
	ID             string          `json:"id"`
	CloudID        string          `json:"cloud_id"`
	ClusterID      string          `json:"cluster_id"`
	EventType      string          `json:"type"`
	Category       string          `json:"category"`
	IsBackend      bool            `json:"is_backend"`
	Entity         string          `json:"entity"`
	Params         json.RawMessage `json:"params"` // map[string]interface{}
	NodeID         string          `json:"nid"`
	Permission     string          `json:"permission"`
	Severity       string          `json:"severity"`
	Time           time.Time       `json:"timestamp"`
	IngestTime     time.Time       `json:"cloud_digested_ts"`
	OrganizationID int64           `json:"org_id"`
	Processed      bool            `json:"processed"`
}

func (event *Event) ComputeProcessingTime() float64 {
	return event.IngestTime.Sub(event.Time).Seconds()
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

type EventQueryOptions struct {
	WithInternalEvents bool
	SortByIngestTime   bool
	IncludeTypes       []string
	ExcludeTypes       []string
	NodeIDs            []int
	MinSeverity        string
	StartTime          time.Time
	EndTime            time.Time
	//Params             string
}

func (options *EventQueryOptions) ToQueryParams() (QueryParams, error) {
	params := make(QueryParams)
	if options.WithInternalEvents {
		params["intr"] = "t"
	}
	if options.SortByIngestTime {
		params["dt"] = "t"
	}
	if len(options.IncludeTypes) != 0 {
		utils.UserError("--type is not supported yet")
	}
	if len(options.ExcludeTypes) != 0 {
		utils.UserError("--exclude-type is not supported yet")
	}
	if len(options.NodeIDs) != 0 {
		utils.UserError("--node-ids is not supported yet")
	}
	if options.MinSeverity != "" {
		params["svr"] = options.MinSeverity
	}
	if !options.StartTime.IsZero() {
		params["frm"] = options.StartTime.Format(time.RFC3339)
	}
	if !options.EndTime.IsZero() {
		params["to"] = options.EndTime.Format(time.RFC3339)
	}
	//if options.Params != "" {
	//	params["params"] = options.Params
	//}
	return params, nil
}

func (client *Client) QueryEvents(clusterID string, options *EventQueryOptions) (*PagedQuery, error) {
	var params QueryParams
	if options != nil {
		var err error
		params, err = options.ToQueryParams()
		if err != nil {
			return nil, err
		}
	}
	query, err := client.QueryEntities(
		fmt.Sprintf("%s/events/list", clusterID),
		&RequestOptions{Prefix: "api", NoMetadata: true, Params: params})
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
