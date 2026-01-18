-- External Authentication Support Migration
-- Adds OAuth2 providers, multi-scope credentials, and credential sharing
-- Migration: 014_external_auth.sql

-- ============================================================================
-- 1. Extend credentials table with scope support
-- ============================================================================

-- Add scope columns to credentials table
ALTER TABLE credentials
ADD COLUMN IF NOT EXISTS scope VARCHAR(20) NOT NULL DEFAULT 'organization',
ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
ADD COLUMN IF NOT EXISTS owner_user_id UUID REFERENCES users(id) ON DELETE CASCADE;

-- Add scope constraint
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS credentials_scope_check;
ALTER TABLE credentials ADD CONSTRAINT credentials_scope_check CHECK (
    (scope = 'organization' AND project_id IS NULL AND owner_user_id IS NULL) OR
    (scope = 'project' AND project_id IS NOT NULL AND owner_user_id IS NULL) OR
    (scope = 'personal' AND project_id IS NULL AND owner_user_id IS NOT NULL)
);

-- Update valid credential types to include new types
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS valid_credential_type;
ALTER TABLE credentials ADD CONSTRAINT valid_credential_type CHECK (
    credential_type IN ('oauth2', 'api_key', 'basic', 'bearer', 'custom', 'query_auth', 'header_auth')
);

