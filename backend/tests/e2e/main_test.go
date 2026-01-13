package e2e

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup: Create test tenant and user before running tests
	if err := doSetup(); err != nil {
		log.Fatalf("Failed to setup test environment: %v", err)
	}

	// Run all tests
	code := m.Run()

	// Cleanup: Remove test data and tenant/user after all tests
	if err := CleanupTestEnvironment(); err != nil {
		log.Printf("Warning: Failed to cleanup test environment: %v", err)
	}

	os.Exit(code)
}
