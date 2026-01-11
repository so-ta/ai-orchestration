-- Migration: Add draft field to workflows table
-- This enables "Save as Draft" functionality without creating version snapshots

-- Add draft column to workflows table
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS draft JSONB;

-- Update default version to 0 for new workflows (no versions yet)
-- Existing workflows keep their versions

-- Rename workflow_versions columns for clarity
ALTER TABLE workflow_versions RENAME COLUMN published_by TO saved_by;
ALTER TABLE workflow_versions RENAME COLUMN published_at TO saved_at;
