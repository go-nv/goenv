package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NewHTTPClient creates an HTTP client with a reasonable timeout.
// This consolidates the repeated pattern of creating clients with timeouts (10+ occurrences).
//
// Example:
//
//	client := NewHTTPClient(30 * time.Second)
//	resp, err := client.Get("https://example.com")
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}

// NewHTTPClientDefault creates an HTTP client with a 30-second timeout.
// Use this for general-purpose HTTP requests.
func NewHTTPClientDefault() *http.Client {
	return NewHTTPClient(30 * time.Second)
}

// NewHTTPClientForDownloads creates an HTTP client with a longer timeout (10 minutes).
// Use this for large file downloads where the operation may take a while.
func NewHTTPClientForDownloads() *http.Client {
	return NewHTTPClient(10 * time.Minute)
}

// FetchJSON fetches JSON from a URL and unmarshals it into the provided structure.
// This consolidates the repeated pattern of HTTP GET + JSON decode (20+ occurrences).
//
// Example:
//
//	var result MyStruct
//	err := FetchJSON(ctx, "https://api.example.com/data", &result)
func FetchJSON(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := NewHTTPClientDefault()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	return nil
}

// FetchJSONWithTimeout is like FetchJSON but with a custom timeout.
func FetchJSONWithTimeout(url string, target interface{}, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return FetchJSON(ctx, url, target)
}

// FetchBytes fetches raw bytes from a URL.
// Useful for downloading files or getting non-JSON content.
//
// Example:
//
//	data, err := FetchBytes(ctx, "https://example.com/file.txt")
func FetchBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := NewHTTPClientDefault()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}