-- Add indexes for scope-based queries
CREATE INDEX IF NOT EXISTS idx_credentials_scope ON credentials(tenant_id, scope);
CREATE INDEX IF NOT EXISTS idx_credentials_project ON credentials(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_credentials_owner ON credentials(owner_user_id) WHERE owner_user_id IS NOT NULL;

-- Add comments
COMMENT ON COLUMN credentials.scope IS 'Credential scope: organization (tenant-wide), project (project-specific), personal (user-specific)';
COMMENT ON COLUMN credentials.project_id IS 'Project ID when scope is project';
COMMENT ON COLUMN credentials.owner_user_id IS 'Owner user ID when scope is personal';

-- ============================================================================
-- 2. OAuth2 Providers table (preset + custom providers)
-- ============================================================================

CREATE TABLE IF NOT EXISTS oauth2_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Identification
    slug VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    icon_url TEXT,

    -- OAuth2 Endpoints
    authorization_url TEXT NOT NULL,
    token_url TEXT NOT NULL,
    revoke_url TEXT,
    userinfo_url TEXT,

    -- Configuration
    pkce_required BOOLEAN DEFAULT false,
    default_scopes TEXT[] DEFAULT '{}',

    -- Metadata
    documentation_url TEXT,
    is_preset BOOLEAN DEFAULT false,

    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oauth2_providers_slug ON oauth2_providers(slug);
CREATE INDEX IF NOT EXISTS idx_oauth2_providers_preset ON oauth2_providers(is_preset);

COMMENT ON TABLE oauth2_providers IS 'OAuth2 provider configurations (preset and custom)';
COMMENT ON COLUMN oauth2_providers.slug IS 'Unique identifier for the provider (e.g., google, github)';
COMMENT ON COLUMN oauth2_providers.is_preset IS 'True for system-defined providers, false for custom';

-- ============================================================================
-- 3. OAuth2 Apps table (tenant-specific client configurations)
-- ============================================================================

CREATE TABLE IF NOT EXISTS oauth2_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES oauth2_providers(id) ON DELETE CASCADE,

    -- Client credentials (encrypted)
    encrypted_client_id BYTEA NOT NULL,
    encrypted_client_secret BYTEA NOT NULL,
    client_id_nonce BYTEA NOT NULL,
    client_secret_nonce BYTEA NOT NULL,

    -- Customization
    custom_scopes TEXT[],
    redirect_uri TEXT,

    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'disabled')),

    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    UNIQUE(tenant_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_oauth2_apps_tenant ON oauth2_apps(tenant_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_apps_provider ON oauth2_apps(provider_id);

COMMENT ON TABLE oauth2_apps IS 'Tenant-specific OAuth2 application configurations';
COMMENT ON COLUMN oauth2_apps.encrypted_client_id IS 'AES-256-GCM encrypted OAuth2 client ID';
COMMENT ON COLUMN oauth2_apps.encrypted_client_secret IS 'AES-256-GCM encrypted OAuth2 client secret';

-- ============================================================================
-- 4. OAuth2 Connections table (individual OAuth2 tokens)
-- ============================================================================

CREATE TABLE IF NOT EXISTS oauth2_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Relationships
    credential_id UUID NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    oauth2_app_id UUID NOT NULL REFERENCES oauth2_apps(id) ON DELETE CASCADE,

    -- Tokens (encrypted)
    encrypted_access_token BYTEA,
    encrypted_refresh_token BYTEA,
    access_token_nonce BYTEA,
    refresh_token_nonce BYTEA,
    token_type VARCHAR(50) DEFAULT 'Bearer',

    -- Expiration
    access_token_expires_at TIMESTAMPTZ,
    refresh_token_expires_at TIMESTAMPTZ,

    -- OAuth2 flow state (temporary)
    state VARCHAR(255),
    code_verifier TEXT,

    -- Account info from userinfo endpoint
    account_id TEXT,
    account_email TEXT,
    account_name TEXT,
    raw_userinfo JSONB,

    -- Status tracking
    status VARCHAR(20) DEFAULT 'pending' CHECK (
        status IN ('pending', 'connected', 'expired', 'revoked', 'error')
    ),
    last_refresh_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    error_message TEXT,

    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oauth2_connections_credential ON oauth2_connections(credential_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_connections_app ON oauth2_connections(oauth2_app_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_connections_status ON oauth2_connections(status);
CREATE INDEX IF NOT EXISTS idx_oauth2_connections_state ON oauth2_connections(state) WHERE state IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_oauth2_connections_expires ON oauth2_connections(access_token_expires_at)
    WHERE status = 'connected';

COMMENT ON TABLE oauth2_connections IS 'Individual OAuth2 token connections linked to credentials';
COMMENT ON COLUMN oauth2_connections.state IS 'CSRF protection state for OAuth2 authorization flow';
COMMENT ON COLUMN oauth2_connections.code_verifier IS 'PKCE code verifier for OAuth2 authorization flow';

-- ============================================================================
-- 5. Credential Shares table (sharing credentials between users/projects)
-- ============================================================================

CREATE TABLE IF NOT EXISTS credential_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,

    -- Share target (one of these must be set)
    shared_with_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    shared_with_project_id UUID REFERENCES projects(id) ON DELETE CASCADE,

    -- Permission level
    permission VARCHAR(20) NOT NULL DEFAULT 'use' CHECK (
        permission IN ('use', 'edit', 'admin')
    ),

    -- Metadata
    shared_by_user_id UUID NOT NULL REFERENCES users(id),
    note TEXT,

    created_at TIMESTAMPTZ DEFAULT now(),
    expires_at TIMESTAMPTZ,

    -- Ensure exactly one target is set
    CONSTRAINT share_target_check CHECK (
        (shared_with_user_id IS NOT NULL AND shared_with_project_id IS NULL) OR
        (shared_with_user_id IS NULL AND shared_with_project_id IS NOT NULL)
    ),

    -- Prevent duplicate shares
    UNIQUE(credential_id, shared_with_user_id),
    UNIQUE(credential_id, shared_with_project_id)
);

CREATE INDEX IF NOT EXISTS idx_credential_shares_credential ON credential_shares(credential_id);
CREATE INDEX IF NOT EXISTS idx_credential_shares_user ON credential_shares(shared_with_user_id)
    WHERE shared_with_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_credential_shares_project ON credential_shares(shared_with_project_id)
    WHERE shared_with_project_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_credential_shares_expires ON credential_shares(expires_at)
    WHERE expires_at IS NOT NULL;

COMMENT ON TABLE credential_shares IS 'Credential sharing between users and projects';
COMMENT ON COLUMN credential_shares.permission IS 'use: can only use, edit: can update values, admin: can delete and re-share';

-- ============================================================================
-- 6. Insert preset OAuth2 providers
-- ============================================================================

INSERT INTO oauth2_providers (slug, name, authorization_url, token_url, revoke_url, userinfo_url, pkce_required, default_scopes, is_preset, icon_url, documentation_url)
VALUES
    ('google', 'Google',
     'https://accounts.google.com/o/oauth2/v2/auth',
     'https://oauth2.googleapis.com/token',
     'https://oauth2.googleapis.com/revoke',
     'https://www.googleapis.com/oauth2/v3/userinfo',
     true,
     ARRAY['openid', 'email', 'profile'],
     true,
     'https://www.google.com/favicon.ico',
     'https://developers.google.com/identity/protocols/oauth2'),

    ('github', 'GitHub',
     'https://github.com/login/oauth/authorize',
     'https://github.com/login/oauth/access_token',
     NULL,
     'https://api.github.com/user',
     false,
     ARRAY['repo', 'user:email'],
     true,
     'https://github.githubassets.com/favicons/favicon.svg',
     'https://docs.github.com/en/developers/apps/building-oauth-apps'),

    ('slack', 'Slack',
     'https://slack.com/oauth/v2/authorize',
     'https://slack.com/api/oauth.v2.access',
     'https://slack.com/api/auth.revoke',
     NULL,
     false,
     ARRAY['chat:write', 'channels:read'],
     true,
     'https://a.slack-edge.com/80588/marketing/img/icons/icon_slack_hash_colored.png',
     'https://api.slack.com/authentication/oauth-v2'),

    ('notion', 'Notion',
     'https://api.notion.com/v1/oauth/authorize',
     'https://api.notion.com/v1/oauth/token',
     NULL,
     NULL,
     false,
     '{}',
     true,
     'https://www.notion.so/images/favicon.ico',
     'https://developers.notion.com/docs/authorization'),

    ('linear', 'Linear',
     'https://linear.app/oauth/authorize',
     'https://api.linear.app/oauth/token',
     'https://api.linear.app/oauth/revoke',
     NULL,
     false,
     ARRAY['read', 'write'],
     true,
     'https://linear.app/favicon.ico',
     'https://developers.linear.app/docs/oauth/authentication'),

    ('microsoft', 'Microsoft',
     'https://login.microsoftonline.com/common/oauth2/v2.0/authorize',
     'https://login.microsoftonline.com/common/oauth2/v2.0/token',
     NULL,
     'https://graph.microsoft.com/v1.0/me',
     true,
     ARRAY['openid', 'email', 'profile', 'offline_access'],
     true,
     'https://www.microsoft.com/favicon.ico',
     'https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-auth-code-flow'),

    ('discord', 'Discord',
     'https://discord.com/api/oauth2/authorize',
     'https://discord.com/api/oauth2/token',
     'https://discord.com/api/oauth2/token/revoke',
     'https://discord.com/api/users/@me',
     false,
     ARRAY['identify', 'email'],
     true,
     'https://discord.com/assets/favicon.ico',
     'https://discord.com/developers/docs/topics/oauth2'),

    ('atlassian', 'Atlassian (Jira/Confluence)',
     'https://auth.atlassian.com/authorize',
     'https://auth.atlassian.com/oauth/token',
     NULL,
     'https://api.atlassian.com/me',
     true,
     ARRAY['read:me', 'read:jira-work', 'write:jira-work'],
     true,
     'https://www.atlassian.com/favicon.ico',
     'https://developer.atlassian.com/cloud/jira/platform/oauth-2-3lo-apps/')
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    authorization_url = EXCLUDED.authorization_url,
    token_url = EXCLUDED.token_url,
    revoke_url = EXCLUDED.revoke_url,
    userinfo_url = EXCLUDED.userinfo_url,
    pkce_required = EXCLUDED.pkce_required,
    default_scopes = EXCLUDED.default_scopes,
    icon_url = EXCLUDED.icon_url,
    documentation_url = EXCLUDED.documentation_url,
    updated_at = now();
