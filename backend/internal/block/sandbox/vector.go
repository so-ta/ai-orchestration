package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VectorService provides vector DB operations with strict tenant isolation
type VectorService interface {
	// Upsert adds or updates documents in a collection
	// ⚠️ tenant_id is automatically applied, cannot be overridden
	Upsert(collection string, documents []VectorDocument, opts *UpsertOptions) (*UpsertResult, error)

	// Query performs similarity search
	// ⚠️ tenant_id filter is automatically applied
	Query(collection string, vector []float32, opts *QueryOptions) (*QueryResult, error)

	// Delete removes documents from a collection
	// ⚠️ Only documents belonging to the tenant can be deleted
	Delete(collection string, ids []string) (*DeleteResult, error)

	// ListCollections returns all collections for the tenant
	// ⚠️ Only returns collections belonging to the tenant
	ListCollections() ([]CollectionInfo, error)
}

// VectorDocument represents a document with optional embedding
type VectorDocument struct {
	ID       string                 `json:"id,omitempty"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Vector   []float32              `json:"vector,omitempty"`
}

// UpsertOptions contains options for upsert operation
type UpsertOptions struct {
	EmbeddingProvider string `json:"embedding_provider,omitempty"`
	EmbeddingModel    string `json:"embedding_model,omitempty"`
}

// UpsertResult contains the result of an upsert operation
type UpsertResult struct {
	UpsertedCount int      `json:"upserted_count"`
	IDs           []string `json:"ids"`
}

// QueryOptions contains options for query operation
type QueryOptions struct {
	TopK           int                    `json:"top_k,omitempty"`
	Threshold      float64                `json:"threshold,omitempty"`
	Filter         map[string]interface{} `json:"filter,omitempty"`
	IncludeContent bool                   `json:"include_content,omitempty"`
	// Hybrid search options (Phase 3.2)
	Keyword       string  `json:"keyword,omitempty"`        // Keyword for hybrid search
	HybridAlpha   float64 `json:"hybrid_alpha,omitempty"`   // Weight for vector score (0-1), default 0.7
}

// QueryResult contains the result of a query operation
type QueryResult struct {
	Matches []QueryMatch `json:"matches"`
}

// QueryMatch represents a single match from similarity search
type QueryMatch struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Content  string                 `json:"content,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DeleteResult contains the result of a delete operation
type DeleteResult struct {
	DeletedCount int `json:"deleted_count"`
}

// CollectionInfo contains information about a collection
type CollectionInfo struct {
	Name          string `json:"name"`
	DocumentCount int    `json:"document_count"`
	Dimension     int    `json:"dimension"`
	CreatedAt     string `json:"created_at"`
}

// VectorServiceImpl implements VectorService with PGVector backend
type VectorServiceImpl struct {
	pool             *pgxpool.Pool
	embeddingService EmbeddingService
	tenantID         uuid.UUID // ⚠️ Set from ExecutionContext, cannot be changed by user
	ctx              context.Context
}

// NewVectorService creates a new VectorService with strict tenant isolation
// ⚠️ tenantID cannot be changed after creation
func NewVectorService(ctx context.Context, tenantID uuid.UUID, pool *pgxpool.Pool, embeddingService EmbeddingService) *VectorServiceImpl {
	return &VectorServiceImpl{
		ctx:              ctx,
		tenantID:         tenantID,
		pool:             pool,
		embeddingService: embeddingService,
	}
}

// Upsert adds or updates documents in a collection
func (s *VectorServiceImpl) Upsert(collection string, documents []VectorDocument, opts *UpsertOptions) (*UpsertResult, error) {
	if len(documents) == 0 {
		return &UpsertResult{UpsertedCount: 0, IDs: []string{}}, nil
	}

	// Get or create collection (with tenant_id)
	collectionID, err := s.getOrCreateCollection(collection, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create collection: %w", err)
	}

	// Generate embeddings for documents without vectors
	if err := s.generateMissingEmbeddings(documents, opts); err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Insert documents
	ids := make([]string, 0, len(documents))
	for _, doc := range documents {
		docID := doc.ID
		if docID == "" {
			docID = uuid.New().String()
		}

		metadataJSON, err := json.Marshal(doc.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		// Convert vector to pgvector format
		vectorStr := s.vectorToString(doc.Vector)

		// ⚠️ tenant_id is always set from s.tenantID
		query := `
			INSERT INTO vector_documents (id, tenant_id, collection_id, content, metadata, embedding, source_type)
			VALUES ($1, $2, $3, $4, $5, $6::vector, $7)
			ON CONFLICT (id) DO UPDATE SET
				content = EXCLUDED.content,
				metadata = EXCLUDED.metadata,
				embedding = EXCLUDED.embedding
		`

		_, err = s.pool.Exec(s.ctx, query,
			docID,
			s.tenantID, // ⚠️ Forced tenant isolation
			collectionID,
			doc.Content,
			metadataJSON,
			vectorStr,
			"api",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert document: %w", err)
		}

		ids = append(ids, docID)
	}

	// Update document count
	if err := s.updateDocumentCount(collectionID); err != nil {
		// Log but don't fail
		slog.Warn("failed to update document count", "error", err, "collection_id", collectionID)
	}

	return &UpsertResult{
		UpsertedCount: len(ids),
		IDs:           ids,
	}, nil
}

// Query performs similarity search with strict tenant isolation
// Supports:
// - Vector similarity search (cosine)
// - Advanced metadata filters ($eq, $ne, $gt, $gte, $lt, $lte, $in, $nin, $and, $or, $exists, $contains)
// - Hybrid search (keyword + vector with RRF)
func (s *VectorServiceImpl) Query(collection string, vector []float32, opts *QueryOptions) (*QueryResult, error) {
	if opts == nil {
		opts = &QueryOptions{}
	}
	if opts.TopK <= 0 {
		opts.TopK = 5
	}

	// Use hybrid search if keyword is provided
	if opts.Keyword != "" {
		return s.queryHybrid(collection, vector, opts)
	}

	return s.queryVector(collection, vector, opts)
}

// queryVector performs pure vector similarity search
func (s *VectorServiceImpl) queryVector(collection string, vector []float32, opts *QueryOptions) (*QueryResult, error) {
	vectorStr := s.vectorToString(vector)

	// Build query with mandatory tenant filter
	// ⚠️ Both collection and documents are filtered by tenant_id
	query := `
		SELECT
			vd.id,
			vd.content,
			vd.metadata,
			1 - (vd.embedding <=> $3::vector) as score
		FROM vector_documents vd
		JOIN vector_collections vc ON vd.collection_id = vc.id
		WHERE vc.tenant_id = $1
		  AND vc.name = $2
		  AND vd.tenant_id = $1
	`

	args := []interface{}{s.tenantID, collection, vectorStr}
	argIndex := 4

	// Add threshold filter if specified
	if opts.Threshold > 0 {
		query += fmt.Sprintf(" AND 1 - (vd.embedding <=> $3::vector) >= $%d", argIndex)
		args = append(args, opts.Threshold)
		argIndex++
	}

	// Add metadata filters using FilterBuilder (Phase 3.1)
	if opts.Filter != nil && len(opts.Filter) > 0 {
		fb := NewFilterBuilder(argIndex, args)
		filterClause, newArgs, err := fb.Build(opts.Filter)
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %w", err)
		}
		if filterClause != "" {
			query += " AND " + filterClause
			args = newArgs
			argIndex = fb.argIndex
		}
	}

	query += fmt.Sprintf(" ORDER BY vd.embedding <=> $3::vector LIMIT $%d", argIndex)
	args = append(args, opts.TopK)

	rows, err := s.pool.Query(s.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return s.scanQueryResults(rows, opts.IncludeContent)
}

// queryHybrid performs hybrid search (vector + keyword) using RRF
func (s *VectorServiceImpl) queryHybrid(collection string, vector []float32, opts *QueryOptions) (*QueryResult, error) {
	vectorStr := s.vectorToString(vector)

	// Default alpha: 0.7 (70% vector, 30% keyword)
	alpha := opts.HybridAlpha
	if alpha <= 0 || alpha > 1 {
		alpha = 0.7
	}

	// RRF (Reciprocal Rank Fusion) query
	// Combines vector similarity and full-text search scores
	query := `
		WITH vector_results AS (
			SELECT
				vd.id,
				vd.content,
				vd.metadata,
				ROW_NUMBER() OVER (ORDER BY vd.embedding <=> $3::vector) as v_rank,
				1 - (vd.embedding <=> $3::vector) as v_score
			FROM vector_documents vd
			JOIN vector_collections vc ON vd.collection_id = vc.id
			WHERE vc.tenant_id = $1
			  AND vc.name = $2
			  AND vd.tenant_id = $1
			LIMIT 50
		),
		keyword_results AS (
			SELECT
				vd.id,
				ROW_NUMBER() OVER (ORDER BY ts_rank(to_tsvector('simple', vd.content), plainto_tsquery('simple', $4)) DESC) as k_rank,
				ts_rank(to_tsvector('simple', vd.content), plainto_tsquery('simple', $4)) as k_score
			FROM vector_documents vd
			JOIN vector_collections vc ON vd.collection_id = vc.id
			WHERE vc.tenant_id = $1
			  AND vc.name = $2
			  AND vd.tenant_id = $1
			  AND to_tsvector('simple', vd.content) @@ plainto_tsquery('simple', $4)
			LIMIT 50
		)
		SELECT
			COALESCE(v.id, k.id) as id,
			v.content,
			v.metadata,
			(
				$5::float * (1.0 / (60 + COALESCE(v.v_rank, 100))) +
				(1 - $5::float) * (1.0 / (60 + COALESCE(k.k_rank, 100)))
			) as rrf_score
		FROM vector_results v
		FULL OUTER JOIN keyword_results k ON v.id = k.id
		WHERE v.id IS NOT NULL OR k.id IS NOT NULL
		ORDER BY rrf_score DESC
		LIMIT $6
	`

	args := []interface{}{s.tenantID, collection, vectorStr, opts.Keyword, alpha, opts.TopK}

	rows, err := s.pool.Query(s.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("hybrid query failed: %w", err)
	}
	defer rows.Close()

	return s.scanQueryResults(rows, opts.IncludeContent)
}

// scanQueryResults scans rows into QueryMatch slice
func (s *VectorServiceImpl) scanQueryResults(rows pgx.Rows, includeContent bool) (*QueryResult, error) {
	matches := []QueryMatch{}
	for rows.Next() {
		var match QueryMatch
		var metadataJSON []byte

		if err := rows.Scan(&match.ID, &match.Content, &metadataJSON, &match.Score); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &match.Metadata); err != nil {
				match.Metadata = map[string]interface{}{}
			}
		}

		// Optionally exclude content
		if !includeContent {
			match.Content = ""
		}

		matches = append(matches, match)
	}

	return &QueryResult{Matches: matches}, nil
}

