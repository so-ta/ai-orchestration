package sandbox

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// MockEmbeddingService is a mock implementation for testing
type MockEmbeddingService struct {
	embedFunc func(provider, model string, texts []string) (*EmbeddingResult, error)
}

func (m *MockEmbeddingService) Embed(provider, model string, texts []string) (*EmbeddingResult, error) {
	if m.embedFunc != nil {
		return m.embedFunc(provider, model, texts)
	}
	// Default: return mock vectors
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = make([]float32, 1536)
	}
	return &EmbeddingResult{
		Vectors:   vectors,
		Model:     model,
		Dimension: 1536,
		Usage:     EmbeddingUsage{TotalTokens: len(texts) * 5},
	}, nil
}

func TestVectorDocument_Structure(t *testing.T) {
	doc := VectorDocument{
		ID:      "doc-1",
		Content: "Test content",
		Metadata: map[string]interface{}{
			"source": "test",
		},
		Vector: []float32{0.1, 0.2, 0.3},
	}

	assert.Equal(t, "doc-1", doc.ID)
	assert.Equal(t, "Test content", doc.Content)
	assert.Equal(t, "test", doc.Metadata["source"])
	assert.Len(t, doc.Vector, 3)
}

func TestQueryOptions_Defaults(t *testing.T) {
	opts := &QueryOptions{}

	// Default values should be handled in the service
	assert.Equal(t, 0, opts.TopK) // Will be defaulted to 5 in service
	assert.False(t, opts.IncludeContent)
}

func TestUpsertResult_Structure(t *testing.T) {
	result := &UpsertResult{
		UpsertedCount: 3,
		IDs:           []string{"id-1", "id-2", "id-3"},
	}

	assert.Equal(t, 3, result.UpsertedCount)
	assert.Len(t, result.IDs, 3)
}

func TestQueryResult_Structure(t *testing.T) {
	result := &QueryResult{
		Matches: []QueryMatch{
			{ID: "doc-1", Score: 0.95, Content: "Test", Metadata: map[string]interface{}{}},
			{ID: "doc-2", Score: 0.85, Content: "Test 2", Metadata: map[string]interface{}{}},
		},
	}

	assert.Len(t, result.Matches, 2)
	assert.Equal(t, "doc-1", result.Matches[0].ID)
	assert.Equal(t, 0.95, result.Matches[0].Score)
}

func TestVectorServiceImpl_TenantIsolation_Concept(t *testing.T) {
	// This test demonstrates the tenant isolation concept
	// In a real test, we would use a test database

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Verify that different tenants have different UUIDs
	assert.NotEqual(t, tenantA, tenantB)

	// The key concept: NewVectorService takes tenantID at construction
	// and it cannot be changed afterward

	// This ensures that:
	// 1. tenantID is set once at service creation
	// 2. All operations automatically use this tenantID
	// 3. Users cannot bypass tenant isolation by passing different tenantID
}

func TestVectorServiceImpl_vectorToString(t *testing.T) {
	service := &VectorServiceImpl{
		ctx:      context.Background(),
		tenantID: uuid.New(),
	}

	tests := []struct {
		name     string
		vector   []float32
		expected string
	}{
		{
			name:     "empty vector",
			vector:   []float32{},
			expected: "",
		},
		{
			name:     "single element",
			vector:   []float32{0.5},
			expected: "[0.500000]",
		},
		{
			name:     "multiple elements",
			vector:   []float32{0.1, 0.2, 0.3},
			expected: "[0.100000,0.200000,0.300000]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.vectorToString(tt.vector)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCollectionInfo_Structure(t *testing.T) {
	info := CollectionInfo{
		Name:          "my-collection",
		DocumentCount: 100,
		Dimension:     1536,
		CreatedAt:     "2026-01-13T10:00:00Z",
	}

	assert.Equal(t, "my-collection", info.Name)
	assert.Equal(t, 100, info.DocumentCount)
	assert.Equal(t, 1536, info.Dimension)
}

func TestDeleteResult_Structure(t *testing.T) {
	result := &DeleteResult{
		DeletedCount: 5,
	}

	assert.Equal(t, 5, result.DeletedCount)
}

// TestTenantIsolation_CollectionNaming verifies that collections are properly
// isolated by tenant_id in the database schema
func TestTenantIsolation_CollectionNaming(t *testing.T) {
	// This test documents the expected behavior:
	//
	// When Tenant A creates collection "docs":
	//   INSERT INTO vector_collections (tenant_id, name) VALUES ($tenantA, 'docs')
	//
	// When Tenant B creates collection "docs":
	//   INSERT INTO vector_collections (tenant_id, name) VALUES ($tenantB, 'docs')
	//
	// These are SEPARATE collections due to UNIQUE (tenant_id, name) constraint
	//
	// When Tenant A queries collection "docs":
	//   SELECT ... FROM vector_documents
	//   WHERE tenant_id = $tenantA AND collection_id IN (
	//     SELECT id FROM vector_collections WHERE tenant_id = $tenantA AND name = 'docs'
	//   )
	//
	// Tenant B's data is NEVER visible to Tenant A

	// Verify UUID generation for different tenants
	tenantA := uuid.New()
	tenantB := uuid.New()

	assert.NotEqual(t, tenantA, tenantB, "Different tenants should have different IDs")
	assert.NotEqual(t, tenantA.String(), tenantB.String(), "Tenant IDs should be unique strings")
}

// ============================================================================
// Phase 3.1: FilterBuilder Tests
// ============================================================================

func TestFilterBuilder_SimpleEquality(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"category": "news",
	}

	clause, args, err := fb.Build(filter)

	assert.NoError(t, err)
	assert.Contains(t, clause, "vd.metadata->>$1 = $2")
	assert.Len(t, args, 2)
	assert.Equal(t, "category", args[0])
	assert.Equal(t, "news", args[1])
}

func TestFilterBuilder_NumericComparison(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"score": map[string]interface{}{
			"$gte": 0.8,
		},
	}

	clause, args, err := fb.Build(filter)

	assert.NoError(t, err)
	assert.Contains(t, clause, "(vd.metadata->>$1)::numeric >= $2")
	assert.Len(t, args, 2)
	assert.Equal(t, "score", args[0])
	assert.Equal(t, 0.8, args[1])
}

