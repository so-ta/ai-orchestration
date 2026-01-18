package e2e

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	_ "github.com/lib/pq"
)

var (
	setupOnce sync.Once
	setupErr  error
	db        *sql.DB

	// Test tenant and user IDs
	testTenantID = "00000000-0000-0000-0000-000000000099"
	testUserID   = "00000000-0000-0000-0000-000000000098"
)

// SetupTestEnvironment ensures a test tenant and user exist in the database.
// This function is idempotent and can be called multiple times safely.
func SetupTestEnvironment(t *testing.T) {
	setupOnce.Do(func() {
		setupErr = doSetup()
	})
	if setupErr != nil {
		t.Fatalf("Failed to setup test environment: %v", setupErr)
	}
}

func doSetup() error {
	dbURL := getEnv("DATABASE_URL", "postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable")

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create test tenant if not exists
	_, err = db.Exec(`
		INSERT INTO tenants (id, name, slug, status, plan, feature_flags, limits)
		VALUES ($1, 'E2E Test Tenant', 'e2e-test-tenant', 'active', 'free',
			'{"api_access": true, "audit_logs": true, "sso_enabled": false, "custom_blocks": true, "copilot_enabled": true}',
			'{"max_users": 50, "max_workflows": 100, "max_runs_per_day": 1000}'
		)
		ON CONFLICT (id) DO NOTHING
	`, testTenantID)
	if err != nil {
		return fmt.Errorf("failed to create test tenant: %w", err)
	}

	// Create test user if not exists
	_, err = db.Exec(`
		INSERT INTO users (id, tenant_id, email, name, role)
		VALUES ($1, $2, 'e2e-test@example.com', 'E2E Test User', 'tenant_admin')
		ON CONFLICT (id) DO NOTHING
	`, testUserID, testTenantID)
	if err != nil {
		return fmt.Errorf("failed to create test user: %w", err)
	}

	return nil
}

// CleanupTestData removes test data created during tests.
// Call this in TestMain or at the end of test suites.
func CleanupTestData() error {
	if db == nil {
		return nil
	}

	// Delete in order respecting foreign key constraints
	// Note: Tables renamed from workflow_* to project_*
	tables := []string{
		"step_runs",
		"runs",
		"edges",
		"steps",
		"block_groups",
		"schedules",
		"project_versions",
		"projects",
		"secrets",
		"credentials",
		"audit_logs",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1", table), testTenantID)
		if err != nil {
			return fmt.Errorf("failed to clean %s: %w", table, err)
		}
	}

	return nil
}

// CleanupTestEnvironment removes the test tenant and user.
// This should be called at the very end of all tests.
func CleanupTestEnvironment() error {
	if db == nil {
		return nil
	}

	// Clean up all test data first
	if err := CleanupTestData(); err != nil {
		return err
	}

	// Delete test user
	_, err := db.Exec("DELETE FROM users WHERE id = $1", testUserID)
	if err != nil {
		return fmt.Errorf("failed to delete test user: %w", err)
	}

	// Delete test tenant
	_, err = db.Exec("DELETE FROM tenants WHERE id = $1", testTenantID)
	if err != nil {
		return fmt.Errorf("failed to delete test tenant: %w", err)
	}

	return db.Close()
}

// GetTestTenantID returns the test tenant ID for use in tests
func GetTestTenantID() string {
	return testTenantID
}

// GetTestUserID returns the test user ID for use in tests
func GetTestUserID() string {
	return testUserID
}
