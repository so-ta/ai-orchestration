package main

import (
	"context"
	"fmt"
	"log/slog"
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
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/souta/ai-orchestration/internal/copilot/agent"
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
	// Load .env file (try multiple locations)
	// First try parent directory (when running from backend/), then current directory
	envPaths := []string{
		"../.env",                                                   // When running from backend/
		".env",                                                      // Current directory
		"/Users/souta/Product/ai-orchestration/.env",               // Absolute path (dev)
	}
	var loaded bool
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			fmt.Printf("Loaded .env from: %s\n", path)
			loaded = true
			break
		}
	}
	if !loaded {
		cwd, _ := os.Getwd()
		fmt.Printf("Warning: Could not load .env file. CWD: %s\n", cwd)
	}

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting AI Orchestration API Server...")

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
		logger.Warn("Failed to initialize telemetry", "error", err)
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := telemetryProvider.Shutdown(shutdownCtx); err != nil {
				logger.Error("Error shutting down telemetry", "error", err)
			}
		}()
	}

	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable")
	pool, err := database.NewPool(ctx, database.DefaultConfig(dbURL))
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("Connected to database")

	// Redis connection
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	redisClient, err := redispkg.NewClient(ctx, &redispkg.Config{URL: redisURL})
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("Connected to Redis")

	// Initialize encryptor for credentials
	encryptor, err := crypto.NewEncryptor()
	if err != nil {
		logger.Error("Failed to initialize encryptor", "error", err)
		os.Exit(1)
	}
	logger.Info("Encryptor initialized")

	// Initialize repositories
	projectRepo := postgres.NewProjectRepository(pool)
	stepRepo := postgres.NewStepRepository(pool)
	edgeRepo := postgres.NewEdgeRepository(pool)
	runRepo := postgres.NewRunRepository(pool)
	stepRunRepo := postgres.NewStepRunRepository(pool)
	versionRepo := postgres.NewProjectVersionRepository(pool)
	scheduleRepo := postgres.NewScheduleRepository(pool)
	auditRepo := postgres.NewAuditLogRepository(pool)
	blockRepo := postgres.NewBlockDefinitionRepository(pool)
	blockVersionRepo := postgres.NewBlockVersionRepository(pool)
	blockGroupRepo := postgres.NewBlockGroupRepository(pool)
	credentialRepo := postgres.NewCredentialRepository(pool)
	copilotSessionRepo := postgres.NewCopilotSessionRepository(pool)
	usageRepo := postgres.NewUsageRepository(pool)
	budgetRepo := postgres.NewBudgetRepository(pool)
	tenantRepo := postgres.NewTenantRepository(pool)
	oauth2ProviderRepo := postgres.NewOAuth2ProviderRepository(pool)
	oauth2AppRepo := postgres.NewOAuth2AppRepository(pool)
	oauth2ConnectionRepo := postgres.NewOAuth2ConnectionRepository(pool)
	credentialShareRepo := postgres.NewCredentialShareRepository(pool)
	// N8N-style feature repositories
	// Note: AgentMemoryRepository and AgentChatSessionRepository are instantiated
	// on-demand within sandbox execution with proper tenant context
	agentMemoryRepo := postgres.NewAgentMemoryRepository(pool)
	agentChatSessionRepo := postgres.NewAgentChatSessionRepository(pool)
	// Silence unused variable warnings - these repos are passed to sandbox executor
	_ = agentMemoryRepo
	_ = agentChatSessionRepo
	templateRepo := postgres.NewProjectTemplateRepository(pool)
	templateReviewRepo := postgres.NewTemplateReviewRepository(pool)
	gitSyncRepo := postgres.NewProjectGitSyncRepository(pool)
	blockPackageRepo := postgres.NewCustomBlockPackageRepository(pool)

	// Initialize usecases
	projectUsecase := usecase.NewProjectUsecase(projectRepo, stepRepo, edgeRepo, versionRepo, blockRepo).
		WithBlockGroupRepo(blockGroupRepo)
	stepUsecase := usecase.NewStepUsecase(projectRepo, stepRepo, blockRepo, credentialRepo)
	edgeUsecase := usecase.NewEdgeUsecase(projectRepo, stepRepo, edgeRepo).
		WithBlockGroupRepo(blockGroupRepo).
		WithBlockDefinitionRepo(blockRepo)
	runUsecase := usecase.NewRunUsecase(projectRepo, runRepo, versionRepo, stepRepo, edgeRepo, stepRunRepo, redisClient)
	scheduleUsecase := usecase.NewScheduleUsecase(scheduleRepo, projectRepo, runRepo)
	auditService := usecase.NewAuditService(auditRepo)
	blockGroupUsecase := usecase.NewBlockGroupUsecase(projectRepo, blockGroupRepo, stepRepo)
	blockUsecase := usecase.NewBlockUsecase(blockRepo, blockVersionRepo)
	credentialUsecase := usecase.NewCredentialUsecase(credentialRepo, encryptor)
	copilotUsecase := usecase.NewCopilotUsecase(projectRepo, stepRepo, runRepo, stepRunRepo, copilotSessionRepo, blockRepo)
	usageUsecase := usecase.NewUsageUsecase(usageRepo, budgetRepo)

	// Agent-based copilot usecase
	agentUsecase := agent.NewAgentUsecase(
		copilotSessionRepo, blockRepo, projectRepo, stepRepo, edgeRepo, runRepo, stepRunRepo,
	)

	// OAuth2 service
	oauth2BaseURL := getEnv("BASE_URL", "http://localhost:8090")
	oauth2Service := usecase.NewOAuth2Service(
		oauth2ProviderRepo,
		oauth2AppRepo,
		oauth2ConnectionRepo,
		credentialRepo,
		encryptor,
		oauth2BaseURL,
	)
	credentialShareService := usecase.NewCredentialShareService(credentialShareRepo, credentialRepo)

	// N8N-style feature usecases
	templateUsecase := usecase.NewTemplateUsecase(templateRepo, templateReviewRepo, projectRepo, stepRepo, edgeRepo)
	gitSyncUsecase := usecase.NewGitSyncUsecase(gitSyncRepo, projectRepo)
	blockPackageUsecase := usecase.NewBlockPackageUsecase(blockPackageRepo, blockRepo)

	// Initialize handlers
	projectHandler := handler.NewProjectHandler(projectUsecase, auditService)
	stepHandler := handler.NewStepHandler(stepUsecase)
	edgeHandler := handler.NewEdgeHandler(edgeUsecase)
	runHandler := handler.NewRunHandler(runUsecase, auditService)
	scheduleHandler := handler.NewScheduleHandler(scheduleUsecase, auditService)
	auditHandler := handler.NewAuditHandler(auditService)
	blockHandler := handler.NewBlockHandler(blockRepo, blockUsecase)
	blockGroupHandler := handler.NewBlockGroupHandler(blockGroupUsecase)
	credentialHandler := handler.NewCredentialHandler(credentialUsecase, auditService)
	copilotHandler := handler.NewCopilotHandler(copilotUsecase, runUsecase)
	usageHandler := handler.NewUsageHandler(usageUsecase)
	adminTenantHandler := handler.NewAdminTenantHandler(tenantRepo)
	variablesHandler := handler.NewVariablesHandler(pool)
	oauth2Handler := handler.NewOAuth2Handler(oauth2Service, auditService)
	credentialShareHandler := handler.NewCredentialShareHandler(credentialShareService, auditService)
	copilotAgentHandler := handler.NewCopilotAgentHandler(agentUsecase)

	// N8N-style feature handlers
	templateHandler := handler.NewTemplateHandler(templateUsecase, auditService)
	gitSyncHandler := handler.NewGitSyncHandler(gitSyncUsecase, auditService)
	blockPackageHandler := handler.NewBlockPackageHandler(blockPackageUsecase, auditService)

	// Initialize auth middleware
	authConfig := &authmw.AuthConfig{
		KeycloakURL: getEnv("KEYCLOAK_URL", "http://localhost:8180"),
		Realm:       getEnv("KEYCLOAK_REALM", "ai-orchestration"),
		Enabled:     getEnv("AUTH_ENABLED", "false") == "true",
		DevTenantID: getEnv("DEV_TENANT_ID", "00000000-0000-0000-0000-000000000001"),
	}
	authMiddleware := authmw.NewAuthMiddleware(authConfig)
	logger.Info("Auth middleware configured", "enabled", authConfig.Enabled)

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
	logger.Info("Rate limiter configured",
		"enabled", rateLimitConfig.Enabled,
		"tenant_limit_per_min", rateLimitConfig.TenantLimit,
		"workflow_limit_per_min", rateLimitConfig.WorkflowLimit,
		"webhook_limit_per_min", rateLimitConfig.WebhookLimit)

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
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:3001", "http://127.0.0.1:3001"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Tenant-ID", "X-Dev-Role"},
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
			r.Get("/", projectHandler.List)
			r.Post("/", projectHandler.Create)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", projectHandler.Get)
				r.Put("/", projectHandler.Update)
				r.Delete("/", projectHandler.Delete)

				// Save and Draft operations
				r.Post("/save", projectHandler.Save)
				r.Post("/draft", projectHandler.SaveDraft)
				r.Delete("/draft", projectHandler.DiscardDraft)
				r.Post("/restore", projectHandler.RestoreVersion)

				// Deprecated: kept for backward compatibility
				r.Post("/publish", projectHandler.Publish)

				// Versions
				r.Route("/versions", func(r chi.Router) {
					r.Get("/", projectHandler.ListVersions)
					r.Get("/{version}", projectHandler.GetVersion)
				})

				// Steps
				r.Route("/steps", func(r chi.Router) {
					r.Get("/", stepHandler.List)
					r.Post("/", stepHandler.Create)
					r.Put("/{step_id}", stepHandler.Update)
					r.Delete("/{step_id}", stepHandler.Delete)

					// Inline step testing (without existing run)
					r.Post("/{step_id}/test", runHandler.TestStepInline)

					// Retry configuration (N8N-style)
					r.Get("/{step_id}/retry-config", stepHandler.GetRetryConfig)
					r.Put("/{step_id}/retry-config", stepHandler.UpdateRetryConfig)
					r.Delete("/{step_id}/retry-config", stepHandler.DeleteRetryConfig)

					// Step-specific Copilot
					r.Route("/{step_id}/copilot", func(r chi.Router) {
						r.Post("/suggest", copilotHandler.SuggestForStep)
						r.Post("/explain", copilotHandler.ExplainStep)
					})
				})

				// Git Sync (N8N-style)
				r.Route("/git-sync", func(r chi.Router) {
					r.Get("/", gitSyncHandler.GetByProject)
					r.Post("/", gitSyncHandler.Create)
					r.Put("/", gitSyncHandler.Update)
					r.Delete("/", gitSyncHandler.Delete)
					r.Post("/sync", gitSyncHandler.TriggerSync)
				})

				// Workflow-level Copilot (with session management)
				r.Route("/copilot", func(r chi.Router) {
					// Legacy endpoints
					r.Get("/session", copilotHandler.GetOrCreateSession)
					r.Post("/sessions/new", copilotHandler.StartNewSession)
					r.Post("/chat", copilotHandler.ChatWithSession)
					r.Post("/generate", copilotHandler.GenerateProject)

					// Builder session endpoints (integrated from /builder)
					r.Get("/sessions", copilotHandler.ListCopilotSessionsByProject)
					r.Post("/sessions", copilotHandler.StartCopilotSession)
					r.Route("/sessions/{session_id}", func(r chi.Router) {
						r.Get("/", copilotHandler.GetCopilotSession)
						r.Delete("/", copilotHandler.DeleteCopilotSession)
						r.Get("/messages", copilotHandler.GetSessionMessages)
						r.Post("/messages", copilotHandler.SendCopilotMessage)
						r.Post("/construct", copilotHandler.ConstructCopilotWorkflow)
						r.Post("/refine", copilotHandler.RefineCopilotWorkflow)
						r.Post("/finalize", copilotHandler.FinalizeCopilotSession)
					})

					// Agent-based copilot (NEW: autonomous tool-calling agent)
					r.Route("/agent", func(r chi.Router) {
						r.Post("/sessions", copilotAgentHandler.StartAgentSession)
						r.Route("/sessions/{session_id}", func(r chi.Router) {
							r.Post("/messages", copilotAgentHandler.SendAgentMessage)
							r.Get("/stream", copilotAgentHandler.StreamAgentMessage)
							r.Post("/cancel", copilotAgentHandler.CancelAgentStream)
						})
					})
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

				// Credential shares
				r.Route("/shares", func(r chi.Router) {
					r.Get("/", credentialShareHandler.ListByCredential)
					r.Post("/user", credentialShareHandler.ShareWithUser)
					r.Post("/project", credentialShareHandler.ShareWithProject)
					r.Put("/{share_id}", credentialShareHandler.UpdateShare)
					r.Delete("/{share_id}", credentialShareHandler.RevokeShare)
				})
			})
		})

		// OAuth2 (External service authentication)
		r.Route("/oauth2", func(r chi.Router) {
			// Providers (read-only list of supported OAuth2 providers)
			r.Get("/providers", oauth2Handler.ListProviders)
			r.Get("/providers/{slug}", oauth2Handler.GetProvider)

			// Apps (tenant's OAuth2 client configurations)
			r.Route("/apps", func(r chi.Router) {
				r.Get("/", oauth2Handler.ListApps)
				r.Post("/", oauth2Handler.CreateApp)
				r.Route("/{app_id}", func(r chi.Router) {
					r.Get("/", oauth2Handler.GetApp)
					r.Delete("/", oauth2Handler.DeleteApp)
				})
			})

			// Authorization flow
			r.Post("/authorize/start", oauth2Handler.StartAuthorization)
			r.Get("/callback", oauth2Handler.HandleCallback)

			// Connections (OAuth2 token management)
			r.Route("/connections", func(r chi.Router) {
				r.Get("/{connection_id}", oauth2Handler.GetConnection)
				r.Post("/{connection_id}/refresh", oauth2Handler.RefreshConnection)
				r.Post("/{connection_id}/revoke", oauth2Handler.RevokeConnection)
				r.Delete("/{connection_id}", oauth2Handler.DeleteConnection)
			})

			// Connection by credential (for looking up connection from credential)
			r.Get("/connections/by-credential/{credential_id}", oauth2Handler.GetConnectionByCredential)
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
				r.Post("/generate", copilotHandler.AsyncGenerateProject)
				r.Post("/suggest", copilotHandler.AsyncSuggest)
				r.Post("/diagnose", copilotHandler.AsyncDiagnose)
				r.Post("/optimize", copilotHandler.AsyncOptimize)
			})

			// Polling endpoint for async results
			r.Get("/runs/{id}", copilotHandler.GetCopilotRun)

			// Agent tools (list available tools for the agent)
			r.Get("/agent/tools", copilotAgentHandler.GetAvailableTools)
		})

		// Usage tracking and cost management
		r.Route("/usage", func(r chi.Router) {
			r.Get("/summary", usageHandler.GetSummary)
			r.Get("/daily", usageHandler.GetDaily)
			r.Get("/by-workflow", usageHandler.GetByProject)
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

		// Tenant variables (organization-level)
		r.Route("/tenant/variables", func(r chi.Router) {
			r.Get("/", variablesHandler.GetTenantVariables)
			r.Put("/", variablesHandler.UpdateTenantVariables)
		})

		// User variables (personal)
		r.Route("/user/variables", func(r chi.Router) {
			r.Get("/", variablesHandler.GetUserVariables)
			r.Put("/", variablesHandler.UpdateUserVariables)
		})

		// ============================================================================
		// N8N-Style Features (Phase 2-4)
		// ============================================================================

		// Templates (N8N-style workflow templates)
		r.Route("/templates", func(r chi.Router) {
			r.Get("/", templateHandler.List)
			r.Post("/", templateHandler.Create)
			r.Post("/from-project", templateHandler.CreateFromProject)
			r.Get("/categories", templateHandler.GetCategories)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", templateHandler.Get)
				r.Put("/", templateHandler.Update)
				r.Delete("/", templateHandler.Delete)
				r.Post("/use", templateHandler.Use)
				r.Get("/reviews", templateHandler.GetReviews)
				r.Post("/reviews", templateHandler.AddReview)
			})
		})

		// Template Marketplace
		r.Route("/marketplace", func(r chi.Router) {
			r.Get("/templates", templateHandler.ListPublic)
		})

		// Git Sync (global list)
		r.Route("/git-sync", func(r chi.Router) {
			r.Get("/", gitSyncHandler.List)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", gitSyncHandler.Get)
				r.Put("/", gitSyncHandler.Update)
				r.Delete("/", gitSyncHandler.Delete)
				r.Post("/sync", gitSyncHandler.TriggerSync)
			})
		})

		// Custom Block Packages (Block SDK)
		r.Route("/block-packages", func(r chi.Router) {
			r.Get("/", blockPackageHandler.List)
			r.Post("/", blockPackageHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", blockPackageHandler.Get)
				r.Put("/", blockPackageHandler.Update)
				r.Delete("/", blockPackageHandler.Delete)
				r.Post("/publish", blockPackageHandler.Publish)
				r.Post("/deprecate", blockPackageHandler.Deprecate)
			})
		})

		// Admin routes for system block management
		r.Route("/admin/blocks", func(r chi.Router) {
			r.Use(authmw.RequireAdmin)
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
			r.Use(authmw.RequireAdmin)
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

	// Server
	port := getEnv("PORT", "8090")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Server listening", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited gracefully")
}

func healthHandler(pool *pgxpool.Pool, redisClient redisPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Liveness probe - basic check that the service is running
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			slog.Debug("failed to write health response", "error", err)
		}
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
		if _, err := w.Write([]byte(response)); err != nil {
			slog.Debug("failed to write readiness response", "error", err)
		}
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
