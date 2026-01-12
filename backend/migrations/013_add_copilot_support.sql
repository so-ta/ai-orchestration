-- Migration: 013_add_copilot_support.sql
-- Description: Add support for Copilot meta-workflow architecture
-- - Add trigger_source and trigger_metadata to runs table
-- - Add is_system and system_slug to workflows table

-- ============================================
-- 1. Extend runs table for internal triggers
-- ============================================

-- Add trigger_source column (identifies internal caller: copilot, audit-system, etc.)
ALTER TABLE runs ADD COLUMN IF NOT EXISTS trigger_source VARCHAR(100);

-- Add trigger_metadata column (stores feature, user_id, session_id, etc.)
ALTER TABLE runs ADD COLUMN IF NOT EXISTS trigger_metadata JSONB DEFAULT '{}';

-- Add index for querying by trigger_source
CREATE INDEX IF NOT EXISTS idx_runs_trigger_source ON runs(trigger_source)
    WHERE trigger_source IS NOT NULL;

-- Add comments
COMMENT ON COLUMN runs.trigger_source IS 'Internal trigger source identifier: copilot, audit-system, etc.';
COMMENT ON COLUMN runs.trigger_metadata IS 'Additional metadata about the trigger: feature, user_id, session_id, etc.';

-- ============================================
-- 2. Extend workflows table for system workflows
-- ============================================

-- Add is_system column (true for system workflows like Copilot)
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE;

-- Add system_slug column (unique identifier for system workflows)
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS system_slug VARCHAR(100);

-- Add unique index for system_slug (only when not null)
CREATE UNIQUE INDEX IF NOT EXISTS idx_workflows_system_slug ON workflows(system_slug)
    WHERE system_slug IS NOT NULL;

-- Add comments
COMMENT ON COLUMN workflows.is_system IS 'True for system workflows (e.g., Copilot). These are accessible across all tenants.';
COMMENT ON COLUMN workflows.system_slug IS 'Unique slug for system workflows (e.g., copilot-generate). Used for internal lookups.';

-- ============================================
-- 3. Update triggered_by constraint to include 'internal'
-- ============================================

-- Note: PostgreSQL doesn't support modifying CHECK constraints directly.
-- If there's an existing constraint, we need to drop and recreate it.
-- For safety, we'll just ensure the column accepts the new value.

-- The triggered_by column is VARCHAR, so no constraint update needed.
-- The application layer (Go code) handles validation.
