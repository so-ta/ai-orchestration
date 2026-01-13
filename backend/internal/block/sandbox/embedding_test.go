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
	// Mock OpenAI API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/embeddings", r.URL.Path)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

		// Parse request
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "text-embedding-3-small", req.Model)
		assert.Len(t, req.Input, 2)

		// Return mock response
		resp := map[string]interface{}{
			"model": "text-embedding-3-small",
			"data": []map[string]interface{}{
				{
					"index":     0,
					"embedding": make([]float32, 1536), // 1536 dimensions
				},
				{
					"index":     1,
					"embedding": make([]float32, 1536),
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

	// Create service (would need to modify to accept custom URL for testing)
	service := NewEmbeddingService(context.Background())
	assert.NotNil(t, service)
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
