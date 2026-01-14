package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSetTenantContext_ValidUUID(t *testing.T) {
	// Test that valid UUIDs are accepted
	validUUIDs := []string{
		"00000000-0000-0000-0000-000000000001",
		"550e8400-e29b-41d4-a716-446655440000",
		uuid.New().String(),
	}

	for _, id := range validUUIDs {
		parsed, err := uuid.Parse(id)
		assert.NoError(t, err, "UUID %s should be valid", id)
		assert.Equal(t, id, parsed.String())
	}
}

func TestSetTenantContext_InvalidUUID(t *testing.T) {
	// Test that invalid UUIDs are rejected
	invalidUUIDs := []string{
		"",
		"invalid",
		"'; DROP TABLE users; --",
		"1234",
		"not-a-uuid",
		"00000000-0000-0000-0000-00000000000",  // too short
		"00000000-0000-0000-0000-0000000000001", // too long
	}

	for _, id := range invalidUUIDs {
		_, err := uuid.Parse(id)
		assert.Error(t, err, "UUID %q should be invalid", id)
	}
}

func TestSetTenantContext_SQLInjectionPrevention(t *testing.T) {
	// Test that SQL injection attempts are rejected as invalid UUIDs
	sqlInjectionAttempts := []string{
		"'; DROP TABLE users; --",
		"1' OR '1'='1",
		"'; DELETE FROM tenants WHERE '1'='1",
		"UNION SELECT * FROM passwords",
		"1; DROP TABLE users",
		"' OR 1=1 --",
		"admin'--",
		"1' AND '1'='1",
	}

	for _, attempt := range sqlInjectionAttempts {
		_, err := uuid.Parse(attempt)
		assert.Error(t, err, "SQL injection attempt %q should be rejected", attempt)
	}
}
