package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client is an API client for a given service URL
type Client struct {
	BaseURL    string
	apiKey     string
	HTTPClient *http.Client
}

// NewClient creates and returns a new Client instance
func NewClient(url string, apiKey string) *Client {
	url = strings.TrimRight(url, "/")
	return &Client{
		BaseURL: fmt.Sprintf("%s/api/v3", url),
		apiKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: time.Minute,
		},
	}
}

// GetClient returns a new Client instance, instantiated
// with values from the CLI configuration file
func GetClient() *Client {
	config := ReadCLIConfig()
	return NewClient(config.CloudURL, config.APIKey)
}

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type successResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

// Cluster API data
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
	Relationships interface{} `json:"relationships"`
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

func (c *Client) sendRequest(req *http.Request, v interface{}) error {
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.apiKey))

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		var errRes errorResponse
		if err = json.NewDecoder(res.Body).Decode(&errRes); err == nil {
			return errors.New(errRes.Message)
		}

		return fmt.Errorf("%s %s returned HTTP %d", req.Method, req.URL, res.StatusCode)
	}

	fullResponse := successResponse{
		Data: v,
	}
	if err = json.NewDecoder(res.Body).Decode(&fullResponse); err != nil {
		return err
	}

	return nil
}

// GetCluster returns a single cluster
func (c *Client) GetCluster(ctx context.Context, id string) (*Cluster, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/clusters/%s", c.BaseURL, id), nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	res := Cluster{}
	if err := c.sendRequest(req, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
