/**
 * OAuth2 Types
 * 外部サービス認証機能の型定義
 */

// OAuth2 Provider (Preset or custom OAuth2 provider)
export interface OAuth2Provider {
  id: string
  slug: string
  name: string
  icon_url?: string
  authorization_url: string
  token_url: string
  revoke_url?: string
  userinfo_url?: string
  pkce_required: boolean
  default_scopes: string[]
  available_scopes?: string[]
  description?: string
  authorization_type?: string
  documentation_url?: string
  is_preset: boolean
  created_at: string
  updated_at: string
}

// OAuth2 App (Tenant's OAuth2 client configuration)
export interface OAuth2App {
  id: string
  tenant_id: string
  provider_id: string
  provider_slug: string
  provider?: OAuth2Provider
  client_id: string
  scopes?: string[]
  custom_scopes?: string[]
  redirect_uri?: string
  status: 'active' | 'disabled'
  created_at: string
  updated_at: string
}

// OAuth2 Connection Status
export type OAuth2ConnectionStatus = 'pending' | 'connected' | 'expired' | 'revoked' | 'error'

// OAuth2 Connection (Individual token/connection)
export interface OAuth2Connection {
  id: string
  credential_id: string
  oauth2_app_id: string
  token_type: string
  access_token_expires_at?: string
  refresh_token_expires_at?: string
  account_id?: string
  account_email?: string
  account_name?: string
  status: OAuth2ConnectionStatus
  last_refresh_at?: string
  last_used_at?: string
  error_message?: string
  created_at: string
  updated_at: string
}

// Credential Scope (where credentials can be used)
export type CredentialScope = 'organization' | 'project' | 'personal'

// Share Permission (level of access when sharing)
export type SharePermission = 'use' | 'edit' | 'admin'

// Credential Share (sharing a credential with user or project)
export interface CredentialShare {
  id: string
  credential_id: string
  target_user_id?: string
  target_user_email?: string
  target_project_id?: string
  target_project_name?: string
  shared_with_user_id?: string
  shared_with_project_id?: string
  permission: SharePermission
  shared_by_user_id: string
  note?: string
  expires_at?: string
  created_at: string
}

// API Request Types

export interface CreateOAuth2AppRequest {
  provider_slug: string
  client_id: string
  client_secret: string
  scopes?: string[]
  custom_scopes?: string[]
  custom_authorization_url?: string
  custom_token_url?: string
}

export interface StartAuthorizationRequest {
  provider_slug: string
  name: string
  scope: CredentialScope
  project_id?: string
  scopes?: string[]
}

export interface StartAuthorizationResponse {
  authorization_url: string
  state: string
  credential_id: string
}

export interface ShareWithUserRequest {
  target_user_email: string
  shared_with_user_id?: string
  permission: SharePermission
  note?: string
  expires_at?: string
}

export interface ShareWithProjectRequest {
  target_project_id: string
  shared_with_project_id?: string
  permission: SharePermission
  note?: string
  expires_at?: string
}

export interface UpdateShareRequest {
  permission?: SharePermission
  note?: string
  expires_at?: string
}

// Provider Response (with app configuration status)
export interface OAuth2ProviderWithStatus extends OAuth2Provider {
  app_configured: boolean
}

// Required Credential (from BlockDefinition)
export interface RequiredCredential {
  name: string
  type: 'api_key' | 'oauth2' | 'bearer' | 'basic' | 'custom'
  scope: 'system' | 'tenant'
  description: string
  required: boolean
}

// Credential Binding (mapping in Step)
export interface CredentialBinding {
  name: string
  credential_id: string
}
