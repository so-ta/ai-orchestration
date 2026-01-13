package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
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
		fmt.Printf("warning: failed to update document count: %v\n", err)
	}

	return &UpsertResult{
		UpsertedCount: len(ids),
		IDs:           ids,
	}, nil
}

// Query performs similarity search with strict tenant isolation
func (s *VectorServiceImpl) Query(collection string, vector []float32, opts *QueryOptions) (*QueryResult, error) {
	if opts == nil {
		opts = &QueryOptions{}
	}
	if opts.TopK <= 0 {
		opts.TopK = 5
	}

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

	// Add metadata filters if specified
	if opts.Filter != nil && len(opts.Filter) > 0 {
		for key, value := range opts.Filter {
			query += fmt.Sprintf(" AND vd.metadata->>$%d = $%d", argIndex, argIndex+1)
			args = append(args, key, fmt.Sprintf("%v", value))
			argIndex += 2
		}
	}

	query += fmt.Sprintf(" ORDER BY vd.embedding <=> $3::vector LIMIT $%d", argIndex)
	args = append(args, opts.TopK)

	rows, err := s.pool.Query(s.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

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
		if !opts.IncludeContent {
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