// Delete removes documents from a collection
func (s *VectorServiceImpl) Delete(collection string, ids []string) (*DeleteResult, error) {
	if len(ids) == 0 {
		return &DeleteResult{DeletedCount: 0}, nil
	}

	// Build placeholder list for IDs
	placeholders := make([]string, len(ids))
	args := []interface{}{s.tenantID, collection}
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args = append(args, id)
	}

	// ⚠️ tenant_id filter ensures only tenant's documents can be deleted
	query := fmt.Sprintf(`
		DELETE FROM vector_documents vd
		USING vector_collections vc
		WHERE vd.collection_id = vc.id
		  AND vc.tenant_id = $1
		  AND vc.name = $2
		  AND vd.tenant_id = $1
		  AND vd.id IN (%s)
	`, strings.Join(placeholders, ","))

	result, err := s.pool.Exec(s.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("delete failed: %w", err)
	}

	return &DeleteResult{
		DeletedCount: int(result.RowsAffected()),
	}, nil
}

// ListCollections returns all collections for the tenant
func (s *VectorServiceImpl) ListCollections() ([]CollectionInfo, error) {
	// ⚠️ Only returns collections belonging to the tenant
	query := `
		SELECT name, document_count, dimension, created_at
		FROM vector_collections
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(s.ctx, query, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer rows.Close()

	collections := []CollectionInfo{}
	for rows.Next() {
		var info CollectionInfo
		var createdAt interface{}

		if err := rows.Scan(&info.Name, &info.DocumentCount, &info.Dimension, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if t, ok := createdAt.(string); ok {
			info.CreatedAt = t
		} else {
			info.CreatedAt = fmt.Sprintf("%v", createdAt)
		}

		collections = append(collections, info)
	}

	return collections, nil
}

// getOrCreateCollection gets or creates a collection with tenant isolation
func (s *VectorServiceImpl) getOrCreateCollection(name string, opts *UpsertOptions) (uuid.UUID, error) {
	provider := "openai"
	model := "text-embedding-3-small"
	dimension := 1536

	if opts != nil {
		if opts.EmbeddingProvider != "" {
			provider = opts.EmbeddingProvider
		}
		if opts.EmbeddingModel != "" {
			model = opts.EmbeddingModel
			// Set dimension based on model
			if strings.Contains(model, "large") {
				dimension = 3072
			}
		}
	}

	// Try to get existing collection
	var collectionID uuid.UUID
	err := s.pool.QueryRow(s.ctx,
		"SELECT id FROM vector_collections WHERE tenant_id = $1 AND name = $2",
		s.tenantID, name,
	).Scan(&collectionID)

	if err == nil {
		return collectionID, nil
	}

	if err != pgx.ErrNoRows {
		return uuid.Nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// Create new collection
	collectionID = uuid.New()
	_, err = s.pool.Exec(s.ctx,
		`INSERT INTO vector_collections (id, tenant_id, name, embedding_provider, embedding_model, dimension)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		collectionID, s.tenantID, name, provider, model, dimension,
	)
	if err != nil {
		// Check if another request created it
		err2 := s.pool.QueryRow(s.ctx,
			"SELECT id FROM vector_collections WHERE tenant_id = $1 AND name = $2",
			s.tenantID, name,
		).Scan(&collectionID)
		if err2 == nil {
			return collectionID, nil
		}
		return uuid.Nil, fmt.Errorf("failed to create collection: %w", err)
	}

	return collectionID, nil
}

