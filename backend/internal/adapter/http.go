package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPAdapter implements the Adapter interface for HTTP requests
type HTTPAdapter struct {
	id         string
	name       string
	httpClient *http.Client
}

// HTTPConfig holds the configuration for HTTP adapter
type HTTPConfig struct {
	URL         string            `json:"url"`          // Target URL with {{variable}} placeholders
	Method      string            `json:"method"`       // GET, POST, PUT, PATCH, DELETE
	Headers     map[string]string `json:"headers"`      // Request headers
	Body        string            `json:"body"`         // Request body template (for POST/PUT/PATCH)
	BodyType    string            `json:"body_type"`    // json, form, raw
	QueryParams map[string]string `json:"query_params"` // Query parameters
	TimeoutSec  int               `json:"timeout_sec"`  // Request timeout in seconds
	FollowRedirects bool          `json:"follow_redirects"` // Follow HTTP redirects
}

// HTTPOutput represents the output of an HTTP request
type HTTPOutput struct {
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}       `json:"body"`
	BodyRaw    string            `json:"body_raw"`
	DurationMs int               `json:"duration_ms"`
}

// NewHTTPAdapter creates a new HTTP adapter
func NewHTTPAdapter() *HTTPAdapter {
	return &HTTPAdapter{
		id:   "http",
		name: "HTTP Request",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a *HTTPAdapter) ID() string   { return a.id }
func (a *HTTPAdapter) Name() string { return a.name }

// Execute runs the HTTP adapter
func (a *HTTPAdapter) Execute(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	// Parse config
	var config HTTPConfig
	if req.Config != nil {
		if err := json.Unmarshal(req.Config, &config); err != nil {
			return nil, fmt.Errorf("invalid HTTP config: %w", err)
		}
	}

	// Validate config
	if config.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}
	if config.Method == "" {
		config.Method = "GET"
	}
	config.Method = strings.ToUpper(config.Method)

	// Set default timeout
	if config.TimeoutSec <= 0 {
		config.TimeoutSec = 30
	}

	// Config templates are now expanded by Executor before reaching the adapter
	// All config values can be used directly

	// Build URL with query parameters
	url := config.URL
	if len(config.QueryParams) > 0 {
		params := make([]string, 0, len(config.QueryParams))
		for key, value := range config.QueryParams {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
		if strings.Contains(url, "?") {
			url += "&" + strings.Join(params, "&")
		} else {
			url += "?" + strings.Join(params, "&")
		}
	}

	// Build request body
	var bodyReader io.Reader
	if config.Body != "" && (config.Method == "POST" || config.Method == "PUT" || config.Method == "PATCH") {
		bodyReader = bytes.NewBufferString(config.Body)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, config.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range config.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set content type if body is present
	if bodyReader != nil && httpReq.Header.Get("Content-Type") == "" {
		switch config.BodyType {
		case "form":
			httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		case "raw":
			httpReq.Header.Set("Content-Type", "text/plain")
		default:
			httpReq.Header.Set("Content-Type", "application/json")
		}
	}

	// Configure client
	client := a.httpClient
	if config.TimeoutSec > 0 {
		client = &http.Client{
			Timeout: time.Duration(config.TimeoutSec) * time.Second,
		}
		if !config.FollowRedirects {
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}
	}

	// Execute request
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response headers
	respHeaders := make(map[string]string)
	for key := range resp.Header {
		respHeaders[key] = resp.Header.Get(key)
	}

	// Try to parse body as JSON
	var parsedBody interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &parsedBody); err != nil {
			// Not JSON, use raw string
			parsedBody = nil
		}
	}

	// Build output
	output := HTTPOutput{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    respHeaders,
		Body:       parsedBody,
		BodyRaw:    string(respBody),
		DurationMs: int(time.Since(start).Milliseconds()),
	}

	outputJSON, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	// Check for error status codes
	metadata := map[string]string{
		"adapter":     a.id,
		"status_code": fmt.Sprintf("%d", resp.StatusCode),
		"method":      config.Method,
	}

	// Return error for 4xx/5xx status codes
	if resp.StatusCode >= 400 {
		return &Response{
			Output:     outputJSON,
			DurationMs: int(time.Since(start).Milliseconds()),
			Metadata:   metadata,
		}, fmt.Errorf("HTTP request returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return &Response{
		Output:     outputJSON,
		DurationMs: int(time.Since(start).Milliseconds()),
		Metadata:   metadata,
	}, nil
}

func (a *HTTPAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"description": "Input data for variable substitution in URL, headers, and body templates",
		"additionalProperties": true
	}`)
}

func (a *HTTPAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"status_code": {"type": "integer", "description": "HTTP status code"},
			"status": {"type": "string", "description": "HTTP status text"},
			"headers": {
				"type": "object",
				"description": "Response headers",
				"additionalProperties": {"type": "string"}
			},
			"body": {"description": "Parsed JSON body (null if not JSON)"},
			"body_raw": {"type": "string", "description": "Raw response body"},
			"duration_ms": {"type": "integer", "description": "Request duration in milliseconds"}
		},
		"required": ["status_code", "status", "body_raw", "duration_ms"]
	}`)
}
