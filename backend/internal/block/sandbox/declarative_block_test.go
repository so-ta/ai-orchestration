package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Test Helpers
// ============================================================================

// MockAPIServer creates a configurable mock server for testing block execution
type MockAPIServer struct {
	server   *httptest.Server
	requests []RecordedRequest
	handlers map[string]http.HandlerFunc
}

// RecordedRequest stores details of received requests for verification
type RecordedRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    map[string]interface{}
}

// NewMockAPIServer creates a new mock API server
func NewMockAPIServer() *MockAPIServer {
	m := &MockAPIServer{
		handlers: make(map[string]http.HandlerFunc),
		requests: make([]RecordedRequest, 0),
	}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body once and store it for both recording and handler
		var body map[string]interface{}
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			if len(bodyBytes) > 0 {
				json.Unmarshal(bodyBytes, &body)
			}
		}

		// Record the request
		m.requests = append(m.requests, RecordedRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Headers: r.Header.Clone(),
			Body:    body,
		})

		// Restore body for handler (create new reader from bytes)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// Find and execute handler
		key := r.Method + " " + r.URL.Path
		if handler, ok := m.handlers[key]; ok {
			handler(w, r)
			return
		}

		// Check for pattern handlers (e.g., "POST /repos/*/issues")
		for pattern, handler := range m.handlers {
			if matchPattern(pattern, key) {
				handler(w, r)
				return
			}
		}

		// Default 404
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))

	return m
}

// matchPattern checks if a request key matches a pattern with wildcards
func matchPattern(pattern, key string) bool {
	patternParts := strings.Split(pattern, "/")
	keyParts := strings.Split(key, "/")

	if len(patternParts) != len(keyParts) {
		return false
	}

	for i, part := range patternParts {
		if part == "*" {
			continue
		}
		if part != keyParts[i] {
			return false
		}
	}
	return true
}

// Handle registers a handler for a specific method and path
func (m *MockAPIServer) Handle(method, path string, handler http.HandlerFunc) {
	m.handlers[method+" "+path] = handler
}

// HandleJSON registers a handler that returns JSON
func (m *MockAPIServer) HandleJSON(method, path string, statusCode int, response interface{}) {
	m.handlers[method+" "+path] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}
}

// URL returns the server URL
func (m *MockAPIServer) URL() string {
	return m.server.URL
}

// Close shuts down the server
func (m *MockAPIServer) Close() {
	m.server.Close()
}

// GetRequests returns all recorded requests
func (m *MockAPIServer) GetRequests() []RecordedRequest {
	return m.requests
}

// LastRequest returns the last recorded request
func (m *MockAPIServer) LastRequest() *RecordedRequest {
	if len(m.requests) == 0 {
		return nil
	}
	return &m.requests[len(m.requests)-1]
}

// ClearRequests clears all recorded requests
func (m *MockAPIServer) ClearRequests() {
	m.requests = make([]RecordedRequest, 0)
}

// createTestExecutionContext creates an ExecutionContext for testing
func createTestExecutionContext(serverURL string, credentials map[string]interface{}) *ExecutionContext {
	httpClient := NewHTTPClient(30 * time.Second)
	return &ExecutionContext{
		HTTP:        httpClient,
		Credentials: credentials,
	}
}

// ============================================================================
// GitHub Block Tests (Declarative Configuration)
// ============================================================================