// generateMissingEmbeddings generates embeddings for documents without vectors
func (s *VectorServiceImpl) generateMissingEmbeddings(documents []VectorDocument, opts *UpsertOptions) error {
	// Find documents without embeddings
	var textsToEmbed []string
	var indices []int

	for i, doc := range documents {
		if len(doc.Vector) == 0 {
			textsToEmbed = append(textsToEmbed, doc.Content)
			indices = append(indices, i)
		}
	}

	if len(textsToEmbed) == 0 {
		return nil
	}

	// Generate embeddings
	provider := "openai"
	model := "text-embedding-3-small"
	if opts != nil {
		if opts.EmbeddingProvider != "" {
			provider = opts.EmbeddingProvider
		}
		if opts.EmbeddingModel != "" {
			model = opts.EmbeddingModel
		}
	}

	result, err := s.embeddingService.Embed(provider, model, textsToEmbed)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	// Assign vectors to documents
	for i, idx := range indices {
		documents[idx].Vector = result.Vectors[i]
	}

	return nil
}

// updateDocumentCount updates the document count for a collection
func (s *VectorServiceImpl) updateDocumentCount(collectionID uuid.UUID) error {
	_, err := s.pool.Exec(s.ctx, `
		UPDATE vector_collections
		SET document_count = (
			SELECT COUNT(*) FROM vector_documents WHERE collection_id = $1
		)
		WHERE id = $1 AND tenant_id = $2
	`, collectionID, s.tenantID)
	return err
}