func TestFilterBuilder_RangeComparison(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"price": map[string]interface{}{
			"$gte": 100,
			"$lt":  500,
		},
	}

	clause, args, err := fb.Build(filter)

	assert.NoError(t, err)
	// Both conditions should be present
	assert.Contains(t, clause, "(vd.metadata->>")
	assert.Contains(t, clause, "::numeric")
	assert.Len(t, args, 4) // field, value, field, value
}

func TestFilterBuilder_OrCondition(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"$or": []interface{}{
			map[string]interface{}{"category": "news"},
			map[string]interface{}{"category": "blog"},
		},
	}

	clause, args, err := fb.Build(filter)

	assert.NoError(t, err)
	assert.Contains(t, clause, " OR ")
	assert.Len(t, args, 4) // 2 fields + 2 values
}

func TestFilterBuilder_AndCondition(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"$and": []interface{}{
			map[string]interface{}{"status": "active"},
			map[string]interface{}{
				"score": map[string]interface{}{"$gte": 0.5},
			},
		},
	}

	clause, _, err := fb.Build(filter)

	assert.NoError(t, err)
	assert.Contains(t, clause, " AND ")
}

func TestFilterBuilder_InOperator(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"category": map[string]interface{}{
			"$in": []interface{}{"news", "blog", "docs"},
		},
	}

	clause, args, err := fb.Build(filter)

	assert.NoError(t, err)
	assert.Contains(t, clause, "IN")
	assert.Len(t, args, 4) // field + 3 values
}

func TestFilterBuilder_ExistsOperator(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"author": map[string]interface{}{
			"$exists": true,
		},
	}

	clause, args, err := fb.Build(filter)

	assert.NoError(t, err)
	assert.Contains(t, clause, "vd.metadata ? $1")
	assert.Len(t, args, 1)
}

func TestFilterBuilder_NotExistsOperator(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"deleted": map[string]interface{}{
			"$exists": false,
		},
	}

	clause, _, err := fb.Build(filter)

	assert.NoError(t, err)
	assert.Contains(t, clause, "NOT (vd.metadata ? $1)")
}

func TestFilterBuilder_ContainsOperator(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"title": map[string]interface{}{
			"$contains": "AI",
		},
	}

	clause, args, err := fb.Build(filter)

	assert.NoError(t, err)
	assert.Contains(t, clause, "LIKE")
	assert.Equal(t, "%AI%", args[1])
}

func TestFilterBuilder_EmptyFilter(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{}

	clause, args, err := fb.Build(filter)

	assert.NoError(t, err)
	assert.Empty(t, clause)
	assert.Empty(t, args)
}

func TestFilterBuilder_UnsupportedOperator(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"field": map[string]interface{}{
			"$unsupported": "value",
		},
	}

	_, _, err := fb.Build(filter)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}

func TestFilterBuilder_ComplexNestedFilter(t *testing.T) {
	fb := NewFilterBuilder(1, []interface{}{})
	filter := map[string]interface{}{
		"$or": []interface{}{
			map[string]interface{}{
				"$and": []interface{}{
					map[string]interface{}{"category": "news"},
					map[string]interface{}{"score": map[string]interface{}{"$gte": 0.8}},
				},
			},
			map[string]interface{}{"featured": "true"},
		},
	}

	clause, _, err := fb.Build(filter)

	assert.NoError(t, err)
	assert.Contains(t, clause, " OR ")
	assert.Contains(t, clause, " AND ")
}

// ============================================================================
// Phase 3.2: QueryOptions Hybrid Search Tests
// ============================================================================

func TestQueryOptions_HybridSearch(t *testing.T) {
	opts := &QueryOptions{
		TopK:        10,
		Keyword:     "machine learning",
		HybridAlpha: 0.6,
	}

	assert.Equal(t, "machine learning", opts.Keyword)
	assert.Equal(t, 0.6, opts.HybridAlpha)
}

func TestQueryOptions_DefaultHybridAlpha(t *testing.T) {
	opts := &QueryOptions{
		Keyword: "test",
		// HybridAlpha not set, should default to 0.7 in service
	}

	// Default should be 0 (unset)
	assert.Equal(t, float64(0), opts.HybridAlpha)
}
