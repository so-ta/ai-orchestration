package testing

import (
	"fmt"
	"time"

	"github.com/souta/ai-orchestration/internal/block/sandbox"
)

// CreateMockExecutionContext creates a mock execution context for testing block code
func CreateMockExecutionContext() *sandbox.ExecutionContext {
	return &sandbox.ExecutionContext{
		HTTP:      sandbox.NewHTTPClient(30 * time.Second),
		LLM:       &MockLLMService{},
		Workflow:  &MockWorkflowService{},
		Human:     &MockHumanService{},
		Adapter:   &MockAdapterService{},
		Embedding: &MockEmbeddingService{},
		Vector:    &MockVectorService{},
		Blocks:    &MockBlocksService{},
		Workflows: &MockWorkflowsService{},
		Runs:      &MockRunsService{},
		Logger:    func(args ...interface{}) {},
	}
}

// MockLLMService mocks the LLM API
type MockLLMService struct {
	Response map[string]interface{}
	Error    error
}

func (m *MockLLMService) Chat(provider, model string, request map[string]interface{}) (map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Response != nil {
		return m.Response, nil
	}
	// Default mock response
	return map[string]interface{}{
		"content": "Mock LLM response",
		"usage": map[string]interface{}{
			"input_tokens":  10,
			"output_tokens": 20,
		},
	}, nil
}

// MockWorkflowService mocks the Workflow service
type MockWorkflowService struct {
	Response            map[string]interface{}
	ExecuteStepResponse map[string]interface{}
	Error               error
}

func (m *MockWorkflowService) Run(workflowID string, input map[string]interface{}) (map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Response != nil {
		return m.Response, nil
	}
	return map[string]interface{}{
		"result": "Mock workflow result",
	}, nil
}

func (m *MockWorkflowService) ExecuteStep(stepName string, input map[string]interface{}) (map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.ExecuteStepResponse != nil {
		return m.ExecuteStepResponse, nil
	}
	return map[string]interface{}{
		"result":    "Mock step result",
		"step_name": stepName,
	}, nil
}

// MockHumanService mocks the Human service
type MockHumanService struct {
	Response map[string]interface{}
	Error    error
}

func (m *MockHumanService) RequestApproval(request map[string]interface{}) (map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Response != nil {
		return m.Response, nil
	}
	return map[string]interface{}{
		"approved": true,
		"comment":  "Mock approval",
	}, nil
}

// MockAdapterService mocks the Adapter service
type MockAdapterService struct {
	Response map[string]interface{}
	Error    error
}

func (m *MockAdapterService) Call(adapterID string, input map[string]interface{}) (map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Response != nil {
		return m.Response, nil
	}
	return map[string]interface{}{
		"result": "Mock adapter result",
	}, nil
}

// MockEmbeddingService mocks the Embedding service
type MockEmbeddingService struct {
	Response *sandbox.EmbeddingResult
	Error    error
}

func (m *MockEmbeddingService) Embed(provider, model string, texts []string) (*sandbox.EmbeddingResult, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Response != nil {
		return m.Response, nil
	}
	// Generate mock vectors
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = make([]float32, 1536) // OpenAI embedding dimension
		for j := range vectors[i] {
			vectors[i][j] = float32(i+j) * 0.001
		}
	}
	return &sandbox.EmbeddingResult{
		Vectors:   vectors,
		Model:     model,
		Dimension: 1536,
		Usage: sandbox.EmbeddingUsage{
			TotalTokens: len(texts) * 10,
		},
	}, nil
}

// MockVectorService mocks the Vector service
type MockVectorService struct {
	UpsertResponse       *sandbox.UpsertResult
	QueryResponse        *sandbox.QueryResult
	DeleteResponse       *sandbox.DeleteResult
	ListCollectionsResp  []sandbox.CollectionInfo
	Error                error
}