// vectorToString converts a float32 slice to pgvector string format
func (s *VectorServiceImpl) vectorToString(vector []float32) string {
	if len(vector) == 0 {
		return ""
	}

	strs := make([]string, len(vector))
	for i, v := range vector {
		strs[i] = fmt.Sprintf("%f", v)
	}
	return "[" + strings.Join(strs, ",") + "]"
}

// ============================================================================
// Phase 3.1: Advanced Metadata Filter Support
// ============================================================================

// FilterBuilder builds SQL WHERE clauses from filter expressions
// Supports: $eq, $ne, $gt, $gte, $lt, $lte, $in, $nin, $and, $or, $exists
type FilterBuilder struct {
	args     []interface{}
	argIndex int
}

// NewFilterBuilder creates a new FilterBuilder
func NewFilterBuilder(startIndex int, initialArgs []interface{}) *FilterBuilder {
	return &FilterBuilder{
		args:     initialArgs,
		argIndex: startIndex,
	}
}

// Build converts a filter map to SQL WHERE clause
func (fb *FilterBuilder) Build(filter map[string]interface{}) (string, []interface{}, error) {
	if len(filter) == 0 {
		return "", fb.args, nil
	}

	clause, err := fb.buildCondition(filter)
	if err != nil {
		return "", nil, err
	}

	return clause, fb.args, nil
}