func TestDeclarativeBlock_GitHubCreateIssue(t *testing.T) {
	// Setup mock server
	mock := NewMockAPIServer()
	defer mock.Close()

	// Register GitHub API response
	mock.Handle("POST", "/repos/octocat/hello-world/issues", func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Contains(t, r.Header.Get("Accept"), "application/json")

		// Parse and verify body
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Test Issue Title", body["title"])
		assert.Equal(t, "This is the issue body", body["body"])

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       12345,
			"number":   1,
			"url":      "https://api.github.com/repos/octocat/hello-world/issues/1",
			"html_url": "https://github.com/octocat/hello-world/issues/1",
			"state":    "open",
		})
	})

	// Create block definition with declarative config
	block := &domain.BlockDefinition{
		Slug: "github_create_issue",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/repos/{{owner}}/{{repo}}/issues",
			Method: "POST",
			Body: map[string]interface{}{
				"title": "{{input.title}}",
				"body":  "{{input.body}}",
			},
			Headers: map[string]string{
				"Authorization": "Bearer {{secret.token}}",
				"Accept":        "application/json",
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200, 201},
			OutputMapping: map[string]string{
				"id":       "body.id",
				"number":   "body.number",
				"url":      "body.url",
				"html_url": "body.html_url",
			},
		},
	}

	// Create sandbox and execution context
	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), map[string]interface{}{
		"token": "test-token",
	})

	// Execute block
	config := map[string]interface{}{
		"owner": "octocat",
		"repo":  "hello-world",
	}
	input := map[string]interface{}{
		"title": "Test Issue Title",
		"body":  "This is the issue body",
	}

	result, err := sb.ExecuteWithDeclarative(context.Background(), block, config, input, execCtx)
	require.NoError(t, err)

	// Verify result
	assert.EqualValues(t, float64(12345), result["id"])
	assert.EqualValues(t, float64(1), result["number"])
	assert.Equal(t, "https://api.github.com/repos/octocat/hello-world/issues/1", result["url"])
	assert.Equal(t, "https://github.com/octocat/hello-world/issues/1", result["html_url"])

	// Verify request was made correctly
	req := mock.LastRequest()
	require.NotNil(t, req)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/repos/octocat/hello-world/issues", req.Path)
}

func TestDeclarativeBlock_GitHubAddComment(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	mock.HandleJSON("POST", "/repos/octocat/hello-world/issues/42/comments", 201, map[string]interface{}{
		"id":       98765,
		"url":      "https://api.github.com/repos/octocat/hello-world/issues/comments/98765",
		"html_url": "https://github.com/octocat/hello-world/issues/42#issuecomment-98765",
		"body":     "Great work!",
	})

	block := &domain.BlockDefinition{
		Slug: "github_add_comment",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/repos/{{owner}}/{{repo}}/issues/{{issue_number}}/comments",
			Method: "POST",
			Body: map[string]interface{}{
				"body": "{{input.comment}}",
			},
			Headers: map[string]string{
				"Authorization": "Bearer {{secret.github_token}}",
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{201},
			OutputMapping: map[string]string{
				"id":       "body.id",
				"html_url": "body.html_url",
			},
		},
	}

	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), map[string]interface{}{
		"github_token": "ghp_xxxx",
	})

	config := map[string]interface{}{
		"owner":        "octocat",
		"repo":         "hello-world",
		"issue_number": "42",
	}
	input := map[string]interface{}{
		"comment": "Great work!",
	}

	result, err := sb.ExecuteWithDeclarative(context.Background(), block, config, input, execCtx)
	require.NoError(t, err)

	assert.EqualValues(t, float64(98765), result["id"])
	assert.Contains(t, result["html_url"], "issuecomment-98765")
}

// ============================================================================
// Webhook Block Tests (Slack, Discord)
// ============================================================================

func TestDeclarativeBlock_SlackWebhook(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	mock.Handle("POST", "/webhook/slack", func(w http.ResponseWriter, r *http.Request) {
		// Verify content type
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse body
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Hello from test!", body["text"])

		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	block := &domain.BlockDefinition{
		Slug: "slack",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/webhook/slack",
			Method: "POST",
			Body: map[string]interface{}{
				"text":    "{{input.message}}",
				"channel": "{{channel}}",
			},
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200, 204},
			OutputMapping: map[string]string{
				"success": "true",
				"status":  "status",
			},
		},
	}

	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), nil)

	config := map[string]interface{}{
		"channel": "#general",
	}
	input := map[string]interface{}{
		"message": "Hello from test!",
	}

	result, err := sb.ExecuteWithDeclarative(context.Background(), block, config, input, execCtx)
	require.NoError(t, err)

	assert.Equal(t, true, result["success"])
	assert.EqualValues(t, 200, result["status"])
}

