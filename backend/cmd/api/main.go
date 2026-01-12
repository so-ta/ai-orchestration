package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/souta/ai-orchestration/internal/handler"
	authmw "github.com/souta/ai-orchestration/internal/middleware"
	"github.com/souta/ai-orchestration/internal/repository/postgres"
	"github.com/souta/ai-orchestration/internal/usecase"
	"github.com/souta/ai-orchestration/pkg/crypto"
	"github.com/souta/ai-orchestration/pkg/database"
	redispkg "github.com/souta/ai-orchestration/pkg/redis"
	"github.com/souta/ai-orchestration/pkg/telemetry"
)

func main() {
	log.Println("Starting AI Orchestration API Server...")

	ctx := context.Background()

	// Initialize telemetry
	telemetryConfig := &telemetry.Config{
		ServiceName:    "ai-orchestration-api",
		ServiceVersion: "1.0.0",
		Environment:    getEnv("ENVIRONMENT", "development"),
		OTLPEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		Enabled:        getEnv("TELEMETRY_ENABLED", "false") == "true",
	}
	telemetryProvider, err := telemetry.NewProvider(ctx, telemetryConfig)
	if err != nil {
		log.Printf("Warning: Failed to initialize telemetry: %v", err)
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := telemetryProvider.Shutdown(shutdownCtx); err != nil {
				log.Printf("Error shutting down telemetry: %v", err)
			}
		}()
	}

	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable")
	pool, err := database.NewPool(ctx, database.DefaultConfig(dbURL))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to database")

	// Redis connection
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	redisClient, err := redispkg.NewClient(ctx, &redispkg.Config{URL: redisURL})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis")

	// Initialize encryptor for credentials
	encryptor, err := crypto.NewEncryptor()
	if err != nil {
		log.Fatalf("Failed to initialize encryptor: %v", err)
	}
	log.Println("Encryptor initialized")

	// Initialize repositories
	workflowRepo := postgres.NewWorkflowRepository(pool)
	stepRepo := postgres.NewStepRepository(pool)
	edgeRepo := postgres.NewEdgeRepository(pool)
	runRepo := postgres.NewRunRepository(pool)
	stepRunRepo := postgres.NewStepRunRepository(pool)
	versionRepo := postgres.NewWorkflowVersionRepository(pool)
	scheduleRepo := postgres.NewScheduleRepository(pool)
	webhookRepo := postgres.NewWebhookRepository(pool)
	auditRepo := postgres.NewAuditLogRepository(pool)
	blockRepo := postgres.NewBlockDefinitionRepository(pool)
	blockVersionRepo := postgres.NewBlockVersionRepository(pool)
	blockGroupRepo := postgres.NewBlockGroupRepository(pool)
	credentialRepo := postgres.NewCredentialRepository(pool)
	copilotSessionRepo := postgres.NewCopilotSessionRepository(pool)
	usageRepo := postgres.NewUsageRepository(pool)
	budgetRepo := postgres.NewBudgetRepository(pool)
	tenantRepo := postgres.NewTenantRepository(pool)

	// Initialize usecases
	workflowUsecase := usecase.NewWorkflowUsecase(workflowRepo, stepRepo, edgeRepo, versionRepo)
	stepUsecase := usecase.NewStepUsecase(workflowRepo, stepRepo)
	edgeUsecase := usecase.NewEdgeUsecase(workflowRepo, stepRepo, edgeRepo)
	runUsecase := usecase.NewRunUsecase(workflowRepo, runRepo, versionRepo, stepRepo, edgeRepo, stepRunRepo, redisClient)
	scheduleUsecase := usecase.NewScheduleUsecase(scheduleRepo, workflowRepo, runRepo)
	webhookUsecase := usecase.NewWebhookUsecase(webhookRepo, workflowRepo, runRepo)
	auditService := usecase.NewAuditService(auditRepo)
	blockGroupUsecase := usecase.NewBlockGroupUsecase(workflowRepo, blockGroupRepo, stepRepo)
	blockUsecase := usecase.NewBlockUsecase(blockRepo, blockVersionRepo)
	credentialUsecase := usecase.NewCredentialUsecase(credentialRepo, encryptor)
	copilotUsecase := usecase.NewCopilotUsecase(workflowRepo, stepRepo, runRepo, stepRunRepo, copilotSessionRepo)
	usageUsecase := usecase.NewUsageUsecase(usageRepo, budgetRepo)

	// Initialize handlers
	workflowHandler := handler.NewWorkflowHandler(workflowUsecase)
	stepHandler := handler.NewStepHandler(stepUsecase)
	edgeHandler := handler.NewEdgeHandler(edgeUsecase)
	runHandler := handler.NewRunHandler(runUsecase)
	scheduleHandler := handler.NewScheduleHandler(scheduleUsecase)
	webhookHandler := handler.NewWebhookHandler(webhookUsecase)
	auditHandler := handler.NewAuditHandler(auditService)
	blockHandler := handler.NewBlockHandler(blockRepo, blockUsecase)
	blockGroupHandler := handler.NewBlockGroupHandler(blockGroupUsecase)
	credentialHandler := handler.NewCredentialHandler(credentialUsecase)
	copilotHandler := handler.NewCopilotHandler(copilotUsecase, runUsecase)
	usageHandler := handler.NewUsageHandler(usageUsecase)
	adminTenantHandler := handler.NewAdminTenantHandler(tenantRepo)

	// Initialize auth middleware
	authConfig := &authmw.AuthConfig{
		KeycloakURL: getEnv("KEYCLOAK_URL", "http://localhost:8180"),
		Realm:       getEnv("KEYCLOAK_REALM", "ai-orchestration"),
		Enabled:     getEnv("AUTH_ENABLED", "false") == "true",
		DevTenantID: getEnv("DEV_TENANT_ID", "00000000-0000-0000-0000-000000000001"),
	}
	authMiddleware := authmw.NewAuthMiddleware(authConfig)
	log.Printf("Auth middleware enabled: %v", authConfig.Enabled)

	// Initialize rate limiter
	rateLimitConfig := &authmw.RateLimitConfig{
		Enabled:        getEnv("RATE_LIMIT_ENABLED", "true") == "true",
		TenantLimit:    getEnvInt("RATE_LIMIT_TENANT", 1000),
		TenantWindow:   time.Minute,
		WorkflowLimit:  getEnvInt("RATE_LIMIT_WORKFLOW", 100),
		WorkflowWindow: time.Minute,
		WebhookLimit:   getEnvInt("RATE_LIMIT_WEBHOOK", 60),
		WebhookWindow:  time.Minute,
	}
	rateLimiter := authmw.NewRateLimiter(redisClient, rateLimitConfig)
	log.Printf("Rate limiter enabled: %v (tenant: %d/min, workflow: %d/min, webhook: %d/min)",
		rateLimitConfig.Enabled,
		rateLimitConfig.TenantLimit,
		rateLimitConfig.WorkflowLimit,
		rateLimitConfig.WebhookLimit)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Telemetry middleware (OpenTelemetry tracing)
	if telemetryProvider != nil && telemetryProvider.IsEnabled() {
		r.Use(telemetry.HTTPMiddleware)
	}

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Tenant-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", healthHandler(pool, redisClient))
	r.Get("/ready", readinessHandler(pool, redisClient))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth middleware
		r.Use(authMiddleware.Handler)
		// Tenant-level rate limiting
		r.Use(rateLimiter.TenantRateLimitMiddleware())

		// Workflows
		r.Route("/workflows", func(r chi.Router) {
			r.Get("/", workflowHandler.List)
			r.Post("/", workflowHandler.Create)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", workflowHandler.Get)
				r.Put("/", workflowHandler.Update)
				r.Delete("/", workflowHandler.Delete)

				// Save and Draft operations
				r.Post("/save", workflowHandler.Save)
				r.Post("/draft", workflowHandler.SaveDraft)
				r.Delete("/draft", workflowHandler.DiscardDraft)
				r.Post("/restore", workflowHandler.RestoreVersion)

				// Deprecated: kept for backward compatibility
				r.Post("/publish", workflowHandler.Publish)

				// Versions
				r.Route("/versions", func(r chi.Router) {
					r.Get("/", workflowHandler.ListVersions)
					r.Get("/{version}", workflowHandler.GetVersion)
				})

				// Steps
				r.Route("/steps", func(r chi.Router) {
					r.Get("/", stepHandler.List)
					r.Post("/", stepHandler.Create)
					r.Put("/{step_id}", stepHandler.Update)
					r.Delete("/{step_id}", stepHandler.Delete)

					// Step-specific Copilot
					r.Route("/{step_id}/copilot", func(r chi.Router) {
						r.Post("/suggest", copilotHandler.SuggestForStep)
						r.Post("/explain", copilotHandler.ExplainStep)
					})
				})

				// Workflow-level Copilot (with session management)
				r.Route("/copilot", func(r chi.Router) {
					r.Get("/session", copilotHandler.GetOrCreateSession)
					r.Get("/sessions", copilotHandler.ListSessions)
					r.Post("/sessions/new", copilotHandler.StartNewSession)
					r.Get("/sessions/{session_id}", copilotHandler.GetSessionMessages)
					r.Post("/chat", copilotHandler.ChatWithSession)
					r.Post("/generate", copilotHandler.GenerateWorkflow)
				})

				// Edges
				r.Route("/edges", func(r chi.Router) {
					r.Get("/", edgeHandler.List)
					r.Post("/", edgeHandler.Create)
					r.Delete("/{edge_id}", edgeHandler.Delete)
				})

				// Block Groups
				r.Route("/block-groups", func(r chi.Router) {
					r.Get("/", blockGroupHandler.List)
					r.Post("/", blockGroupHandler.Create)
					r.Route("/{group_id}", func(r chi.Router) {
						r.Get("/", blockGroupHandler.Get)
						r.Put("/", blockGroupHandler.Update)
						r.Delete("/", blockGroupHandler.Delete)
						r.Get("/steps", blockGroupHandler.GetStepsByGroup)
						r.Post("/steps", blockGroupHandler.AddStepToGroup)
						r.Delete("/steps/{step_id}", blockGroupHandler.RemoveStepFromGroup)
					})
				})

				// Runs (with workflow-level rate limiting for creation)
				r.Route("/runs", func(r chi.Router) {
					r.Get("/", runHandler.List)
					r.With(rateLimiter.WorkflowRateLimitMiddleware(func(req *http.Request) (uuid.UUID, error) {
						return uuid.Parse(chi.URLParam(req, "id"))
					})).Post("/", runHandler.Create)
				})
			})
		})

		// Runs (direct access)
		r.Route("/runs", func(r chi.Router) {
			r.Get("/{run_id}", runHandler.Get)
			r.Post("/{run_id}/cancel", runHandler.Cancel)
			r.Post("/{run_id}/resume", runHandler.ResumeFromStep)

			// Step execution and history
			r.Route("/{run_id}/steps/{step_id}", func(r chi.Router) {
				r.Post("/execute", runHandler.ExecuteSingleStep)
				r.Get("/history", runHandler.GetStepHistory)
			})
		})

		// Schedules
		r.Route("/schedules", func(r chi.Router) {
			r.Get("/", scheduleHandler.List)
			r.Post("/", scheduleHandler.Create)
			r.Route("/{schedule_id}", func(r chi.Router) {
				r.Get("/", scheduleHandler.Get)
				r.Put("/", scheduleHandler.Update)
				r.Delete("/", scheduleHandler.Delete)
				r.Post("/pause", scheduleHandler.Pause)
				r.Post("/resume", scheduleHandler.Resume)
				r.Post("/trigger", scheduleHandler.Trigger)
			})
		})

		// Webhooks
		r.Route("/webhooks", func(r chi.Router) {
			r.Get("/", webhookHandler.List)
			r.Post("/", webhookHandler.Create)
			r.Route("/{webhook_id}", func(r chi.Router) {
				r.Get("/", webhookHandler.Get)
				r.Put("/", webhookHandler.Update)
				r.Delete("/", webhookHandler.Delete)
				r.Post("/enable", webhookHandler.Enable)
				r.Post("/disable", webhookHandler.Disable)
				r.Post("/regenerate-secret", webhookHandler.RegenerateSecret)
			})
		})

		// Audit logs
		r.Route("/audit-logs", func(r chi.Router) {
			r.Get("/", auditHandler.List)
			r.Get("/{resource_type}/{resource_id}", auditHandler.GetByResource)
		})

		// Block definitions (Block Registry)
		r.Route("/blocks", func(r chi.Router) {
			r.Get("/", blockHandler.List)
			r.Post("/", blockHandler.Create)
			r.Get("/{slug}", blockHandler.Get)
			r.Put("/{slug}", blockHandler.Update)
			r.Delete("/{slug}", blockHandler.Delete)
		})

		// Credentials (API keys, tokens, etc.)
		r.Route("/credentials", func(r chi.Router) {
			r.Get("/", credentialHandler.List)
			r.Post("/", credentialHandler.Create)
			r.Route("/{credential_id}", func(r chi.Router) {
				r.Get("/", credentialHandler.Get)
				r.Put("/", credentialHandler.Update)
				r.Delete("/", credentialHandler.Delete)
				r.Post("/revoke", credentialHandler.Revoke)
				r.Post("/activate", credentialHandler.Activate)
			})
		})

		// Copilot (AI-assisted workflow building)
		r.Route("/copilot", func(r chi.Router) {
			// Legacy synchronous endpoints
			r.Post("/suggest", copilotHandler.Suggest)
			r.Post("/diagnose", copilotHandler.Diagnose)
			r.Post("/explain", copilotHandler.Explain)
			r.Post("/optimize", copilotHandler.Optimize)
			r.Post("/chat", copilotHandler.Chat)

			// Async endpoints (meta-workflow architecture)
			r.Route("/async", func(r chi.Router) {
				r.Post("/generate", copilotHandler.AsyncGenerateWorkflow)
				r.Post("/suggest", copilotHandler.AsyncSuggest)
				r.Post("/diagnose", copilotHandler.AsyncDiagnose)
				r.Post("/optimize", copilotHandler.AsyncOptimize)
			})

			// Polling endpoint for async results
			r.Get("/runs/{id}", copilotHandler.GetCopilotRun)
		})

		// Usage tracking and cost management
		r.Route("/usage", func(r chi.Router) {
			r.Get("/summary", usageHandler.GetSummary)
			r.Get("/daily", usageHandler.GetDaily)
			r.Get("/by-workflow", usageHandler.GetByWorkflow)
			r.Get("/by-model", usageHandler.GetByModel)
			r.Get("/pricing", usageHandler.GetPricing)

			// Budget management
			r.Route("/budgets", func(r chi.Router) {
				r.Get("/", usageHandler.ListBudgets)
				r.Post("/", usageHandler.CreateBudget)
				r.Put("/{id}", usageHandler.UpdateBudget)
				r.Delete("/{id}", usageHandler.DeleteBudget)
			})
		})

		// Run usage (nested under runs)
		r.Get("/runs/{run_id}/usage", usageHandler.GetByRun)

		// Admin routes for system block management
		r.Route("/admin/blocks", func(r chi.Router) {
			// TODO: Add admin role check middleware
			r.Get("/", blockHandler.ListSystemBlocks)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", blockHandler.GetSystemBlock)
				r.Put("/", blockHandler.UpdateSystemBlock)
				r.Get("/versions", blockHandler.ListBlockVersions)
				r.Get("/versions/{version}", blockHandler.GetBlockVersion)
				r.Post("/rollback", blockHandler.RollbackBlock)
			})
		})

		// Admin routes for tenant management
		r.Route("/admin/tenants", func(r chi.Router) {
			// TODO: Add admin role check middleware
			r.Get("/", adminTenantHandler.List)
			r.Post("/", adminTenantHandler.Create)
			r.Get("/stats/overview", adminTenantHandler.GetOverviewStats)
			r.Route("/{tenant_id}", func(r chi.Router) {
				r.Get("/", adminTenantHandler.Get)
				r.Put("/", adminTenantHandler.Update)
				r.Delete("/", adminTenantHandler.Delete)
				r.Post("/suspend", adminTenantHandler.Suspend)
				r.Post("/activate", adminTenantHandler.Activate)
				r.Get("/stats", adminTenantHandler.GetStats)
			})
		})
	})

	// Public webhook trigger endpoint (no auth required, but with webhook rate limiting)
	r.With(rateLimiter.WebhookRateLimitMiddleware(func(req *http.Request) (string, error) {
		return chi.URLParam(req, "webhook_id"), nil
	})).Post("/api/v1/webhooks/{webhook_id}/trigger", webhookHandler.Trigger)

	// Server
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func healthHandler(pool *pgxpool.Pool, redisClient redisPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Liveness probe - basic check that the service is running
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}

// redisPinger interface for Redis health check
type redisPinger interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

func readinessHandler(pool *pgxpool.Pool, redisClient redisPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check database
		dbStatus := "ok"
		if err := pool.Ping(ctx); err != nil {
			dbStatus = "error"
		}

		// Check Redis
		redisStatus := "ok"
		if err := redisClient.Ping(ctx).Err(); err != nil {
			redisStatus = "error"
		}

		// Determine overall status
		status := "ok"
		httpStatus := http.StatusOK
		if dbStatus != "ok" || redisStatus != "ok" {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		response := fmt.Sprintf(`{"status":"%s","components":{"database":"%s","redis":"%s"}}`,
			status, dbStatus, redisStatus)
		w.Write([]byte(response))
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