// buildCondition recursively builds conditions
func (fb *FilterBuilder) buildCondition(filter map[string]interface{}) (string, error) {
	var conditions []string

	for key, value := range filter {
		switch key {
		case "$and":
			clause, err := fb.buildLogicalOp(value, "AND")
			if err != nil {
				return "", err
			}
			conditions = append(conditions, clause)

		case "$or":
			clause, err := fb.buildLogicalOp(value, "OR")
			if err != nil {
				return "", err
			}
			conditions = append(conditions, clause)

		default:
			// Field-level condition
			clause, err := fb.buildFieldCondition(key, value)
			if err != nil {
				return "", err
			}
			conditions = append(conditions, clause)
		}
	}

	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " AND ") + ")", nil
}

// buildLogicalOp builds $and / $or conditions
func (fb *FilterBuilder) buildLogicalOp(value interface{}, op string) (string, error) {
	arr, ok := value.([]interface{})
	if !ok {
		return "", fmt.Errorf("$%s requires an array", strings.ToLower(op))
	}

	var clauses []string
	for _, item := range arr {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("$%s array items must be objects", strings.ToLower(op))
		}
		clause, err := fb.buildCondition(itemMap)
		if err != nil {
			return "", err
		}
		clauses = append(clauses, clause)
	}

	if len(clauses) == 0 {
		return "TRUE", nil
	}
	return "(" + strings.Join(clauses, " "+op+" ") + ")", nil
}

// buildFieldCondition builds a condition for a single field
func (fb *FilterBuilder) buildFieldCondition(field string, value interface{}) (string, error) {
	// Check if value is an operator object
	if opMap, ok := value.(map[string]interface{}); ok {
		return fb.buildOperatorCondition(field, opMap)
	}

	// Simple equality
	fb.args = append(fb.args, field, fmt.Sprintf("%v", value))
	clause := fmt.Sprintf("vd.metadata->>$%d = $%d", fb.argIndex, fb.argIndex+1)
	fb.argIndex += 2
	return clause, nil
}

