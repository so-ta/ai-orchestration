package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RateLimitScope defines the scope of rate limiting
type RateLimitScope string

const (
	RateLimitScopeTenant   RateLimitScope = "tenant"
	RateLimitScopeWorkflow RateLimitScope = "workflow"
	RateLimitScopeWebhook  RateLimitScope = "webhook"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Enabled enables/disables rate limiting
	Enabled bool

	// Tenant-level limits (requests per window)
	TenantLimit  int
	TenantWindow time.Duration

	// Workflow-level limits (requests per window per workflow)
	WorkflowLimit  int
	WorkflowWindow time.Duration

	// Webhook-level limits (requests per window per webhook key)
	WebhookLimit  int
	WebhookWindow time.Duration
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:        true,
		TenantLimit:    1000, // 1000 requests per minute per tenant
		TenantWindow:   time.Minute,
		WorkflowLimit:  100, // 100 requests per minute per workflow
		WorkflowWindow: time.Minute,
		WebhookLimit:   60, // 60 requests per minute per webhook key
		WebhookWindow:  time.Minute,
	}
}

// RateLimiter handles rate limiting using Redis
type RateLimiter struct {
	redis  *redis.Client
	config *RateLimitConfig
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client, config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	return &RateLimiter{
		redis:  redisClient,
		config: config,
	}
}

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
	Limit     int
}

// checkLimit performs the rate limit check using Redis sliding window
func (rl *RateLimiter) checkLimit(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	now := time.Now()
	windowStart := now.Add(-window)
	resetAt := now.Add(window)

	// Use a Lua script for atomic operations
	script := redis.NewScript(`
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_ms = tonumber(ARGV[4])

		-- Remove old entries outside the window
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

		-- Count current entries
		local count = redis.call('ZCARD', key)

		if count < limit then
			-- Add new entry
			redis.call('ZADD', key, now, now .. '-' .. math.random())
			redis.call('PEXPIRE', key, window_ms)
			return {1, limit - count - 1}
		else
			return {0, 0}
		end
	`)

	nowMs := now.UnixMilli()
	windowStartMs := windowStart.UnixMilli()
	windowMs := window.Milliseconds()

	result, err := script.Run(ctx, rl.redis, []string{key}, nowMs, windowStartMs, limit, windowMs).Slice()
	if err != nil {
		return nil, fmt.Errorf("rate limit script error: %w", err)
	}

	allowed := result[0].(int64) == 1
	remaining := int(result[1].(int64))

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   resetAt,
		Limit:     limit,
	}, nil
}

// CheckTenant checks the tenant-level rate limit
func (rl *RateLimiter) CheckTenant(ctx context.Context, tenantID uuid.UUID) (*RateLimitResult, error) {
	key := fmt.Sprintf("ratelimit:tenant:%s", tenantID.String())
	return rl.checkLimit(ctx, key, rl.config.TenantLimit, rl.config.TenantWindow)
}

// CheckWorkflow checks the workflow-level rate limit
func (rl *RateLimiter) CheckWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID) (*RateLimitResult, error) {
	key := fmt.Sprintf("ratelimit:workflow:%s:%s", tenantID.String(), workflowID.String())
	return rl.checkLimit(ctx, key, rl.config.WorkflowLimit, rl.config.WorkflowWindow)
}

// CheckWebhook checks the webhook-level rate limit
func (rl *RateLimiter) CheckWebhook(ctx context.Context, webhookKey string) (*RateLimitResult, error) {
	key := fmt.Sprintf("ratelimit:webhook:%s", webhookKey)
	return rl.checkLimit(ctx, key, rl.config.WebhookLimit, rl.config.WebhookWindow)
}

// setRateLimitHeaders sets rate limit headers on the response
func setRateLimitHeaders(w http.ResponseWriter, result *RateLimitResult, scope RateLimitScope) {
	prefix := "X-RateLimit"
	if scope != "" {
		prefix = fmt.Sprintf("X-RateLimit-%s", scope)
	}
	w.Header().Set(fmt.Sprintf("%s-Limit", prefix), strconv.Itoa(result.Limit))
	w.Header().Set(fmt.Sprintf("%s-Remaining", prefix), strconv.Itoa(result.Remaining))
	w.Header().Set(fmt.Sprintf("%s-Reset", prefix), strconv.FormatInt(result.ResetAt.Unix(), 10))
}

