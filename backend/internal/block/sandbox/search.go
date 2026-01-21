package sandbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// SearchService provides web search capabilities to scripts using Tavily API
type SearchService struct {
	apiKey string
	client *http.Client
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// NewSearchService creates a new SearchService
// Reads configuration from environment variable:
// - TAVILY_API_KEY: Tavily API key
func NewSearchService() *SearchService {
	return &SearchService{
		apiKey: os.Getenv("TAVILY_API_KEY"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// tavilyRequest represents the request body for Tavily API
type tavilyRequest struct {
	APIKey            string `json:"api_key"`
	Query             string `json:"query"`
	SearchDepth       string `json:"search_depth"`
	IncludeAnswer     bool   `json:"include_answer"`
	IncludeRawContent bool   `json:"include_raw_content"`
	MaxResults        int    `json:"max_results"`
}

// tavilyResponse represents the response from Tavily API
type tavilyResponse struct {
	Query   string `json:"query"`
	Results []struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
	Error string `json:"error,omitempty"`
}

// Search performs a web search using Tavily API
// query: the search query string
// numResults: number of results to return (1-10, default 5)
// Returns a slice of SearchResult or error
func (s *SearchService) Search(query string, numResults int) ([]SearchResult, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("Tavily API not configured. Set TAVILY_API_KEY environment variable")
	}

	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Validate and set default for numResults
	if numResults <= 0 || numResults > 10 {
		numResults = 5
	}

	// Build the request body
	reqBody := tavilyRequest{
		APIKey:            s.apiKey,
		Query:             query,
		SearchDepth:       "basic",
		IncludeAnswer:     false,
		IncludeRawContent: false,
		MaxResults:        numResults,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make the request
	req, err := http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}

	// Parse the response
	var apiResponse tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	// Check for API-level errors
	if apiResponse.Error != "" {
		return nil, fmt.Errorf("search API error: %s", apiResponse.Error)
	}

	// Convert to SearchResult slice
	results := make([]SearchResult, len(apiResponse.Results))
	for i, item := range apiResponse.Results {
		results[i] = SearchResult{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: item.Content,
		}
	}

	return results, nil
}

// IsConfigured returns true if the search service has valid configuration
func (s *SearchService) IsConfigured() bool {
	return s.apiKey != ""
}
