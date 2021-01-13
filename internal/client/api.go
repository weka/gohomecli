package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/weka/gohomecli/internal/env"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/weka/gohomecli/internal/utils"
)

var logger = utils.GetLogger("API")

type metaData struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type rawResponse struct {
	Data json.RawMessage `json:"data"`
	Meta metaData        `json:"meta"`
}

type entityEnvelope struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Attributes    interface{}     `json:"attributes"`
	Relationships json.RawMessage `json:"relationships"`
}

type responseEnvelope struct {
	Data entityEnvelope `json:"data"`
	Meta metaData       `json:"meta"`
}

type queryResultsEnvelope struct {
	Data []struct {
		ID            string          `json:"id"`
		Type          string          `json:"type"`
		Attributes    json.RawMessage `json:"attributes"`
		Relationships json.RawMessage `json:"relationships"`
	}
	Meta metaData `json:"meta"`
}

// Client is an API client for a given service URL
type Client struct {
	BaseURL       string
	DefaultPrefix string
	apiKey        string
	HTTPClient    *http.Client
}

// NewClient creates and returns a new Client instance
func NewClient(url string, apiKey string) *Client {
	url = strings.TrimRight(url, "/")
	return &Client{
		BaseURL:       url,
		DefaultPrefix: "api/v3",
		apiKey:        apiKey,
		HTTPClient: &http.Client{
			Timeout: time.Minute,
		},
	}
}

// GetClient returns a new Client instance, instantiated
// with values from the CLI configuration file
func GetClient() *Client {
	return NewClient(env.CurrentSiteConfig.CloudURL, env.CurrentSiteConfig.APIKey)
}

func (client *Client) getFullURL(url string, options *RequestOptions) string {
	if options.Prefix == "" {
		options.Prefix = client.DefaultPrefix
	}
	fullURL := fmt.Sprintf("%s/%s/%s", client.BaseURL, options.Prefix, url)
	if options.Params != nil {
		fullURL = fmt.Sprintf("%s?%s", fullURL, queryParamsToString(options.Params))
	}
	return fullURL
}

type QueryParams map[string]interface{}

type RequestOptions struct {
	Prefix              string
	Params              QueryParams
	Body                io.Reader
	NoMetadata          bool
	NoAutoFetchNextPage bool
}

func (client *Client) SendRequest(method string, url string, result interface{}, options *RequestOptions) error {
	if options == nil {
		options = &RequestOptions{}
	}
	fullURL := client.getFullURL(url, options)
	req, err := http.NewRequest(method, fullURL, options.Body)
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

	if err = json.NewDecoder(res.Body).Decode(result); err != nil {
		logger.Error().Err(err).Msg("Unable to parse JSON")
		return err
	}

	return nil
}

// Get sends a GET request, and does not expect the response to be enveloped
func (client *Client) Get(url string, result interface{}, options *RequestOptions) error {
	return client.SendRequest("GET", url, result, options)
}

// GetAPIEntity is a general implementation for getting a single object from an
// API resource
func (client *Client) GetAPIEntity(resource string, id string, result interface{}) error {
	entity := responseEnvelope{
		Data: entityEnvelope{
			Attributes: result,
		},
	}
	return client.Get(fmt.Sprintf("%s/%s", resource, id), &entity, nil)
}

func queryParamsToString(queryParamGroups ...map[string]interface{}) string {
	var parts []string
	for _, queryParams := range queryParamGroups {
		for param, value := range queryParams {
			parts = append(parts, fmt.Sprintf("%s=%v", param, value))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "&")
}