// writeRateLimitError writes a rate limit exceeded error response
func writeRateLimitError(w http.ResponseWriter, result *RateLimitResult, scope RateLimitScope) {
	setRateLimitHeaders(w, result, scope)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.FormatInt(int64(time.Until(result.ResetAt).Seconds()), 10))
	w.WriteHeader(http.StatusTooManyRequests)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":       "RATE_LIMIT_EXCEEDED",
			"message":    fmt.Sprintf("Rate limit exceeded for %s scope", scope),
			"retry_at":   result.ResetAt.Format(time.RFC3339),
			"limit":      result.Limit,
			"scope":      scope,
		},
	}); err != nil {
		slog.Error("failed to encode rate limit error response", "error", err, "scope", scope)
	}
}

// TenantRateLimitMiddleware creates a middleware that rate limits by tenant
func (rl *RateLimiter) TenantRateLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			tenantID, ok := r.Context().Value(TenantIDKey).(uuid.UUID)
			if !ok {
				// No tenant ID in context, skip rate limiting
				next.ServeHTTP(w, r)
				return
			}

			result, err := rl.CheckTenant(r.Context(), tenantID)
			if err != nil {
				// On error, allow the request but log
				next.ServeHTTP(w, r)
				return
			}

			setRateLimitHeaders(w, result, RateLimitScopeTenant)

			if !result.Allowed {
				writeRateLimitError(w, result, RateLimitScopeTenant)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// WorkflowRateLimitMiddleware creates a middleware that rate limits by workflow
// This should be used on workflow-specific endpoints
func (rl *RateLimiter) WorkflowRateLimitMiddleware(getWorkflowID func(*http.Request) (uuid.UUID, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			tenantID, ok := r.Context().Value(TenantIDKey).(uuid.UUID)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			workflowID, err := getWorkflowID(r)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			result, err := rl.CheckWorkflow(r.Context(), tenantID, workflowID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			setRateLimitHeaders(w, result, RateLimitScopeWorkflow)

			if !result.Allowed {
				writeRateLimitError(w, result, RateLimitScopeWorkflow)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// WebhookRateLimitMiddleware creates a middleware that rate limits by webhook key
func (rl *RateLimiter) WebhookRateLimitMiddleware(getWebhookKey func(*http.Request) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			webhookKey, err := getWebhookKey(r)
			if err != nil || webhookKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			result, err := rl.CheckWebhook(r.Context(), webhookKey)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			setRateLimitHeaders(w, result, RateLimitScopeWebhook)

			if !result.Allowed {
				writeRateLimitError(w, result, RateLimitScopeWebhook)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MultiScopeRateLimitMiddleware checks multiple rate limits in sequence
// Returns 429 if any limit is exceeded
func (rl *RateLimiter) MultiScopeRateLimitMiddleware(
	getWorkflowID func(*http.Request) (uuid.UUID, error),
	getWebhookKey func(*http.Request) (string, error),
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			// Check webhook limit first (if applicable)
			if getWebhookKey != nil {
				webhookKey, err := getWebhookKey(r)
				if err == nil && webhookKey != "" {
					result, err := rl.CheckWebhook(ctx, webhookKey)
					if err == nil {
						setRateLimitHeaders(w, result, RateLimitScopeWebhook)
						if !result.Allowed {
							writeRateLimitError(w, result, RateLimitScopeWebhook)
							return
						}
					}
				}
			}

			// Check tenant limit
			tenantID, ok := ctx.Value(TenantIDKey).(uuid.UUID)
			if ok {
				result, err := rl.CheckTenant(ctx, tenantID)
				if err == nil {
					setRateLimitHeaders(w, result, RateLimitScopeTenant)
					if !result.Allowed {
						writeRateLimitError(w, result, RateLimitScopeTenant)
						return
					}
				}

				// Check workflow limit (if applicable)
				if getWorkflowID != nil {
					workflowID, err := getWorkflowID(r)
					if err == nil {
						result, err := rl.CheckWorkflow(ctx, tenantID, workflowID)
						if err == nil {
							setRateLimitHeaders(w, result, RateLimitScopeWorkflow)
							if !result.Allowed {
								writeRateLimitError(w, result, RateLimitScopeWorkflow)
								return
							}
						}
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UpdateConfig updates the rate limit configuration
func (rl *RateLimiter) UpdateConfig(config *RateLimitConfig) {
	rl.config = config
}

// GetConfig returns the current rate limit configuration
func (rl *RateLimiter) GetConfig() *RateLimitConfig {
	return rl.config
}
