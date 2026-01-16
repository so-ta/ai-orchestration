// Package testutil provides common utilities for integration tests.
package testutil

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// LoadEnvFile loads environment variables from a file
func LoadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove quotes if present
		value = strings.Trim(value, `"'`)
		os.Setenv(key, value)
	}
	return scanner.Err()
}

// SkipIfNotIntegration skips the test if INTEGRATION_TEST is not set
func SkipIfNotIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test (set INTEGRATION_TEST=1 to run)")
	}
}

// LoadTestEnv loads .env.test.local if it exists
func LoadTestEnv(t *testing.T) {
	t.Helper()
	// Try to find .env.test.local in various locations
	paths := []string{
		".env.test.local",
		"../.env.test.local",
		"../../.env.test.local",
		"../../../.env.test.local",
		"../../../../.env.test.local",
		filepath.Join(os.Getenv("HOME"), ".env.test.local"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			if err := LoadEnvFile(path); err != nil {
				t.Logf("Warning: failed to load %s: %v", path, err)
			} else {
				t.Logf("Loaded environment from %s", path)
				return
			}
		}
	}
	t.Log("No .env.test.local found, using existing environment variables")
}

// RequireEnvVar checks if an environment variable is set and skips if not
func RequireEnvVar(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping: %s not set", key)
	}
	return value
}