func TestDeclarativeBlock_DiscordWebhook(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	mock.Handle("POST", "/webhook/discord", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Discord test message", body["content"])
		assert.Equal(t, "TestBot", body["username"])

		w.WriteHeader(204) // Discord returns 204 on success
	})

	block := &domain.BlockDefinition{
		Slug: "discord",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/webhook/discord",
			Method: "POST",
			Body: map[string]interface{}{
				"content":  "{{input.content}}",
				"username": "{{username}}",
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200, 204},
			OutputMapping: map[string]string{
				"success": "true",
			},
		},
	}

	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), nil)

	config := map[string]interface{}{
		"username": "TestBot",
	}
	input := map[string]interface{}{
		"content": "Discord test message",
	}

	result, err := sb.ExecuteWithDeclarative(context.Background(), block, config, input, execCtx)
	require.NoError(t, err)

	assert.Equal(t, true, result["success"])
}

// ============================================================================
// REST API Block Tests (Bearer Auth, API Key Auth)
// ============================================================================

func TestDeclarativeBlock_BearerAuth(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	mock.Handle("GET", "/api/v1/user", func(w http.ResponseWriter, r *http.Request) {
		// Verify Bearer token
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-token-123" {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    1,
			"name":  "John Doe",
			"email": "john@example.com",
		})
	})

	block := &domain.BlockDefinition{
		Slug: "bearer-api-test",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/api/v1/user",
			Method: "GET",
			Headers: map[string]string{
				"Authorization": "Bearer {{secret.api_token}}",
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200},
			OutputMapping: map[string]string{
				"user_id":   "body.id",
				"user_name": "body.name",
				"email":     "body.email",
			},
		},
	}

	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), map[string]interface{}{
		"api_token": "valid-token-123",
	})

	result, err := sb.ExecuteWithDeclarative(context.Background(), block, map[string]interface{}{}, map[string]interface{}{}, execCtx)
	require.NoError(t, err)

	assert.EqualValues(t, float64(1), result["user_id"])
	assert.Equal(t, "John Doe", result["user_name"])
	assert.Equal(t, "john@example.com", result["email"])
}

