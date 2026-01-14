-- AI Orchestration Seed Data
-- This file contains initial data for the database
--
-- NOTE: Block definitions and workflows are now managed programmatically.
-- Use the seeder command to migrate them:
--   go run ./cmd/seeder              # Migrate blocks and workflows
--   go run ./cmd/seeder --validate   # Validate only
--   go run ./cmd/seeder --dry-run    # Preview changes
--
-- This file only contains tenant data for initial setup.

SET search_path = public;

--
-- Data for Name: tenants; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO tenants (id, name, slug, settings, created_at, updated_at, deleted_at, status, plan, owner_email, owner_name, billing_email, metadata, feature_flags, limits, suspended_at, suspended_reason) VALUES ('00000000-0000-0000-0000-000000000001', 'Default Tenant', 'default-tenant', '{"data_retention_days": 30}', '2026-01-12 09:22:38.988109+00', '2026-01-12 09:22:38.988109+00', NULL, 'active', 'free', NULL, NULL, NULL, '{}', '{"api_access": true, "audit_logs": true, "sso_enabled": false, "custom_blocks": true, "copilot_enabled": true, "advanced_analytics": true, "max_concurrent_runs": 10}', '{"max_users": 50, "max_workflows": 100, "max_storage_mb": 10240, "retention_days": 90, "max_credentials": 100, "max_runs_per_day": 1000}', NULL, NULL)
ON CONFLICT (id) DO NOTHING;

-- NOTE: Block definitions are managed by programmatic seeder.
-- See: backend/internal/seed/blocks/
-- Run: go run ./cmd/seeder --blocks-only

-- NOTE: Workflows are managed by programmatic seeder.
-- See: backend/internal/seed/workflows/
-- Run: go run ./cmd/seeder --workflows-only
