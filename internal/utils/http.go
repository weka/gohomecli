package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// HTTPClient provides common HTTP operations
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient creates a new HTTP client
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: http.DefaultClient,
	}
}

// doGet performs an HTTP GET request and returns the response body reader
// Caller is responsible for closing the response body
func (c *HTTPClient) doGet(ctx context.Context, url string, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return resp, nil
}

// Get performs an HTTP GET request with timeout and returns the response body
func (c *HTTPClient) Get(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	resp, err := c.doGet(ctx, url, timeout)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// DownloadFile downloads a file from URL to local path with timeout
func (c *HTTPClient) DownloadFile(ctx context.Context, url, localPath string, timeout time.Duration) error {
	resp, err := c.doGet(ctx, url, timeout)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}

	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