func TestDeclarativeBlock_APIKeyHeader(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	mock.Handle("GET", "/api/search", func(w http.ResponseWriter, r *http.Request) {
		// Verify API key header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "sk-test-api-key" {
			w.WriteHeader(401)
			return
		}

		query := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"results": []string{"result1", "result2"},
		})
	})

	block := &domain.BlockDefinition{
		Slug: "api-key-test",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/api/search",
			Method: "GET",
			Headers: map[string]string{
				"X-API-Key": "{{secret.api_key}}",
			},
			QueryParams: map[string]string{
				"q": "{{input.query}}",
			},
		},
		Response: &domain.ResponseConfig{
			OutputMapping: map[string]string{
				"query":   "body.query",
				"results": "body.results",
			},
		},
	}

	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), map[string]interface{}{
		"api_key": "sk-test-api-key",
	})

	input := map[string]interface{}{
		"query": "test search",
	}

	result, err := sb.ExecuteWithDeclarative(context.Background(), block, map[string]interface{}{}, input, execCtx)
	require.NoError(t, err)

	assert.Equal(t, "test search", result["query"])
	results := result["results"].([]interface{})
	assert.Len(t, results, 2)
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestDeclarativeBlock_HTTPError(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	mock.HandleJSON("POST", "/api/error", 400, map[string]interface{}{
		"error":   "bad_request",
		"message": "Invalid parameters",
	})

	block := &domain.BlockDefinition{
		Slug: "error-test",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/api/error",
			Method: "POST",
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200, 201},
		},
	}

	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), nil)

	_, err := sb.ExecuteWithDeclarative(context.Background(), block, map[string]interface{}{}, map[string]interface{}{}, execCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestDeclarativeBlock_RateLimited(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	mock.HandleJSON("GET", "/api/rate-limited", 429, map[string]interface{}{
		"error":       "rate_limited",
		"retry_after": 60,
	})

	block := &domain.BlockDefinition{
		Slug: "rate-limit-test",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/api/rate-limited",
			Method: "GET",
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200},
		},
	}

	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), nil)

	_, err := sb.ExecuteWithDeclarative(context.Background(), block, map[string]interface{}{}, map[string]interface{}{}, execCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

// ============================================================================
// Template Expansion Tests
// ============================================================================

func TestDeclarativeBlock_NestedTemplateExpansion(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	mock.Handle("POST", "/api/nested", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// Verify nested structure with safe type assertions
		if data, ok := body["data"].(map[string]interface{}); ok {
			assert.Equal(t, "value1", data["field1"])
			assert.Equal(t, "value2", data["field2"])
		} else {
			t.Errorf("Expected 'data' to be a map, got: %T", body["data"])
		}

		if meta, ok := body["meta"].(map[string]interface{}); ok {
			assert.Equal(t, "test-source", meta["source"])
		} else {
			t.Errorf("Expected 'meta' to be a map, got: %T", body["meta"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	})

	block := &domain.BlockDefinition{
		Slug: "nested-template-test",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/api/nested",
			Method: "POST",
			Body: map[string]interface{}{
				"data": map[string]interface{}{
					"field1": "{{input.f1}}",
					"field2": "{{input.f2}}",
				},
				"meta": map[string]interface{}{
					"source": "{{source}}",
				},
			},
		},
	}

	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), nil)

	config := map[string]interface{}{
		"source": "test-source",
	}
	input := map[string]interface{}{
		"f1": "value1",
		"f2": "value2",
	}

	_, err := sb.ExecuteWithDeclarative(context.Background(), block, config, input, execCtx)
	require.NoError(t, err)
}

// ============================================================================
// Additional Declarative Tests
// ============================================================================

func TestDeclarativeBlock_QueryParams(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	mock.Handle("GET", "/api/list", func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		page := r.URL.Query().Get("page")
		limit := r.URL.Query().Get("limit")
		filter := r.URL.Query().Get("filter")

		assert.Equal(t, "1", page)
		assert.Equal(t, "10", limit)
		assert.Equal(t, "active", filter)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []string{"item1", "item2"},
			"total": 2,
		})
	})

	block := &domain.BlockDefinition{
		Slug: "query-params-test",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/api/list",
			Method: "GET",
			QueryParams: map[string]string{
				"page":   "{{page}}",
				"limit":  "{{limit}}",
				"filter": "{{input.status}}",
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200},
			OutputMapping: map[string]string{
				"items": "body.items",
				"total": "body.total",
			},
		},
	}

	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), nil)

	config := map[string]interface{}{
		"page":  "1",
		"limit": "10",
	}
	input := map[string]interface{}{
		"status": "active",
	}

	result, err := sb.ExecuteWithDeclarative(context.Background(), block, config, input, execCtx)
	require.NoError(t, err)

	items := result["items"].([]interface{})
	assert.Len(t, items, 2)
	assert.EqualValues(t, float64(2), result["total"])
}

func TestDeclarativeBlock_EmptyBody(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	mock.Handle("DELETE", "/api/resource/123", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	block := &domain.BlockDefinition{
		Slug: "delete-test",
		Request: &domain.RequestConfig{
			URL:    mock.URL() + "/api/resource/{{resource_id}}",
			Method: "DELETE",
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{204},
			OutputMapping: map[string]string{
				"success": "true",
			},
		},
	}

	sb := New(DefaultConfig())
	execCtx := createTestExecutionContext(mock.URL(), nil)

	config := map[string]interface{}{
		"resource_id": "123",
	}

	result, err := sb.ExecuteWithDeclarative(context.Background(), block, config, map[string]interface{}{}, execCtx)
	require.NoError(t, err)

	assert.Equal(t, true, result["success"])
}
