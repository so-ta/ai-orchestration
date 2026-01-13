package sandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddingService_Embed_OpenAI(t *testing.T) {
	// Track if the server was called
	serverCalled := false

	// Mock OpenAI API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/embeddings", r.URL.Path)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-api-key")

		// Parse request
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "text-embedding-3-small", req.Model)
		assert.Len(t, req.Input, 2)

		// Return mock response with sample vectors
		resp := map[string]interface{}{
			"model": "text-embedding-3-small",
			"data": []map[string]interface{}{
				{
					"index":     0,
					"embedding": []float32{0.1, 0.2, 0.3},
				},
				{
					"index":     1,
					"embedding": []float32{0.4, 0.5, 0.6},
				},
			},
			"usage": map[string]interface{}{
				"total_tokens": 10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Set up test
	t.Setenv("OPENAI_API_KEY", "test-api-key")

	// Create service and set custom URL to mock server
	service := NewEmbeddingService(context.Background())
	service.SetOpenAIBaseURL(server.URL)

	// Call Embed and verify results
	result, err := service.Embed("openai", "text-embedding-3-small", []string{"hello", "world"})
	require.NoError(t, err)

	// Verify server was called
	assert.True(t, serverCalled, "Mock server should have been called")

	// Verify result
	assert.NotNil(t, result)
	assert.Len(t, result.Vectors, 2)
	assert.Equal(t, "text-embedding-3-small", result.Model)
	assert.Equal(t, 3, result.Dimension)
	assert.Equal(t, 10, result.Usage.TotalTokens)
}

func TestEmbeddingResult_Structure(t *testing.T) {
	result := &EmbeddingResult{
		Vectors: [][]float32{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		},
		Model:     "text-embedding-3-small",
		Dimension: 3,
		Usage: EmbeddingUsage{
			TotalTokens: 10,
		},
	}

	assert.Len(t, result.Vectors, 2)
	assert.Equal(t, "text-embedding-3-small", result.Model)
	assert.Equal(t, 3, result.Dimension)
	assert.Equal(t, 10, result.Usage.TotalTokens)
}

func TestEmbeddingService_UnsupportedProvider(t *testing.T) {
	service := NewEmbeddingService(context.Background())

	_, err := service.Embed("unsupported-provider", "model", []string{"test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported embedding provider")
}
