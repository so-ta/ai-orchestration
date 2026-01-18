package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPAdapter(t *testing.T) {
	adapter := NewHTTPAdapter()

	assert.NotNil(t, adapter)
	assert.Equal(t, "http", adapter.ID())
	assert.Equal(t, "HTTP Request", adapter.Name())
}

func TestHTTPAdapter_Execute_GET(t *testing.T) {
	// Create mock server
	// Note: Template variable substitution is now handled by Executor, not the adapter.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/users", r.URL.Path)
		assert.Equal(t, "123", r.URL.Query().Get("id"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   123,
			"name": "Test User",
		})
	}))
	defer server.Close()

	adapter := NewHTTPAdapter()

	// Use pre-expanded values (as Executor would provide)
	config := HTTPConfig{
		URL:    server.URL + "/api/users",
		Method: "GET",
		QueryParams: map[string]string{
			"id": "123",
		},
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{"user_id": "123"}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "http", resp.Metadata["adapter"])
	assert.Equal(t, "200", resp.Metadata["status_code"])

	var output HTTPOutput
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)
	assert.Equal(t, 200, output.StatusCode)
	assert.NotNil(t, output.Body)
}

func TestHTTPAdapter_Execute_POST(t *testing.T) {
	// Create mock server
	// Note: Template variable substitution is now handled by Executor, not the adapter.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Test Message", body["message"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      456,
		})
	}))
	defer server.Close()

	adapter := NewHTTPAdapter()

	// Use pre-expanded values (as Executor would provide)
	config := HTTPConfig{
		URL:    server.URL + "/api/messages",
		Method: "POST",
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
		},
		Body:     `{"message": "Test Message"}`,
		BodyType: "json",
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "201", resp.Metadata["status_code"])

	var output HTTPOutput
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)
	assert.Equal(t, 201, output.StatusCode)
}

func TestHTTPAdapter_Execute_ErrorStatus(t *testing.T) {
	// Create mock server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Not found",
		})
	}))
	defer server.Close()

	adapter := NewHTTPAdapter()

	config := HTTPConfig{
		URL:    server.URL + "/api/missing",
		Method: "GET",
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	// Should return error for 4xx status
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
	// But still provide response data
	assert.NotNil(t, resp)
	assert.Equal(t, "404", resp.Metadata["status_code"])
}

func TestHTTPAdapter_Execute_Timeout(t *testing.T) {
	// Create mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewHTTPAdapter()

	config := HTTPConfig{
		URL:        server.URL + "/slow",
		Method:     "GET",
		TimeoutSec: 1, // 1 second timeout
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestHTTPAdapter_Execute_ContextCancellation(t *testing.T) {
	// Create mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewHTTPAdapter()

	config := HTTPConfig{
		URL:    server.URL + "/slow",
		Method: "GET",
	}
	configJSON, _ := json.Marshal(config)

	ctx, cancel := context.WithCancel(context.Background())

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	// Cancel immediately
	cancel()

	resp, err := adapter.Execute(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestHTTPAdapter_Execute_InvalidConfig(t *testing.T) {
	adapter := NewHTTPAdapter()

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: json.RawMessage(`invalid json`),
	}

	resp, err := adapter.Execute(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid HTTP config")
}

func TestHTTPAdapter_Execute_MissingURL(t *testing.T) {
	adapter := NewHTTPAdapter()

	config := HTTPConfig{
		Method: "GET",
		// URL is missing
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestHTTPAdapter_Execute_VariableSubstitution(t *testing.T) {
	// Create mock server
	// Note: Template variable substitution is now handled by Executor, not the adapter.
	// This test verifies that pre-expanded values work correctly.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/users/user-123", r.URL.Path)
		assert.Equal(t, "custom-agent", r.Header.Get("User-Agent"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter()

	// Use pre-expanded values (as Executor would provide)
	config := HTTPConfig{
		URL:    server.URL + "/api/users/user-123",
		Method: "GET",
		Headers: map[string]string{
			"User-Agent": "custom-agent",
		},
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestHTTPAdapter_Execute_NonJSONResponse(t *testing.T) {
	// Create mock server that returns plain text
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter()

	config := HTTPConfig{
		URL:    server.URL + "/text",
		Method: "GET",
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	var output HTTPOutput
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", output.BodyRaw)
	assert.Nil(t, output.Body) // Should be nil for non-JSON response
}

func TestHTTPAdapter_InputSchema(t *testing.T) {
	adapter := NewHTTPAdapter()
	schema := adapter.InputSchema()

	assert.NotNil(t, schema)

	var parsed map[string]interface{}
	err := json.Unmarshal(schema, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "object", parsed["type"])
}

func TestHTTPAdapter_OutputSchema(t *testing.T) {
	adapter := NewHTTPAdapter()
	schema := adapter.OutputSchema()

	assert.NotNil(t, schema)

	var parsed map[string]interface{}
	err := json.Unmarshal(schema, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "object", parsed["type"])
}

func TestHTTPAdapter_Execute_PUT(t *testing.T) {
	// Create mock server
	// Note: Template variable substitution is now handled by Executor, not the adapter.
	// This test uses pre-expanded values.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/api/users/123", r.URL.Path)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Name", body["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   123,
			"name": "Updated Name",
		})
	}))
	defer server.Close()

	adapter := NewHTTPAdapter()

	// Use pre-expanded values (as Executor would provide)
	config := HTTPConfig{
		URL:    server.URL + "/api/users/123",
		Method: "PUT",
		Body:   `{"name": "Updated Name"}`,
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "PUT", resp.Metadata["method"])
}

func TestHTTPAdapter_Execute_DELETE(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	adapter := NewHTTPAdapter()

	config := HTTPConfig{
		URL:    server.URL + "/api/users/123",
		Method: "DELETE",
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "204", resp.Metadata["status_code"])
}
