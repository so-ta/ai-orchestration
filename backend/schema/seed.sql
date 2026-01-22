-- AI Orchestration Seed Data
-- This file contains initial data for the database
--
-- NOTE: Block definitions and projects are now managed programmatically.
-- Use the seeder command to migrate them:
--   go run ./cmd/seeder              # Migrate blocks and projects
--   go run ./cmd/seeder --validate   # Validate only
--   go run ./cmd/seeder --dry-run    # Preview changes
--
-- This file only contains tenant data for initial setup.

SET search_path = public;

-- ============================================================================
-- TRUNCATE ALL TABLES (for db-reset)
-- ============================================================================
-- This ensures a clean slate before seeding.
-- Uses IF EXISTS to work on both initial setup and subsequent resets.
-- CASCADE handles FK constraints automatically.

DO $$
DECLARE
    tables TEXT[] := ARRAY[
        'copilot_messages',
        'copilot_sessions',
        'agent_memory',
        'agent_chat_sessions',
        'step_runs',
        'runs',
        'run_number_sequences',
        'edges',
        'steps',
        'block_groups',
        'schedules',
        'project_versions',
        'template_reviews',
        'project_templates',
        'project_git_sync',
        'projects',
        'block_versions',
        'block_definitions',
        'custom_block_packages',
        'credentials',
        'system_credentials',
        'secrets',
        'usage_records',
        'usage_daily_aggregates',
        'usage_budgets',
        'audit_logs',
        'vector_documents',
        'vector_collections',
        'users',
        'tenants'
    ];
    t TEXT;
BEGIN
    FOREACH t IN ARRAY tables
    LOOP
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = t) THEN
            EXECUTE format('TRUNCATE TABLE %I CASCADE', t);
        END IF;
    END LOOP;
END $$;

--
-- Data for Name: tenants; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO tenants (id, name, slug, settings, created_at, updated_at, deleted_at, status, plan, owner_email, owner_name, billing_email, metadata, feature_flags, limits, suspended_at, suspended_reason) VALUES ('00000000-0000-0000-0000-000000000001', 'Default Tenant', 'default-tenant', '{"data_retention_days": 30}', '2026-01-12 09:22:38.988109+00', '2026-01-12 09:22:38.988109+00', NULL, 'active', 'free', NULL, NULL, NULL, '{}', '{"api_access": true, "audit_logs": true, "sso_enabled": false, "custom_blocks": true, "copilot_enabled": true, "advanced_analytics": true, "max_concurrent_runs": 10}', '{"max_users": 50, "max_workflows": 100, "max_storage_mb": 10240, "retention_days": 90, "max_credentials": 100, "max_runs_per_day": 1000}', NULL, NULL)
ON CONFLICT (id) DO NOTHING;

--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: -
-- Default development user (used when AUTH_ENABLED=false)
--

INSERT INTO users (id, tenant_id, email, name, role, created_at, updated_at)
VALUES (
    '4c560a4e-ac47-4bcc-9e5e-4981fe6e98f7',
    '00000000-0000-0000-0000-000000000001',
    'admin@example.com',
    'Admin User',
    'admin',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;

-- NOTE: Block definitions are managed by programmatic seeder.
-- See: backend/internal/seed/blocks/
-- Run: go run ./cmd/seeder --blocks-only

-- NOTE: Projects are managed by programmatic seeder.
-- See: backend/internal/seed/projects/
-- Run: go run ./cmd/seeder --projects-only
