-- Migration: Add block_groups table for control flow constructs
-- Date: 2025-01-11
-- Description: Introduces BlockGroup entity for grouping steps into control flow structures
--              (parallel, try-catch-finally, if-else, switch-case, foreach, while)

-- Create block_groups table
CREATE TABLE IF NOT EXISTS block_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    parent_group_id UUID REFERENCES block_groups(id) ON DELETE CASCADE,
    position_x INT DEFAULT 0,
    position_y INT DEFAULT 0,
    width INT DEFAULT 400,
    height INT DEFAULT 300,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraint: valid block group type
    CONSTRAINT valid_block_group_type CHECK (type IN (
        'parallel',
        'try_catch',
        'if_else',
        'switch_case',
        'foreach',
        'while'
    ))
);

-- Add block_group_id and group_role to steps table
ALTER TABLE steps ADD COLUMN IF NOT EXISTS block_group_id UUID REFERENCES block_groups(id) ON DELETE SET NULL;
ALTER TABLE steps ADD COLUMN IF NOT EXISTS group_role VARCHAR(50);

-- Create block_group_runs table for tracking group execution
CREATE TABLE IF NOT EXISTS block_group_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    block_group_id UUID NOT NULL REFERENCES block_groups(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'pending',
    iteration INT DEFAULT 0,
    input JSONB,
    output JSONB,
    error TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraint: valid status
    CONSTRAINT valid_block_group_run_status CHECK (status IN (
        'pending',
        'running',
        'completed',
        'failed',
        'skipped'
    ))
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_block_groups_workflow ON block_groups(workflow_id);
CREATE INDEX IF NOT EXISTS idx_block_groups_parent ON block_groups(parent_group_id);
CREATE INDEX IF NOT EXISTS idx_steps_block_group ON steps(block_group_id);
CREATE INDEX IF NOT EXISTS idx_block_group_runs_run ON block_group_runs(run_id);
CREATE INDEX IF NOT EXISTS idx_block_group_runs_block_group ON block_group_runs(block_group_id);

-- Comments
COMMENT ON TABLE block_groups IS 'Control flow constructs that group multiple steps';
COMMENT ON COLUMN block_groups.type IS 'Type of control flow: parallel, try_catch, if_else, switch_case, foreach, while';
COMMENT ON COLUMN block_groups.config IS 'Type-specific configuration (JSON)';
COMMENT ON COLUMN block_groups.parent_group_id IS 'Reference to parent group for nested structures';
COMMENT ON COLUMN steps.block_group_id IS 'Reference to containing block group (NULL if not in a group)';
COMMENT ON COLUMN steps.group_role IS 'Role within block group: body, try, catch, finally, then, else, case_N, default';
