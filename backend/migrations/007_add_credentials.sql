-- Migration: Add credentials table for secure storage of API keys, tokens, etc.
-- This table stores encrypted authentication credentials for external services

CREATE TABLE IF NOT EXISTS credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT,

    -- Credential type (oauth2, api_key, basic, bearer, custom)
    credential_type VARCHAR(50) NOT NULL,

    -- Encrypted credential data (AES-256-GCM)
    -- Contains the actual secrets (API keys, tokens, passwords)
    encrypted_data BYTEA NOT NULL,

    -- Encrypted Data Encryption Key (envelope encryption)
    -- The DEK is encrypted with the master KEK from environment
    encrypted_dek BYTEA NOT NULL,

    -- Nonce/IV for data encryption (12 bytes)
    data_nonce BYTEA NOT NULL,

    -- Nonce/IV for DEK encryption (12 bytes)
    dek_nonce BYTEA NOT NULL,

    -- Non-sensitive metadata (can be displayed in UI)
    metadata JSONB NOT NULL DEFAULT '{}',

    -- OAuth2-specific: token expiration
    expires_at TIMESTAMPTZ,

    -- Credential status
    status VARCHAR(20) NOT NULL DEFAULT 'active',

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Unique name per tenant
    CONSTRAINT unique_credential_name UNIQUE (tenant_id, name),

    -- Valid credential types
    CONSTRAINT valid_credential_type CHECK (credential_type IN ('oauth2', 'api_key', 'basic', 'bearer', 'custom')),

    -- Valid status values
    CONSTRAINT valid_credential_status CHECK (status IN ('active', 'expired', 'revoked', 'error'))
);

-- Indexes
CREATE INDEX idx_credentials_tenant ON credentials(tenant_id);
CREATE INDEX idx_credentials_type ON credentials(credential_type);
CREATE INDEX idx_credentials_status ON credentials(status);
CREATE INDEX idx_credentials_expires ON credentials(expires_at) WHERE expires_at IS NOT NULL;

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_credentials_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_credentials_updated_at
    BEFORE UPDATE ON credentials
    FOR EACH ROW
    EXECUTE FUNCTION update_credentials_updated_at();

-- Add credential_id to block_definitions for blocks that require authentication
ALTER TABLE block_definitions
ADD COLUMN IF NOT EXISTS default_credential_id UUID REFERENCES credentials(id) ON DELETE SET NULL;

-- Comment for documentation
COMMENT ON TABLE credentials IS 'Stores encrypted API credentials for external service authentication';
COMMENT ON COLUMN credentials.encrypted_data IS 'AES-256-GCM encrypted credential data (secrets)';
COMMENT ON COLUMN credentials.encrypted_dek IS 'Encrypted Data Encryption Key (envelope encryption)';
COMMENT ON COLUMN credentials.data_nonce IS '12-byte nonce/IV for data encryption';
COMMENT ON COLUMN credentials.dek_nonce IS '12-byte nonce/IV for DEK encryption';
COMMENT ON COLUMN credentials.metadata IS 'Non-sensitive metadata (e.g., service name, account info)';
