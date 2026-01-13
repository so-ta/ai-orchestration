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
// Supported providers: openai, cohere, voyage
func (s *EmbeddingServiceImpl) Embed(provider, model string, texts []string) (*EmbeddingResult, error) {
	switch provider {
	case "openai":
		return s.embedOpenAI(model, texts)
	case "cohere":
		return s.embedCohere(model, texts)
	case "voyage":
		return s.embedVoyage(model, texts)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s (supported: openai, cohere, voyage)", provider)
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

// ============================================================================
// Phase 3.3: Additional Embedding Providers
// ============================================================================

// embedCohere calls Cohere's embedding API
// Models: embed-english-v3.0, embed-multilingual-v3.0, embed-english-light-v3.0
func (s *EmbeddingServiceImpl) embedCohere(model string, texts []string) (*EmbeddingResult, error) {
	apiKey := os.Getenv("COHERE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("COHERE_API_KEY environment variable is not set")
	}

	// Default model
	if model == "" {
		model = "embed-english-v3.0"
	}

	// Cohere embedding request
	type cohereRequest struct {
		Texts     []string `json:"texts"`
		Model     string   `json:"model"`
		InputType string   `json:"input_type"` // search_document, search_query, classification, clustering
	}

	reqBody := cohereRequest{
		Texts:     texts,
		Model:     model,
		InputType: "search_document", // Default for indexing
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(s.ctx, "POST", "https://api.cohere.ai/v1/embed", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Cohere embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Cohere API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Cohere embedding response
	type cohereResponse struct {
		ID         string      `json:"id"`
		Embeddings [][]float32 `json:"embeddings"`
		Texts      []string    `json:"texts"`
		Meta       struct {
			BilledUnits struct {
				InputTokens int `json:"input_tokens"`
			} `json:"billed_units"`
		} `json:"meta"`
	}

	var respData cohereResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse Cohere response: %w", err)
	}

	// Determine dimension from first vector
	dimension := 0
	if len(respData.Embeddings) > 0 && len(respData.Embeddings[0]) > 0 {
		dimension = len(respData.Embeddings[0])
	}

	return &EmbeddingResult{
		Vectors:   respData.Embeddings,
		Model:     model,
		Dimension: dimension,
		Usage: EmbeddingUsage{
			TotalTokens: respData.Meta.BilledUnits.InputTokens,
		},
	}, nil
}

// embedVoyage calls Voyage AI's embedding API
// Models: voyage-3, voyage-3-lite, voyage-code-3, voyage-finance-2, voyage-law-2
func (s *EmbeddingServiceImpl) embedVoyage(model string, texts []string) (*EmbeddingResult, error) {
	apiKey := os.Getenv("VOYAGE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("VOYAGE_API_KEY environment variable is not set")
	}

	// Default model
	if model == "" {
		model = "voyage-3"
	}

	// Voyage embedding request
	type voyageRequest struct {
		Input     []string `json:"input"`
		Model     string   `json:"model"`
		InputType string   `json:"input_type,omitempty"` // document, query
	}

	reqBody := voyageRequest{
		Input:     texts,
		Model:     model,
		InputType: "document", // Default for indexing
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(s.ctx, "POST", "https://api.voyageai.com/v1/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Voyage embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Voyage API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Voyage embedding response (similar to OpenAI format)
	type voyageData struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	}

	type voyageResponse struct {
		Object string       `json:"object"`
		Data   []voyageData `json:"data"`
		Model  string       `json:"model"`
		Usage  struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	var respData voyageResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse Voyage response: %w", err)
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
