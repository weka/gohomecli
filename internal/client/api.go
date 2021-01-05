package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/weka/gohomecli/internal/env"
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

type genericResponseEnvelope struct {
	Data interface{} `json:"data"`
	Meta metaData    `json:"meta"`
}

type genericEntityEnvelope struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Attributes    interface{}     `json:"attributes"`
	Relationships json.RawMessage `json:"relationships"`
}

type genericQueryResultsEnvelope struct {
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
	return NewClient(env.CurrentSiteConfig.CloudURL, env.CurrentSiteConfig.APIKey)
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
		result = &genericResponseEnvelope{
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
	entity := genericEntityEnvelope{
		Attributes: result,
	}
	return client.Get(fmt.Sprintf("%s/%s", resource, id), &entity)
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

type PagedQuery struct {
	Client          *Client
	URL             string
	QueryParams     map[string]interface{}
	Page            int
	PageResults     genericQueryResultsEnvelope
	HasMorePages    bool
	index           int
	maxIndex        int
	queryMetaParams map[string]interface{}
	queryParamsStr  string
}

func (client *Client) QueryEntities(url string, queryParams map[string]interface{}) (*PagedQuery, error) {
	query := PagedQuery{
		Client:          client,
		URL:             url,
		QueryParams:     queryParams,
		queryParamsStr:  queryParamsToString(queryParams),
		Page:            0,
		queryMetaParams: make(map[string]interface{}),
	}
	err := query.fetchNextPage()
	if err != nil {
		return nil, err
	}
	return &query, nil
}

func (query *PagedQuery) fetchNextPage() error {
	query.Page++
	query.queryMetaParams["page"] = query.Page
	query.PageResults = genericQueryResultsEnvelope{}
	allQueryParamsStr := queryParamsToString(query.queryMetaParams)
	if query.queryParamsStr != "" {
		allQueryParamsStr = query.queryParamsStr + "&" + allQueryParamsStr
	}
	urlWithParams := fmt.Sprintf("%s?%s", query.URL, allQueryParamsStr)
	err := query.Client.GetRaw(urlWithParams, &query.PageResults)
	if err != nil {
		return err
	}
	query.HasMorePages = len(query.PageResults.Data) == query.PageResults.Meta.PageSize
	query.index = -1
	query.maxIndex = len(query.PageResults.Data) - 1
	return nil
}

func (query *PagedQuery) NextEntity(result interface{}) (ok bool, err error) {
	if query.index == query.maxIndex {
		if !query.HasMorePages {
			return false, nil
		}
		err := query.fetchNextPage()
		if err != nil {
			return false, err
		}
	}
	query.index++
	err = json.Unmarshal(query.PageResults.Data[query.index].Attributes, result)
	if err != nil {
		return false, err
	}
	return true, nil
}
