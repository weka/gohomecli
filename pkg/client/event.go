package client

import (
	"encoding/json"
	"fmt"
	"time"
)

// Cluster API structure
type Event struct {
	Time           time.Time       `json:"timestamp"`
	IngestTime     time.Time       `json:"cloud_digested_ts"`
	Entity         string          `json:"entity"`
	EventType      string          `json:"type"`
	Category       string          `json:"category"`
	ID             string          `json:"id"`
	NodeID         string          `json:"nid"`
	Permission     string          `json:"permission"`
	Severity       string          `json:"severity"`
	ClusterID      string          `json:"guid"`
	CloudID        string          `json:"cloud_id"`
	Params         json.RawMessage `json:"params"`
	OrganizationID int64           `json:"org_id"`
	IsBackend      bool            `json:"is_backend"`
	Processed      bool            `json:"processed"`
}

func (e Event) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"category":          e.Category,
		"cluster_id":        e.ClusterID,
		"entity":            e.Entity,
		"type":              e.EventType,
		"params":            e.Params,
		"cloud_digested_ts": e.IngestTime,
		"nid":               e.NodeID,
		"permission":        e.Permission,
		"severity":          e.Severity,
		"timestamp":         e.Time,
		"cloud_id":          e.CloudID,
		"org_id":            e.OrganizationID,
		"is_backend":        e.IsBackend,
		"processed":         e.Processed,
	}

	return json.Marshal(m)
}

func (event *Event) ComputeProcessingTime() float64 {
	return event.IngestTime.Sub(event.Time).Seconds()
}

// GetCluster returns a single event
func (client *Client) GetEvent(clusterID, eventID string) (*Event, error) {
	logger.Info().Str("clusterID", clusterID).Str("eventID", eventID).Msg("Fetching event")
	event := &Event{}
	err := client.Get("events/"+eventID, event, &RequestOptions{Prefix: "api"})
	if err != nil {
		return nil, fmt.Errorf("could not fetch event %s: %w", eventID, err)
	}

	return event, nil
}

type EventQueryOptions struct {
	StartTime          time.Time
	EndTime            time.Time
	MinSeverity        string
	IncludeTypes       []string
	ExcludeTypes       []string
	NodeIDs            []int
	Limit              int
	WithInternalEvents bool
	SortByIngestTime   bool
	Wide               bool
}

func (options *EventQueryOptions) ToQueryParams() (*QueryParams, error) {
	params := &QueryParams{}
	if options.WithInternalEvents {
		params.Set("intr", "t")
	}
	if options.SortByIngestTime {
		params.Set("dt", "t")
	}
	if len(options.IncludeTypes) != 0 {
		for _, eventType := range options.IncludeTypes {
			params.Append("et[]", eventType)
		}
	}
	if len(options.ExcludeTypes) != 0 {
		for _, eventType := range options.ExcludeTypes {
			params.Append("ex_et[]", eventType)
		}
	}
	if len(options.NodeIDs) != 0 {
		for _, nodeID := range options.NodeIDs {
			params.Append("node_id", nodeID)
		}
	}
	if options.MinSeverity != "" {
		params.Set("svr", options.MinSeverity)
	}
	if !options.StartTime.IsZero() {
		params.Set("frm", options.StartTime.Format(time.RFC3339))
	}
	if !options.EndTime.IsZero() {
		params.Set("to", options.EndTime.Format(time.RFC3339))
	}

	// if options.Limit!=0 {
	//	params.Set("page_size", limit)
	//}
	// if options.Params != "" {
	//	params.Set("params" ,options.Params)
	//}
	return params, nil
}

func (client *Client) QueryEvents(clusterID string, options *EventQueryOptions) (*PagedQuery, error) {
	var params *QueryParams
	if options != nil {
		var err error
		params, err = options.ToQueryParams()
		if err != nil {
			return nil, err
		}
	}
	query, err := client.QueryEntities(
		clusterID+"/events/list",
		&RequestOptions{Prefix: "api", NoMetadata: false, Params: params, PageSize: options.Limit})
	if err != nil {
		return nil, err
	}

	return query, nil
}

func (query *PagedQuery) NextEvent() (*Event, error) {
	event := &Event{}
	ok, err := query.NextEntity(event)
	if err != nil {
		return nil, fmt.Errorf("failed to get next event: %w", err)
	}
	if !ok {
		return nil, nil
	}

	return event, nil
}
