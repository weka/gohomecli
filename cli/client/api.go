package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/weka/gohomecli/cli"
	"github.com/weka/gohomecli/cli/logging"
)

var logger = logging.GetLogger("API")

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
	return NewClient(cli.CurrentSiteConfig.CloudURL, cli.CurrentSiteConfig.APIKey)
}

type responseEnvelope struct {
	Data interface{} `json:"data"`
	Meta struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	} `json:"meta"`
}

type genericAPIEntity struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Attributes    interface{}
	Relationships json.RawMessage `json:"relationships"`
}

func (client *Client) SendRequest(method string, url string, result interface{}, raw bool) error {
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", client.BaseURL, url), nil)
	if err != nil {
		return err
	}

	req = req.WithContext(context.Background())
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", client.apiKey))

	logger.Debug().
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Msg("Request")

	res, err := client.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		logger.Error().
			Str("method", req.Method).
			Str("url", req.URL.String()).
			Int("status", res.StatusCode).
			Msg("Response")
		return fmt.Errorf("%s %s returned HTTP %d", req.Method, req.URL, res.StatusCode)
	}
	logger.Debug().
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Int("status", res.StatusCode).
		Msg("Response")

	if !raw {
		result = &responseEnvelope{
			Data: result,
		}
	}
	if err = json.NewDecoder(res.Body).Decode(result); err != nil {
		logger.Error().Err(err).Msg("Unable to parse JSON")
		return err
	}

	return nil
}

// Get sends a GET request
func (client *Client) Get(url string, result interface{}) error {
	return client.SendRequest("GET", url, result, false)
}

// GetRaw sends a GET request, and does not expect the response to be enveloped
func (client *Client) GetRaw(url string, result interface{}) error {
	return client.SendRequest("GET", url, result, true)
}

// GetAPIEntity is a general implementation for getting a single object from an
// API resource
func (client *Client) GetAPIEntity(resource string, id string, result interface{}) error {
	entity := genericAPIEntity{
		Attributes: result,
	}
	return client.Get(fmt.Sprintf("%s/%s", resource, id), &entity)
}
