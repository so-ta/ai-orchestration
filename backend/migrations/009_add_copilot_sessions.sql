-- +migrate Up
-- Copilot chat sessions (per user per workflow)
CREATE TABLE copilot_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    title VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_copilot_sessions_tenant ON copilot_sessions(tenant_id);
CREATE INDEX idx_copilot_sessions_user_workflow ON copilot_sessions(tenant_id, user_id, workflow_id);
CREATE INDEX idx_copilot_sessions_active ON copilot_sessions(tenant_id, user_id, workflow_id, is_active) WHERE is_active = true;

-- Copilot chat messages
CREATE TABLE copilot_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES copilot_sessions(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'assistant')),
    content TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_copilot_messages_session ON copilot_messages(session_id);
CREATE INDEX idx_copilot_messages_created ON copilot_messages(session_id, created_at);

-- +migrate Down
DROP TABLE IF EXISTS copilot_messages;
DROP TABLE IF EXISTS copilot_sessions;
