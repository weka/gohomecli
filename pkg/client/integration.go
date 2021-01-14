package client

import (
	"encoding/json"
	"fmt"
)

type Integration struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Configuration struct {
		Type         string          `json:"type"`
		Rule         IntegrationRule `json:"rule"`
		Destinations []string        `json:"destinations"`
	} `json:"configuration"`
}

type IntegrationRule struct {
	RuleType string          `json:"rule_type"`
	Param    json.RawMessage `json:"param"`
}

// GetIntegration returns a single integration
func (client *Client) GetIntegration(id int) (*Integration, error) {
	logger.Info().Int("id", id).Msg("Fetching integration")
	integration := &Integration{}
	err := client.GetAPIEntity("integrations", id, integration)
	if err != nil {
		return nil, fmt.Errorf("could not fetch integration %d: %s", id, err)
	}
	return integration, nil
}

func (client *Client) QueryIntegrations(options *RequestOptions) (*PagedQuery, error) {
	query, err := client.QueryEntities("integrations", options)
	if err != nil {
		return nil, err
	}
	return query, nil
}

func (query *PagedQuery) NextIntegration() (*Integration, error) {
	integration := &Integration{}
	ok, err := query.NextEntity(integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get next integration: %s", err)
	}
	if !ok {
		return nil, nil
	}
	return integration, nil
}

type IntegrationTestRequest struct {
	EventID string `json:"event_id"`
}

func (client *Client) TestIntegration(id int, eventCode string) error {
	logger.Info().Int("id", id).Str("event", eventCode).Msg("testing integration")
	// TODO: This actually doesn't work, but it's exactly the same in the legacy CLI.
	//       Need to check what the server expects and send the correct request.
	options := &RequestOptions{Body: IntegrationTestRequest{EventID: eventCode}}
	err := client.Post(fmt.Sprintf("integrations/%d/test", id), &json.RawMessage{}, options)
	if err != nil {
		return err
	}
	return nil
}
