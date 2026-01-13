package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// EmbeddingService provides embedding capabilities to sandbox scripts
type EmbeddingService interface {
	// Embed converts texts to vector embeddings
	Embed(provider, model string, texts []string) (*EmbeddingResult, error)
}

// EmbeddingResult contains the embedding response
type EmbeddingResult struct {
	Vectors   [][]float32   `json:"vectors"`
	Model     string        `json:"model"`
	Dimension int           `json:"dimension"`
	Usage     EmbeddingUsage `json:"usage"`
}

// EmbeddingUsage tracks token usage for embeddings
type EmbeddingUsage struct {
	TotalTokens int `json:"total_tokens"`
}

// EmbeddingServiceImpl implements EmbeddingService
type EmbeddingServiceImpl struct {
	httpClient *http.Client
	ctx        context.Context
}

// NewEmbeddingService creates a new EmbeddingService
func NewEmbeddingService(ctx context.Context) *EmbeddingServiceImpl {
	return &EmbeddingServiceImpl{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		ctx: ctx,
	}
}

// Embed converts texts to vector embeddings using the specified provider
func (s *EmbeddingServiceImpl) Embed(provider, model string, texts []string) (*EmbeddingResult, error) {
	switch provider {
	case "openai":
		return s.embedOpenAI(model, texts)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", provider)
	}
}

// embedOpenAI calls OpenAI's embedding API
func (s *EmbeddingServiceImpl) embedOpenAI(model string, texts []string) (*EmbeddingResult, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	// OpenAI embedding request
	type embeddingRequest struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}

	reqBody := embeddingRequest{
		Model: model,
		Input: texts,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(s.ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// OpenAI embedding response
	type embeddingData struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	}

	type embeddingResponse struct {
		Data  []embeddingData `json:"data"`
		Model string          `json:"model"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	var respData embeddingResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract vectors in order
	vectors := make([][]float32, len(respData.Data))
	for _, d := range respData.Data {
		vectors[d.Index] = d.Embedding
	}

	// Determine dimension from first vector
	dimension := 0
	if len(vectors) > 0 && len(vectors[0]) > 0 {
		dimension = len(vectors[0])
	}

	return &EmbeddingResult{
		Vectors:   vectors,
		Model:     respData.Model,
		Dimension: dimension,
		Usage: EmbeddingUsage{
			TotalTokens: respData.Usage.TotalTokens,
		},
	}, nil
}
