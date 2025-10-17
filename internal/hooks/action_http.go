package hooks

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPWebhookAction sends HTTP POST requests with structured data
type HTTPWebhookAction struct{}

func (a *HTTPWebhookAction) Name() ActionName {
	return ActionHTTPWebhook
}

func (a *HTTPWebhookAction) Description() string {
	return "Send HTTP request to webhook URL"
}

func (a *HTTPWebhookAction) Validate(params map[string]interface{}) error {
	// Required: url
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("'url' parameter is required and must be a string")
	}

	// Optional: method
	method, _ := params["method"].(string)
	if method == "" {
		method = "POST"
	}
	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true}
	if !validMethods[method] {
		return fmt.Errorf("invalid method: %s (must be GET, POST, PUT, or PATCH)", method)
	}

	// Optional: headers
	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("header '%s' must be a string", key)
			}
			if err := ValidateString(key); err != nil {
				return fmt.Errorf("invalid header name '%s': %w", key, err)
			}
		}
	}

	// Optional: body
	if body, ok := params["body"].(string); ok {
		if len(body) > 10*1024 {
			return fmt.Errorf("body size exceeds maximum (10KB)")
		}
		if err := ValidateString(body); err != nil {
			return fmt.Errorf("invalid body: %w", err)
		}
	}

	// Optional: timeout
	if timeout, ok := params["timeout"].(string); ok {
		if _, err := time.ParseDuration(timeout); err != nil {
			return fmt.Errorf("invalid timeout format: %s", timeout)
		}
	}

	return nil
}

func (a *HTTPWebhookAction) Execute(ctx *HookContext, params map[string]interface{}) error {
	// Get parameters
	url := params["url"].(string)
	method, _ := params["method"].(string)
	if method == "" {
		method = "POST"
	}

	// Interpolate URL
	url = interpolateString(url, ctx.Variables)

	// Validate URL with settings
	if err := ValidateURL(url, ctx.Settings.AllowHTTP, ctx.Settings.AllowInternalIPs); err != nil {
		return err
	}

	// Get body
	var bodyReader io.Reader
	if body, ok := params["body"].(string); ok {
		interpolatedBody := interpolateString(body, ctx.Variables)
		bodyReader = bytes.NewBufferString(interpolatedBody)
	}

	// Create request
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "goenv-hooks/1.0")
	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			headerValue := value.(string)
			interpolatedValue := interpolateString(headerValue, ctx.Variables)
			req.Header.Set(key, interpolatedValue)
		}
	}

	// Set Content-Type if not specified and we have a body
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Get timeout
	timeout := 5 * time.Second
	if timeoutStr, ok := params["timeout"].(string); ok {
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsed
		}
	}

	// Execute request with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
