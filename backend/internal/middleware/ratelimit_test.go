package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// mockRedisClient creates a mock Redis client for testing
// Note: For more complex scenarios, consider using miniredis or similar
func newMockRateLimiter(config *RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		redis:  nil, // Will cause errors when accessed
		config: config,
	}
}

// TestTenantRateLimitMiddleware_Disabled tests middleware when rate limiting is disabled
func TestTenantRateLimitMiddleware_Disabled(t *testing.T) {
	config := &RateLimitConfig{
		Enabled: false,
	}
	rl := newMockRateLimiter(config)

	var handlerCalled bool
	handler := rl.TenantRateLimitMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "handler should be called when rate limiting is disabled")
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestTenantRateLimitMiddleware_NoTenantID tests middleware when tenant ID is missing
func TestTenantRateLimitMiddleware_NoTenantID(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:      true,
		TenantLimit:  100,
		TenantWindow: time.Minute,
	}
	rl := newMockRateLimiter(config)

	var handlerCalled bool
	handler := rl.TenantRateLimitMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Request without tenant ID in context
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "handler should be called when tenant ID is missing")
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestTenantRateLimitMiddleware_RedisError tests middleware when Redis returns an error
// This verifies that the request is still allowed and error is logged
func TestTenantRateLimitMiddleware_RedisError(t *testing.T) {
	// Create a real redis client but don't connect it
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:63790", // Non-existent port
	})
	defer client.Close()

	config := &RateLimitConfig{
		Enabled:      true,
		TenantLimit:  100,
		TenantWindow: time.Minute,
	}

	rl := NewRateLimiter(client, config)

	var handlerCalled bool
	handler := rl.TenantRateLimitMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Request with tenant ID in context
	tenantID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, tenantID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// This should not panic but allow the request through
	assert.NotPanics(t, func() {
		handler.ServeHTTP(rec, req)
	})

	// Handler should be called even when Redis fails (fail-open behavior)
	assert.True(t, handlerCalled, "handler should be called even when Redis fails")
	assert.Equal(t, http.StatusOK, rec.Code)
}

// mockRedisClientWithError is a helper type for testing error scenarios
type mockRedisClientWithError struct {
	*redis.Client
	err error
}

// TestRateLimiter_CheckTenant_Error tests CheckTenant with error
func TestRateLimiter_CheckTenant_Error(t *testing.T) {
	config := DefaultRateLimitConfig()

	// Create a real redis client but don't connect it
	// When we call CheckTenant, it will fail
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:63790", // Non-existent port
	})
	defer client.Close()

	rl := NewRateLimiter(client, config)

	ctx := context.Background()
	tenantID := uuid.New()

	// This should return an error because Redis is not available
	result, err := rl.CheckTenant(ctx, tenantID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestRateLimiter_Config tests rate limiter configuration
func TestRateLimiter_Config(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:        true,
		TenantLimit:    500,
		TenantWindow:   2 * time.Minute,
		WorkflowLimit:  50,
		WorkflowWindow: time.Minute,
		WebhookLimit:   30,
		WebhookWindow:  time.Minute,
	}

	rl := NewRateLimiter(nil, config)

	assert.Equal(t, config, rl.GetConfig())

	// Test UpdateConfig
	newConfig := &RateLimitConfig{
		Enabled:      false,
		TenantLimit:  1000,
		TenantWindow: 5 * time.Minute,
	}
	rl.UpdateConfig(newConfig)
	assert.Equal(t, newConfig, rl.GetConfig())
}

// TestDefaultRateLimitConfig tests default configuration values
func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 1000, config.TenantLimit)
	assert.Equal(t, time.Minute, config.TenantWindow)
	assert.Equal(t, 100, config.WorkflowLimit)
	assert.Equal(t, time.Minute, config.WorkflowWindow)
	assert.Equal(t, 60, config.WebhookLimit)
	assert.Equal(t, time.Minute, config.WebhookWindow)
}

// TestRateLimitResult tests RateLimitResult structure
func TestRateLimitResult(t *testing.T) {
	resetAt := time.Now().Add(time.Minute)
	result := &RateLimitResult{
		Allowed:   true,
		Remaining: 99,
		ResetAt:   resetAt,
		Limit:     100,
	}

	assert.True(t, result.Allowed)
	assert.Equal(t, 99, result.Remaining)
	assert.Equal(t, 100, result.Limit)
	assert.Equal(t, resetAt, result.ResetAt)
}

// mockErrorHandler is a helper to capture error handling behavior
func TestTenantRateLimitMiddleware_ErrorHandling_Integration(t *testing.T) {
	// Skip if Redis is not available (integration test)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:63790", // Non-existent port
	})
	defer client.Close()

	// Verify we can't connect
	err := client.Ping(context.Background()).Err()
	if err == nil {
		t.Skip("Skipping test: Redis connection unexpectedly succeeded")
	}

	config := &RateLimitConfig{
		Enabled:      true,
		TenantLimit:  100,
		TenantWindow: time.Minute,
	}
	rl := NewRateLimiter(client, config)

	var handlerCalled bool
	handler := rl.TenantRateLimitMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	tenantID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, tenantID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Request should still succeed (fail-open)
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}
