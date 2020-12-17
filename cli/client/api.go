package client

import (
	"context"
	"encoding/json"
	"errors"
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
	config := cli.ReadCLIConfig()
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

type entityResponse struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Attributes    interface{}
	Relationships json.RawMessage `json:"relationships"`
}

func (client *Client) sendRequest(req *http.Request, v interface{}) error {
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", client.apiKey))

	logger.Debug().
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Send()

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
			Send()
		var errRes errorResponse
		if err = json.NewDecoder(res.Body).Decode(&errRes); err == nil {
			logger.Error().Err(err).Msg("Unable to parse JSON")
			return errors.New(errRes.Message)
		}
		return fmt.Errorf("%s %s returned HTTP %d", req.Method, req.URL, res.StatusCode)
	}
	logger.Debug().
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Int("status", res.StatusCode).
		Send()

	fullResponse := successResponse{
		Data: v,
	}
	if err = json.NewDecoder(res.Body).Decode(&fullResponse); err != nil {
		logger.Error().Err(err).Msg("Unable to parse JSON")
		return err
	}

	return nil
}

// getAPIEntity is a general implementation for getting a single object from an
// API resource
func (client *Client) getAPIEntity(ctx context.Context, resource string, id string,
	result interface{}) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/%s", client.BaseURL, resource, id), nil)
	if err != nil {
		return err
	}

	entity := entityResponse{
		Attributes: result,
	}
	req = req.WithContext(ctx)

	if err := client.sendRequest(req, &entity); err != nil {
		return err
	}

	return nil
}