// buildOperatorCondition builds conditions with operators
func (fb *FilterBuilder) buildOperatorCondition(field string, ops map[string]interface{}) (string, error) {
	var conditions []string

	for op, val := range ops {
		var clause string

		switch op {
		case "$eq":
			fb.args = append(fb.args, field, fmt.Sprintf("%v", val))
			clause = fmt.Sprintf("vd.metadata->>$%d = $%d", fb.argIndex, fb.argIndex+1)
			fb.argIndex += 2

		case "$ne":
			fb.args = append(fb.args, field, fmt.Sprintf("%v", val))
			clause = fmt.Sprintf("vd.metadata->>$%d != $%d", fb.argIndex, fb.argIndex+1)
			fb.argIndex += 2

		case "$gt":
			fb.args = append(fb.args, field, val)
			clause = fmt.Sprintf("(vd.metadata->>$%d)::numeric > $%d", fb.argIndex, fb.argIndex+1)
			fb.argIndex += 2

		case "$gte":
			fb.args = append(fb.args, field, val)
			clause = fmt.Sprintf("(vd.metadata->>$%d)::numeric >= $%d", fb.argIndex, fb.argIndex+1)
			fb.argIndex += 2

		case "$lt":
			fb.args = append(fb.args, field, val)
			clause = fmt.Sprintf("(vd.metadata->>$%d)::numeric < $%d", fb.argIndex, fb.argIndex+1)
			fb.argIndex += 2

		case "$lte":
			fb.args = append(fb.args, field, val)
			clause = fmt.Sprintf("(vd.metadata->>$%d)::numeric <= $%d", fb.argIndex, fb.argIndex+1)
			fb.argIndex += 2

		case "$in":
			arr, ok := val.([]interface{})
			if !ok {
				return "", fmt.Errorf("$in requires an array")
			}
			placeholders := make([]string, len(arr))
			fb.args = append(fb.args, field)
			fieldIdx := fb.argIndex
			fb.argIndex++
			for i, item := range arr {
				fb.args = append(fb.args, fmt.Sprintf("%v", item))
				placeholders[i] = fmt.Sprintf("$%d", fb.argIndex)
				fb.argIndex++
			}
			clause = fmt.Sprintf("vd.metadata->>$%d IN (%s)", fieldIdx, strings.Join(placeholders, ","))

		case "$nin":
			arr, ok := val.([]interface{})
			if !ok {
				return "", fmt.Errorf("$nin requires an array")
			}
			placeholders := make([]string, len(arr))
			fb.args = append(fb.args, field)
			fieldIdx := fb.argIndex
			fb.argIndex++
			for i, item := range arr {
				fb.args = append(fb.args, fmt.Sprintf("%v", item))
				placeholders[i] = fmt.Sprintf("$%d", fb.argIndex)
				fb.argIndex++
			}
			clause = fmt.Sprintf("vd.metadata->>$%d NOT IN (%s)", fieldIdx, strings.Join(placeholders, ","))

		case "$exists":
			exists, ok := val.(bool)
			if !ok {
				return "", fmt.Errorf("$exists requires a boolean")
			}
			fb.args = append(fb.args, field)
			if exists {
				clause = fmt.Sprintf("vd.metadata ? $%d", fb.argIndex)
			} else {
				clause = fmt.Sprintf("NOT (vd.metadata ? $%d)", fb.argIndex)
			}
			fb.argIndex++

		case "$contains":
			// String contains (LIKE)
			fb.args = append(fb.args, field, fmt.Sprintf("%%%v%%", val))
			clause = fmt.Sprintf("vd.metadata->>$%d LIKE $%d", fb.argIndex, fb.argIndex+1)
			fb.argIndex += 2

		default:
			return "", fmt.Errorf("unsupported operator: %s", op)
		}

		conditions = append(conditions, clause)
	}

	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, " AND ") + ")", nil
}
