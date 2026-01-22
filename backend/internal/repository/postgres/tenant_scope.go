package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantScope provides tenant-scoped database access with built-in validation
// This ensures all database operations are properly scoped to a specific tenant
type TenantScope struct {
	pool     *pgxpool.Pool
	tenantID uuid.UUID
}

// NewTenantScope creates a new TenantScope with the given pool and tenant ID
// Returns an error if tenantID is nil (uuid.Nil)
func NewTenantScope(pool *pgxpool.Pool, tenantID uuid.UUID) (*TenantScope, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return &TenantScope{
		pool:     pool,
		tenantID: tenantID,
	}, nil
}

// TenantID returns the tenant ID for this scope
func (ts *TenantScope) TenantID() uuid.UUID {
	return ts.tenantID
}

// Pool returns the underlying connection pool
func (ts *TenantScope) Pool() *pgxpool.Pool {
	return ts.pool
}

// QueryRow executes a query that returns a single row with automatic tenant filtering
// The query MUST contain a $TENANT_ID placeholder that will be replaced
func (ts *TenantScope) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	// Inject tenant ID as first argument
	newArgs := make([]any, 0, len(args)+1)
	newArgs = append(newArgs, ts.tenantID)
	newArgs = append(newArgs, args...)

	return ts.pool.QueryRow(ctx, sql, newArgs...)
}

// Query executes a query that returns multiple rows with automatic tenant filtering
func (ts *TenantScope) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	// Inject tenant ID as first argument
	newArgs := make([]any, 0, len(args)+1)
	newArgs = append(newArgs, ts.tenantID)
	newArgs = append(newArgs, args...)

	return ts.pool.Query(ctx, sql, newArgs...)
}

// Exec executes a command with automatic tenant filtering
func (ts *TenantScope) Exec(ctx context.Context, sql string, args ...any) error {
	// Inject tenant ID as first argument
	newArgs := make([]any, 0, len(args)+1)
	newArgs = append(newArgs, ts.tenantID)
	newArgs = append(newArgs, args...)

	_, err := ts.pool.Exec(ctx, sql, newArgs...)
	return err
}

// TenantFilter is a helper to build WHERE clauses with tenant filtering
type TenantFilter struct {
	conditions []string
	args       []any
	argIndex   int
}

// NewTenantFilter creates a new filter starting with tenant_id condition
func NewTenantFilter(tenantID uuid.UUID) *TenantFilter {
	return &TenantFilter{
		conditions: []string{"tenant_id = $1"},
		args:       []any{tenantID},
		argIndex:   2, // Next argument will be $2
	}
}

// And adds an AND condition to the filter
func (tf *TenantFilter) And(condition string, args ...any) *TenantFilter {
	// Replace $N placeholders with correct argument numbers
	adjustedCondition := tf.adjustPlaceholders(condition, len(args))
	tf.conditions = append(tf.conditions, adjustedCondition)
	tf.args = append(tf.args, args...)
	return tf
}

// adjustPlaceholders adjusts $1, $2, etc. to the correct indices
func (tf *TenantFilter) adjustPlaceholders(condition string, argCount int) string {
	result := condition
	for i := argCount; i >= 1; i-- {
		old := fmt.Sprintf("$%d", i)
		new := fmt.Sprintf("$%d", tf.argIndex+i-1)
		result = strings.ReplaceAll(result, old, new)
	}
	tf.argIndex += argCount
	return result
}

// NotDeleted adds deleted_at IS NULL condition (commonly required)
func (tf *TenantFilter) NotDeleted() *TenantFilter {
	tf.conditions = append(tf.conditions, "deleted_at IS NULL")
	return tf
}

// ByID adds id = $N condition
func (tf *TenantFilter) ByID(id uuid.UUID) *TenantFilter {
	tf.conditions = append(tf.conditions, fmt.Sprintf("id = $%d", tf.argIndex))
	tf.args = append(tf.args, id)
	tf.argIndex++
	return tf
}

// Where returns the complete WHERE clause
func (tf *TenantFilter) Where() string {
	return "WHERE " + strings.Join(tf.conditions, " AND ")
}

// Args returns all arguments for the query
func (tf *TenantFilter) Args() []any {
	return tf.args
}

// ValidateTenantAccess checks if a resource belongs to the given tenant
// This is used for cross-tenant access validation
func ValidateTenantAccess(ctx context.Context, pool *pgxpool.Pool, table string, resourceID, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("tenant_id is required")
	}

	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL)`, table)

	var exists bool
	err := pool.QueryRow(ctx, query, resourceID, tenantID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("validate tenant access: %w", err)
	}

	if !exists {
		return fmt.Errorf("resource not found or access denied")
	}

	return nil
}

// EnsureTenantMatch validates that the provided tenant ID matches the expected one
// This is a simple helper for cases where you already have both IDs
func EnsureTenantMatch(expected, actual uuid.UUID) error {
	if expected != actual {
		return fmt.Errorf("tenant_id mismatch: access denied")
	}
	return nil
}

// TenantScopedQuery is a helper struct for building tenant-scoped queries
type TenantScopedQuery struct {
	baseSQL  string
	tenantID uuid.UUID
	args     []any
}

// NewTenantScopedQuery creates a new query builder
// The baseSQL should use $1 for tenant_id and start other placeholders from $2
func NewTenantScopedQuery(baseSQL string, tenantID uuid.UUID) *TenantScopedQuery {
	return &TenantScopedQuery{
		baseSQL:  baseSQL,
		tenantID: tenantID,
		args:     []any{tenantID},
	}
}

// WithArgs adds arguments to the query (starting from $2)
func (q *TenantScopedQuery) WithArgs(args ...any) *TenantScopedQuery {
	q.args = append(q.args, args...)
	return q
}

// SQL returns the SQL string
func (q *TenantScopedQuery) SQL() string {
	return q.baseSQL
}

// Args returns all arguments including tenant_id as first argument
func (q *TenantScopedQuery) Args() []any {
	return q.args
}