func (m *MockVectorService) Upsert(collection string, documents []sandbox.VectorDocument, opts *sandbox.UpsertOptions) (*sandbox.UpsertResult, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.UpsertResponse != nil {
		return m.UpsertResponse, nil
	}
	ids := make([]string, len(documents))
	for i, doc := range documents {
		if doc.ID != "" {
			ids[i] = doc.ID
		} else {
			ids[i] = fmt.Sprintf("mock-id-%d", i)
		}
	}
	return &sandbox.UpsertResult{
		UpsertedCount: len(documents),
		IDs:           ids,
	}, nil
}

func (m *MockVectorService) Query(collection string, vector []float32, opts *sandbox.QueryOptions) (*sandbox.QueryResult, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.QueryResponse != nil {
		return m.QueryResponse, nil
	}
	return &sandbox.QueryResult{
		Matches: []sandbox.QueryMatch{
			{ID: "mock-match-1", Score: 0.95, Content: "Mock content 1"},
			{ID: "mock-match-2", Score: 0.85, Content: "Mock content 2"},
		},
	}, nil
}

func (m *MockVectorService) Delete(collection string, ids []string) (*sandbox.DeleteResult, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.DeleteResponse != nil {
		return m.DeleteResponse, nil
	}
	return &sandbox.DeleteResult{
		DeletedCount: len(ids),
	}, nil
}

func (m *MockVectorService) ListCollections() ([]sandbox.CollectionInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.ListCollectionsResp != nil {
		return m.ListCollectionsResp, nil
	}
	return []sandbox.CollectionInfo{
		{Name: "mock-collection", DocumentCount: 100, Dimension: 1536},
	}, nil
}

// MockBlocksService mocks the Blocks service
type MockBlocksService struct {
	ListResponse          []map[string]interface{}
	GetResponse           map[string]interface{}
	GetWithSchemaResponse map[string]interface{}
	Error                 error
}

func (m *MockBlocksService) List() ([]map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.ListResponse != nil {
		return m.ListResponse, nil
	}
	return []map[string]interface{}{
		{"slug": "llm", "name": "LLM"},
		{"slug": "http", "name": "HTTP"},
	}, nil
}

func (m *MockBlocksService) Get(slug string) (map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.GetResponse != nil {
		return m.GetResponse, nil
	}
	return map[string]interface{}{
		"slug": slug,
		"name": slug,
	}, nil
}

func (m *MockBlocksService) GetWithSchema(slug string) (map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.GetWithSchemaResponse != nil {
		return m.GetWithSchemaResponse, nil
	}
	// Return a mock response with config_schema
	return map[string]interface{}{
		"slug":     slug,
		"name":     slug,
		"category": "custom",
		"config_schema": map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		"required_fields": []string{},
	}, nil
}

// MockWorkflowsService mocks the Workflows service
type MockWorkflowsService struct {
	GetResponse  map[string]interface{}
	ListResponse []map[string]interface{}
	Error        error
}

func (m *MockWorkflowsService) Get(workflowID string) (map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.GetResponse != nil {
		return m.GetResponse, nil
	}
	return map[string]interface{}{
		"id":   workflowID,
		"name": "Mock Workflow",
	}, nil
}

func (m *MockWorkflowsService) List() ([]map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.ListResponse != nil {
		return m.ListResponse, nil
	}
	return []map[string]interface{}{
		{"id": "wf-1", "name": "Workflow 1"},
		{"id": "wf-2", "name": "Workflow 2"},
	}, nil
}

// MockRunsService mocks the Runs service
type MockRunsService struct {
	GetResponse         map[string]interface{}
	GetStepRunsResponse []map[string]interface{}
	Error               error
}

func (m *MockRunsService) Get(runID string) (map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.GetResponse != nil {
		return m.GetResponse, nil
	}
	return map[string]interface{}{
		"id":     runID,
		"status": "completed",
	}, nil
}

func (m *MockRunsService) GetStepRuns(runID string) ([]map[string]interface{}, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.GetStepRunsResponse != nil {
		return m.GetStepRunsResponse, nil
	}
	return []map[string]interface{}{
		{"step_id": "step-1", "status": "completed"},
		{"step_id": "step-2", "status": "completed"},
	}, nil
}
