package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/weka/gohomecli/internal/env"
	"github.com/weka/gohomecli/internal/utils"
)

var logger = utils.GetLogger("API")

const maxConcurrentDownloads = 16 // TODO: verify this number

type metaData struct {
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
	HasNextPage bool   `json:"has_next_page"`
	NextCursor  string `json:"next_cursor"`
}

type rawResponse struct {
	Data json.RawMessage `json:"data"`
	Meta metaData        `json:"meta"`
}

type entityEnvelope struct {
	ID            any             `json:"id"`
	Type          string          `json:"type"`
	Attributes    any             `json:"attributes"`
	Relationships json.RawMessage `json:"relationships"`
}

type responseEnvelope struct {
	Data entityEnvelope `json:"data"`
	Meta metaData       `json:"meta"`
}

type queryResultsEnvelope struct {
	Data []struct {
		ID            any             `json:"id"`
		Type          string          `json:"type"`
		Attributes    json.RawMessage `json:"attributes"`
		Relationships json.RawMessage `json:"relationships"`
	}
	Entries []json.RawMessage `json:"entries"`
	Meta    metaData          `json:"meta"`
}

// Client is an API client for a given service URL
type Client struct {
	HTTPClient    *http.Client
	BaseURL       string
	DefaultPrefix string
	apiKey        string
}

// NewClient creates and returns a new Client instance
func NewClient(url, apiKey string) *Client {
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
	queryParams := options.Params.String()
	if queryParams != "" {
		fullURL = fmt.Sprintf("%s?%s", fullURL, queryParams)
	}

	return fullURL
}

type QueryParams struct {
	Names  []string
	Values []any
}

// Set sets a parameter, and overrides its value if it was already set
func (params *QueryParams) Set(name string, value any) *QueryParams {
	for i, existingName := range params.Names {
		if existingName == name {
			params.Values[i] = value

			return params
		}
	}

	return params.Append(name, value)
}

func (params *QueryParams) GetInt(name string, defaultTo int) int {
	for i, existingName := range params.Names {
		if existingName == name {
			return params.Values[i].(int)
		}
	}

	return defaultTo
}

// Append adds a new parameter, even if one already exists with the same name
func (params *QueryParams) Append(name string, value any) *QueryParams {
	params.Names = append(params.Names, name)
	params.Values = append(params.Values, value)

	return params
}

// String returns all parameters as one string, ready to be appended to the URL
func (params *QueryParams) String() string {
	if params == nil {
		return ""
	}
	var parts []string
	for i, name := range params.Names {
		value := params.Values[i]
		parts = append(parts,
			fmt.Sprintf("%s=%v", url.QueryEscape(name), url.QueryEscape(fmt.Sprint(value))))
	}
	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "&")
}

type RequestOptions struct {
	Body                any
	Params              *QueryParams
	Prefix              string
	PageSize            int
	NoMetadata          bool
	NoAutoFetchNextPage bool
}

func (client *Client) SendRequest(method, url string, result any, options *RequestOptions) error {
	if options == nil {
		options = &RequestOptions{}
	}
	fullURL := client.getFullURL(url, options)
	var body io.Reader = nil
	if options.Body != nil {
		bodyBytes, err := json.Marshal(options.Body)
		if err != nil {
			return fmt.Errorf("failed to unmarshal request body: %w", err)
		}
		body = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return err
	}

	req = req.WithContext(context.Background())
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Token "+client.apiKey)

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

func (client *Client) Download(url, fileName string) error {
	utils.UserOutput("Downloading " + fileName)
	res, err := client.requestBody(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	destFile, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to open destination file: %w", err)
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, res)
	if err != nil {
		return fmt.Errorf("failed to write to destination file: %w", err)
	}

	return nil
}

func (client *Client) requestBody(url string) (io.ReadCloser, error) {
	fullURL := client.getFullURL(url, &RequestOptions{})
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(context.Background())
	req.Header.Set("Authorization", "Token "+client.apiKey)

	logger.Debug().
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Msg("Request")

	res, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		logger.Error().
			Str("method", req.Method).
			Str("url", req.URL.String()).
			Int("status", res.StatusCode).
			Msg("Response")

		return nil, fmt.Errorf("%s %s returned HTTP %d", req.Method, req.URL, res.StatusCode)
	}
	logger.Debug().
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Int("status", res.StatusCode).
		Msg("Response")

	var reader io.ReadCloser
	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(res.Body)
		if err != nil {
			return nil, err
		}
	default:
		reader = res.Body
	}

	return reader, nil
}

func (client *Client) DownloadMany(urlTemplate string, fileNames []string) error {
	sem := semaphore.NewWeighted(maxConcurrentDownloads)
	baseContext := context.Background()
	wg := sync.WaitGroup{}
	for _, file := range fileNames {
		wg.Add(1)
		_ = sem.Acquire(baseContext, 1)
		go func(file string) {
			client.Download(fmt.Sprintf(urlTemplate, file), file)
			wg.Done()
			sem.Release(1)
		}(file)
	}
	wg.Wait()

	return nil
}

// Get sends a GET request, and does not expect the response to be enveloped
func (client *Client) Get(url string, result any, options *RequestOptions) error {
	return client.SendRequest("GET", url, result, options)
}

// GetAPIEntity is a general implementation for getting a single object from an
// API resource
func (client *Client) GetAPIEntity(resource string, id, result any) error {
	entity := responseEnvelope{
		Data: entityEnvelope{
			Attributes: result,
		},
	}

	return client.Get(fmt.Sprintf("%s/%v", resource, id), &entity, nil)
}

// Post sends a POST request, and does not expect the response to be enveloped
func (client *Client) Post(url string, result any, options *RequestOptions) error {
	return client.SendRequest("POST", url, result, options)
}

func (client *Client) Read(url string) (io.ReadCloser, error) {
	return client.requestBody(url)
}
