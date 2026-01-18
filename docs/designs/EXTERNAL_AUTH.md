# 外部サービス認証設計

> **Status**: 📋 設計中
> **Author**: Claude
> **Created**: 2025-01-18

---

## 概要

外部サービス連携ブロックで使用する認証情報の管理機能を設計する。
n8nと同等の認証方式をサポートし、組織/プロジェクト/個人の3階層スコープで管理可能とする。

### 設計原則

1. **明示的バインディング必須**: 同名の認証情報が複数スコープに存在しても暗黙的な解決は行わない
2. **最小権限の原則**: 共有時は「使用のみ」をデフォルトとし、詳細の閲覧を制限
3. **既存設計との整合性**: 環境変数と同じ3階層スコープを採用

---

## 認証タイプ

### Phase 1: 必須（MVP）

| タイプ | 説明 | 用途例 |
|--------|------|--------|
| `api_key` | APIキー（ヘッダー/クエリ） | Tavily, SendGrid |
| `bearer` | Bearer Token | GitHub, Notion |
| `basic` | Basic認証 | レガシーAPI |
| `oauth2` | OAuth2（フルフロー） | Google, Slack, GitHub Apps |

### Phase 2: 拡張

| タイプ | 説明 | 用途例 |
|--------|------|--------|
| `query_auth` | 複数クエリパラメータ認証 | 一部のレガシーAPI |
| `header_auth` | 複数ヘッダー認証 | AWS Signature等 |
| `oauth1` | OAuth 1.0a | Twitter (旧API) |
| `digest` | Digest認証 | 一部の企業システム |

---

## スコープ設計

### 階層構造

```
System Credentials (既存)
├── スコープ: システム全体
├── 管理者: オペレーター（SaaS運営者）
└── 用途: LLM API Key など共通リソース

Organization Credentials
├── スコープ: テナント全体
├── 管理者: テナント管理者
└── 用途: 組織共通のSlack/GitHub連携

Project Credentials
├── スコープ: プロジェクト内
├── 管理者: プロジェクト管理者
└── 用途: プロジェクト固有のAPI連携

Personal Credentials
├── スコープ: 個人のみ
├── 管理者: ユーザー本人
└── 用途: 個人のGitHub Token、テスト用APIキー
```

### 重要: 暗黙的解決の禁止

**同名の認証情報が複数スコープに存在する場合でも、暗黙的な優先順位による解決は行わない。**

```
❌ 禁止: 暗黙的解決
   github_token が Organization と Personal の両方に存在
   → 自動的に Personal を優先して使用

✅ 必須: 明示的バインディング
   ステップ設定で credential_id を明示的に指定
   → 指定された認証情報のみを使用
```

**理由**:
- 意図しない権限でワークフローが実行されることを防止
- 認証情報の使用箇所を明確化
- 監査・トラブルシューティングの容易化

---

## DBスキーマ設計

### credentials テーブル拡張

```sql
-- 既存テーブルへのカラム追加
ALTER TABLE credentials
ADD COLUMN scope VARCHAR(20) NOT NULL DEFAULT 'organization',
ADD COLUMN project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
ADD COLUMN owner_user_id UUID REFERENCES users(id) ON DELETE CASCADE;

-- スコープ制約
ALTER TABLE credentials ADD CONSTRAINT credentials_scope_check CHECK (
    (scope = 'organization' AND project_id IS NULL AND owner_user_id IS NULL) OR
    (scope = 'project' AND project_id IS NOT NULL AND owner_user_id IS NULL) OR
    (scope = 'personal' AND project_id IS NULL AND owner_user_id IS NOT NULL)
);

-- インデックス
CREATE INDEX idx_credentials_scope ON credentials(tenant_id, scope);
CREATE INDEX idx_credentials_project ON credentials(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_credentials_owner ON credentials(owner_user_id) WHERE owner_user_id IS NOT NULL;

COMMENT ON COLUMN credentials.scope IS 'organization: テナント全体, project: プロジェクト内, personal: 個人のみ';
```

### oauth2_providers テーブル（新規）

OAuth2プロバイダーの設定を管理。

```sql
CREATE TABLE oauth2_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 識別
    slug VARCHAR(50) NOT NULL UNIQUE,  -- e.g., "google", "github", "slack"
    name VARCHAR(100) NOT NULL,
    icon_url TEXT,

    -- エンドポイント
    authorization_url TEXT NOT NULL,
    token_url TEXT NOT NULL,
    revoke_url TEXT,
    userinfo_url TEXT,

    -- 設定
    pkce_required BOOLEAN DEFAULT false,
    default_scopes TEXT[] DEFAULT '{}',

    -- メタデータ
    documentation_url TEXT,
    is_preset BOOLEAN DEFAULT false,  -- true = システム定義, false = カスタム

    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- プリセットプロバイダー
INSERT INTO oauth2_providers (slug, name, authorization_url, token_url, revoke_url, userinfo_url, pkce_required, default_scopes, is_preset) VALUES
('google', 'Google', 'https://accounts.google.com/o/oauth2/v2/auth', 'https://oauth2.googleapis.com/token', 'https://oauth2.googleapis.com/revoke', 'https://www.googleapis.com/oauth2/v3/userinfo', true, ARRAY['openid', 'email', 'profile'], true),
('github', 'GitHub', 'https://github.com/login/oauth/authorize', 'https://github.com/login/oauth/access_token', NULL, 'https://api.github.com/user', false, ARRAY['repo', 'user:email'], true),
('slack', 'Slack', 'https://slack.com/oauth/v2/authorize', 'https://slack.com/api/oauth.v2.access', 'https://slack.com/api/auth.revoke', NULL, false, ARRAY['chat:write', 'channels:read'], true),
('notion', 'Notion', 'https://api.notion.com/v1/oauth/authorize', 'https://api.notion.com/v1/oauth/token', NULL, NULL, false, '{}', true),
('linear', 'Linear', 'https://linear.app/oauth/authorize', 'https://api.linear.app/oauth/token', 'https://api.linear.app/oauth/revoke', NULL, false, ARRAY['read', 'write'], true),
('microsoft', 'Microsoft', 'https://login.microsoftonline.com/common/oauth2/v2.0/authorize', 'https://login.microsoftonline.com/common/oauth2/v2.0/token', NULL, 'https://graph.microsoft.com/v1.0/me', true, ARRAY['openid', 'email', 'profile'], true);
```

### oauth2_apps テーブル（新規）

テナントごとのOAuth2アプリケーション設定（Client ID/Secret）。

```sql
CREATE TABLE oauth2_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES oauth2_providers(id),

    -- クライアント設定（暗号化）
    encrypted_client_id BYTEA NOT NULL,
    encrypted_client_secret BYTEA NOT NULL,
    client_id_nonce BYTEA NOT NULL,
    client_secret_nonce BYTEA NOT NULL,

    -- カスタマイズ
    custom_scopes TEXT[],  -- NULL = プロバイダーのデフォルトを使用
    redirect_uri TEXT,     -- NULL = システムデフォルト

    -- 状態
    status VARCHAR(20) DEFAULT 'active',  -- active, disabled

    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    UNIQUE(tenant_id, provider_id)
);

CREATE INDEX idx_oauth2_apps_tenant ON oauth2_apps(tenant_id);
```

### oauth2_connections テーブル（新規）

個々のOAuth2接続（トークン）を管理。

```sql
CREATE TABLE oauth2_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 関連
    credential_id UUID NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    oauth2_app_id UUID NOT NULL REFERENCES oauth2_apps(id),

    -- トークン（暗号化）
    encrypted_access_token BYTEA,
    encrypted_refresh_token BYTEA,
    access_token_nonce BYTEA,
    refresh_token_nonce BYTEA,
    token_type VARCHAR(50) DEFAULT 'Bearer',

    -- 有効期限
    access_token_expires_at TIMESTAMPTZ,
    refresh_token_expires_at TIMESTAMPTZ,

    -- OAuth2フロー用（一時的）
    state VARCHAR(255),
    code_verifier TEXT,  -- PKCE

    -- アカウント情報
    account_id TEXT,
    account_email TEXT,
    account_name TEXT,
    raw_userinfo JSONB,

    -- 状態
    status VARCHAR(20) DEFAULT 'pending',  -- pending, connected, expired, revoked, error
    last_refresh_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    error_message TEXT,

    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_oauth2_connections_credential ON oauth2_connections(credential_id);
CREATE INDEX idx_oauth2_connections_status ON oauth2_connections(status);
CREATE INDEX idx_oauth2_connections_expires ON oauth2_connections(access_token_expires_at)
    WHERE status = 'connected';
```

### credential_shares テーブル（新規）

認証情報の共有を管理。

```sql
CREATE TABLE credential_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,

    -- 共有先（どちらか一方）
    shared_with_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    shared_with_project_id UUID REFERENCES projects(id) ON DELETE CASCADE,

    -- 権限
    permission VARCHAR(20) NOT NULL DEFAULT 'use',  -- use, edit, admin

    -- メタデータ
    shared_by_user_id UUID NOT NULL REFERENCES users(id),
    note TEXT,  -- 共有時のメモ

    created_at TIMESTAMPTZ DEFAULT now(),
    expires_at TIMESTAMPTZ,  -- NULL = 無期限

    CONSTRAINT share_target_check CHECK (
        (shared_with_user_id IS NOT NULL AND shared_with_project_id IS NULL) OR
        (shared_with_user_id IS NULL AND shared_with_project_id IS NOT NULL)
    ),

    -- 同じ認証情報を同じ相手に二重共有しない
    UNIQUE(credential_id, shared_with_user_id),
    UNIQUE(credential_id, shared_with_project_id)
);

CREATE INDEX idx_credential_shares_credential ON credential_shares(credential_id);
CREATE INDEX idx_credential_shares_user ON credential_shares(shared_with_user_id)
    WHERE shared_with_user_id IS NOT NULL;
CREATE INDEX idx_credential_shares_project ON credential_shares(shared_with_project_id)
    WHERE shared_with_project_id IS NOT NULL;

COMMENT ON COLUMN credential_shares.permission IS 'use: 使用のみ（詳細非表示）, edit: 編集可能, admin: 削除・再共有可能';
```

---

## ドメインモデル

### Credential（拡張）

```go
// backend/internal/domain/credential.go

type CredentialScope string

const (
    CredentialScopeOrganization CredentialScope = "organization"
    CredentialScopeProject      CredentialScope = "project"
    CredentialScopePersonal     CredentialScope = "personal"
)

type Credential struct {
    ID             uuid.UUID
    TenantID       uuid.UUID
    Name           string
    Description    string
    CredentialType CredentialType
    Scope          CredentialScope  // 新規
    ProjectID      *uuid.UUID       // 新規: scope=project の場合
    OwnerUserID    *uuid.UUID       // 新規: scope=personal の場合

    // 暗号化データ（既存）
    EncryptedData  []byte
    EncryptedDEK   []byte
    DataNonce      []byte
    DEKNonce       []byte

    Metadata       CredentialMetadata
    ExpiresAt      *time.Time
    Status         CredentialStatus

    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// CredentialType の拡張
type CredentialType string

const (
    CredentialTypeAPIKey     CredentialType = "api_key"
    CredentialTypeBearer     CredentialType = "bearer"
    CredentialTypeBasic      CredentialType = "basic"
    CredentialTypeOAuth2     CredentialType = "oauth2"
    CredentialTypeCustom     CredentialType = "custom"

    // Phase 2
    CredentialTypeQueryAuth  CredentialType = "query_auth"
    CredentialTypeHeaderAuth CredentialType = "header_auth"
)
```

### OAuth2Provider

```go
// backend/internal/domain/oauth2.go

type OAuth2Provider struct {
    ID               uuid.UUID
    Slug             string
    Name             string
    IconURL          string

    AuthorizationURL string
    TokenURL         string
    RevokeURL        string
    UserinfoURL      string

    PKCERequired     bool
    DefaultScopes    []string

    DocumentationURL string
    IsPreset         bool

    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type OAuth2App struct {
    ID                    uuid.UUID
    TenantID              uuid.UUID
    ProviderID            uuid.UUID

    // 暗号化済み
    EncryptedClientID     []byte
    EncryptedClientSecret []byte
    ClientIDNonce         []byte
    ClientSecretNonce     []byte

    CustomScopes          []string
    RedirectURI           string
    Status                string

    CreatedAt             time.Time
    UpdatedAt             time.Time

    // 関連
    Provider              *OAuth2Provider
}

type OAuth2Connection struct {
    ID                     uuid.UUID
    CredentialID           uuid.UUID
    OAuth2AppID            uuid.UUID

    // トークン（暗号化済み）
    EncryptedAccessToken   []byte
    EncryptedRefreshToken  []byte
    AccessTokenNonce       []byte
    RefreshTokenNonce      []byte
    TokenType              string

    AccessTokenExpiresAt   *time.Time
    RefreshTokenExpiresAt  *time.Time

    // フロー用
    State                  string
    CodeVerifier           string

    // アカウント情報
    AccountID              string
    AccountEmail           string
    AccountName            string
    RawUserinfo            json.RawMessage

    Status                 OAuth2ConnectionStatus
    LastRefreshAt          *time.Time
    LastUsedAt             *time.Time
    ErrorMessage           string

    CreatedAt              time.Time
    UpdatedAt              time.Time
}

type OAuth2ConnectionStatus string

const (
    OAuth2StatusPending   OAuth2ConnectionStatus = "pending"
    OAuth2StatusConnected OAuth2ConnectionStatus = "connected"
    OAuth2StatusExpired   OAuth2ConnectionStatus = "expired"
    OAuth2StatusRevoked   OAuth2ConnectionStatus = "revoked"
    OAuth2StatusError     OAuth2ConnectionStatus = "error"
)
```

### CredentialShare

```go
// backend/internal/domain/credential_share.go

type SharePermission string

const (
    SharePermissionUse   SharePermission = "use"   // 使用のみ（詳細非表示）
    SharePermissionEdit  SharePermission = "edit"  // 編集可能
    SharePermissionAdmin SharePermission = "admin" // 削除・再共有可能
)

type CredentialShare struct {
    ID                  uuid.UUID
    CredentialID        uuid.UUID

    SharedWithUserID    *uuid.UUID
    SharedWithProjectID *uuid.UUID

    Permission          SharePermission
    SharedByUserID      uuid.UUID
    Note                string

    CreatedAt           time.Time
    ExpiresAt           *time.Time
}
```

---

## API設計

### Credentials API

```yaml
# 認証情報一覧（アクセス可能な全スコープ）
GET /api/v1/credentials
  Query:
    scope: organization | project | personal | shared  # フィルタ（省略時は全て）
    project_id: UUID  # scope=project の場合必須
    type: api_key | bearer | oauth2 | ...  # タイプフィルタ
  Response:
    credentials:
      - id, name, type, scope, status, created_at
      - masked_preview: "••••••abc123"  # 末尾のみ表示
      - oauth2_account: { email, name }  # OAuth2の場合
      - shared_info: { permission, shared_by }  # 共有された認証情報の場合

# 認証情報作成
POST /api/v1/credentials
  Body:
    name: string
    description?: string
    credential_type: api_key | bearer | basic | oauth2 | custom
    scope: organization | project | personal
    project_id?: UUID  # scope=project の場合必須
    data:
      # api_key / bearer
      api_key?: string
      header_name?: string  # デフォルト: Authorization
      header_prefix?: string  # デフォルト: Bearer (bearerの場合)

      # basic
      username?: string
      password?: string

      # custom
      custom?: Record<string, any>
  Response:
    credential: { id, name, type, scope, status, created_at }

# 認証情報詳細（権限チェック: 所有者 or edit以上の共有）
GET /api/v1/credentials/:id
  Response:
    credential: { ... }
    data: { ... }  # use権限の場合は含まれない
    shares: [{ user, project, permission }]

# 認証情報更新
PUT /api/v1/credentials/:id
  Body:
    name?: string
    description?: string
    data?: { ... }

# 認証情報削除
DELETE /api/v1/credentials/:id

# 接続テスト
POST /api/v1/credentials/:id/test
  Response:
    success: boolean
    message?: string
    latency_ms?: number
```

### OAuth2 API

```yaml
# OAuth2プロバイダー一覧
GET /api/v1/oauth2/providers
  Response:
    providers:
      - id, slug, name, icon_url, pkce_required
      - app_configured: boolean  # テナントでClient設定済みか

# OAuth2アプリ設定（テナント管理者のみ）
POST /api/v1/oauth2/apps
  Body:
    provider_id: UUID
    client_id: string
    client_secret: string
    custom_scopes?: string[]
  Response:
    app: { id, provider, status }

PUT /api/v1/oauth2/apps/:id
DELETE /api/v1/oauth2/apps/:id

# OAuth2認可フロー開始
POST /api/v1/oauth2/authorize/start
  Body:
    provider_slug: string  # e.g., "google", "github"
    scope: organization | project | personal
    project_id?: UUID
    name: string  # 認証情報名
    scopes?: string[]  # 追加スコープ
  Response:
    authorization_url: string  # ユーザーをここにリダイレクト
    state: string

# OAuth2コールバック（リダイレクト先）
GET /api/v1/oauth2/callback
  Query:
    code: string
    state: string
    error?: string
  Response:
    redirect_to: "/credentials?connected=true"  # フロントエンドにリダイレクト

# OAuth2接続詳細
GET /api/v1/oauth2/connections/:id
  Response:
    connection:
      id, status, account_email, account_name
      access_token_expires_at, last_refresh_at

# 手動トークンリフレッシュ
POST /api/v1/oauth2/connections/:id/refresh

# 接続解除（トークン無効化）
DELETE /api/v1/oauth2/connections/:id
```

### Credential Sharing API

```yaml
# 共有作成
POST /api/v1/credentials/:id/shares
  Body:
    user_id?: UUID
    project_id?: UUID
    permission: use | edit | admin
    note?: string
    expires_at?: timestamp
  Response:
    share: { id, permission, shared_with }

# 共有一覧
GET /api/v1/credentials/:id/shares
  Response:
    shares:
      - id, user/project, permission, shared_by, created_at

# 共有更新
PUT /api/v1/credentials/:id/shares/:share_id
  Body:
    permission?: use | edit | admin
    expires_at?: timestamp

# 共有削除
DELETE /api/v1/credentials/:id/shares/:share_id
```

### Step Credential Binding

```yaml
# ステップの認証情報バインディング更新
PUT /api/v1/workflows/:wf_id/steps/:step_id/credential-bindings
  Body:
    bindings:
      - required_credential_name: string  # BlockDefinition.RequiredCredentials の name
        credential_id: UUID               # 明示的に指定
  Response:
    bindings: [{ name, credential_id, credential_name, type }]

# 利用可能な認証情報一覧（バインディング用）
GET /api/v1/credentials/available
  Query:
    project_id: UUID
    credential_type?: string  # フィルタ
    required_scope?: system | tenant  # BlockDefinition.RequiredCredentials の scope
  Response:
    credentials:
      # required_scope=system の場合: システム認証情報のみ
      # required_scope=tenant の場合: 以下を結合
      #   - Organization credentials
      #   - Project credentials (指定されたproject_id)
      #   - Personal credentials (現在のユーザー)
      #   - Shared credentials (現在のユーザー/プロジェクトに共有されたもの)
      - id, name, type, scope, source  # source: own | shared
```

---

## 認証情報解決フロー

### 実行時の解決（明示的バインディング）

```go
// backend/internal/usecase/credential_resolver.go

type CredentialResolver struct {
    credentialRepo CredentialRepository
    oauth2Repo     OAuth2Repository
    encryptor      Encryptor
}

// ResolveForStep: ステップ実行時に認証情報を解決
func (r *CredentialResolver) ResolveForStep(
    ctx context.Context,
    block *domain.BlockDefinition,
    step *domain.Step,
    tenantID uuid.UUID,
    userID uuid.UUID,
) (*ResolvedCredentials, error) {
    result := &ResolvedCredentials{
        Secrets: make(map[string]string),
    }

    // 1. BlockDefinition から RequiredCredentials を取得
    required, err := block.ParseRequiredCredentials()
    if err != nil {
        return nil, fmt.Errorf("parse required credentials: %w", err)
    }

    // 2. Step から credential_bindings を取得
    bindings, err := step.ParseCredentialBindings()
    if err != nil {
        return nil, fmt.Errorf("parse credential bindings: %w", err)
    }

    // 3. 各必須認証情報を解決
    for _, req := range required {
        // 明示的バインディングを確認
        credID, exists := bindings[req.Name]
        if !exists {
            if req.Required {
                return nil, fmt.Errorf("credential binding not found: %s", req.Name)
            }
            continue
        }

        // 認証情報を取得（アクセス権チェック込み）
        cred, err := r.getCredentialWithAccessCheck(ctx, credID, tenantID, userID, step.ProjectID)
        if err != nil {
            return nil, fmt.Errorf("get credential %s: %w", req.Name, err)
        }

        // OAuth2の場合はトークンをリフレッシュ
        if cred.CredentialType == domain.CredentialTypeOAuth2 {
            token, err := r.getValidOAuth2Token(ctx, cred.ID)
            if err != nil {
                return nil, fmt.Errorf("get oauth2 token: %w", err)
            }
            result.Secrets[req.Name] = token
        } else {
            // 復号化
            data, err := r.decrypt(cred)
            if err != nil {
                return nil, fmt.Errorf("decrypt credential: %w", err)
            }
            result.Secrets[req.Name] = data.GetSecretValue()
        }
    }

    return result, nil
}

// getCredentialWithAccessCheck: アクセス権をチェックして認証情報を取得
func (r *CredentialResolver) getCredentialWithAccessCheck(
    ctx context.Context,
    credID uuid.UUID,
    tenantID uuid.UUID,
    userID uuid.UUID,
    projectID *uuid.UUID,
) (*domain.Credential, error) {
    cred, err := r.credentialRepo.GetByID(ctx, credID)
    if err != nil {
        return nil, err
    }

    // テナントチェック
    if cred.TenantID != tenantID {
        return nil, ErrCredentialNotFound
    }

    // スコープ別アクセスチェック
    switch cred.Scope {
    case domain.CredentialScopeOrganization:
        // 同一テナントなら OK
        return cred, nil

    case domain.CredentialScopeProject:
        // 同一プロジェクト、または共有されている場合 OK
        if projectID != nil && cred.ProjectID != nil && *cred.ProjectID == *projectID {
            return cred, nil
        }
        // 共有チェック
        if r.hasShareAccess(ctx, credID, userID, projectID) {
            return cred, nil
        }
        return nil, ErrCredentialAccessDenied

    case domain.CredentialScopePersonal:
        // 所有者、または共有されている場合 OK
        if cred.OwnerUserID != nil && *cred.OwnerUserID == userID {
            return cred, nil
        }
        // 共有チェック
        if r.hasShareAccess(ctx, credID, userID, projectID) {
            return cred, nil
        }
        return nil, ErrCredentialAccessDenied
    }

    return nil, ErrCredentialAccessDenied
}
```

### OAuth2トークン自動リフレッシュ

```go
// getValidOAuth2Token: 有効なアクセストークンを取得（必要に応じてリフレッシュ）
func (r *CredentialResolver) getValidOAuth2Token(ctx context.Context, credentialID uuid.UUID) (string, error) {
    conn, err := r.oauth2Repo.GetConnectionByCredentialID(ctx, credentialID)
    if err != nil {
        return "", err
    }

    // 有効期限チェック（5分のバッファ）
    if conn.AccessTokenExpiresAt != nil &&
       conn.AccessTokenExpiresAt.Before(time.Now().Add(5 * time.Minute)) {
        // リフレッシュが必要
        if err := r.refreshOAuth2Token(ctx, conn); err != nil {
            // リフレッシュ失敗 → ステータスを更新
            conn.Status = domain.OAuth2StatusExpired
            conn.ErrorMessage = err.Error()
            r.oauth2Repo.Update(ctx, conn)
            return "", fmt.Errorf("token refresh failed: %w", err)
        }
        // 再取得
        conn, _ = r.oauth2Repo.GetConnectionByCredentialID(ctx, credentialID)
    }

    // トークン復号化
    token, err := r.encryptor.Decrypt(conn.EncryptedAccessToken, conn.AccessTokenNonce)
    if err != nil {
        return "", err
    }

    // 最終使用時刻を更新
    conn.LastUsedAt = ptr(time.Now())
    r.oauth2Repo.Update(ctx, conn)

    return string(token), nil
}
```

---

## UI/UXフロー

### 認証情報管理画面

```
┌─────────────────────────────────────────────────────────────────┐
│  Credentials                                           [+ Add]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [All] [Organization] [Project] [Personal] [Shared with me]    │
│  ─────────────────────────────────────────────────────────────  │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 🏢 Slack Workspace                                       │   │
│  │    OAuth2 • Connected as team@company.com               │   │
│  │    Organization • Created 3 days ago                    │   │
│  │                                    [Test] [Share] [Edit]│   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 👤 My GitHub                                             │   │
│  │    OAuth2 • Connected as @username                      │   │
│  │    Personal • Created 1 week ago                        │   │
│  │                                    [Test] [Share] [Edit]│   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 🔗 Team Notion                         Shared by @alice │   │
│  │    OAuth2 • Use only                                    │   │
│  │    ⚠️ You can use this credential but cannot view details│   │
│  │                                              [Test]     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### OAuth2接続フロー

```
┌─────────────────────────────────────────────────────────────────┐
│  Connect to Google                                      [Close] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Step 1: Name your credential                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Google Sheets - Marketing                               │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Step 2: Select scope                                           │
│  ○ Organization - Available to everyone in this workspace       │
│  ● Project - Available to members of selected project           │
│     └─ [Marketing Automation ▼]                                │
│  ○ Personal - Only available to you                             │
│                                                                 │
│  Step 3: Select permissions                                     │
│  ☑ Google Sheets (read & write)                                │
│  ☑ Google Drive (read only)                                    │
│  ☐ Google Calendar                                             │
│                                                                 │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              [Connect with Google]                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  By connecting, you agree to share the selected permissions    │
│  with this application.                                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### ステップでの認証情報バインディング

```
┌─────────────────────────────────────────────────────────────────┐
│  Step Configuration: Send Slack Message                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Required Credentials ──────────────────────────────────┐   │
│  │                                                         │   │
│  │  Slack Token *                                          │   │
│  │  ┌─────────────────────────────────────────────────┐   │   │
│  │  │ 🏢 Slack Workspace (Organization)          ▼    │   │   │
│  │  ├─────────────────────────────────────────────────┤   │   │
│  │  │ 🏢 Slack Workspace (Organization)               │   │   │
│  │  │ 📁 Project Slack (Project)                      │   │   │
│  │  │ 👤 My Slack (Personal)                          │   │   │
│  │  │ 🔗 Team Slack (Shared by @bob)                  │   │   │
│  │  └─────────────────────────────────────────────────┘   │   │
│  │                                                         │   │
│  │  ⚠️ You must explicitly select which credential to use  │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─ Configuration ─────────────────────────────────────────┐   │
│  │                                                         │   │
│  │  Channel:  #general                                     │   │
│  │  Message:  {{input.summary}}                            │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 実装フェーズ

### Phase 1: 基盤（2週間目安）

1. DBマイグレーション
   - credentials テーブル拡張（scope, project_id, owner_user_id）
   - oauth2_providers, oauth2_apps, oauth2_connections テーブル
   - credential_shares テーブル

2. ドメイン層
   - Credential 拡張
   - OAuth2Provider, OAuth2App, OAuth2Connection
   - CredentialShare

3. リポジトリ層
   - CredentialRepository 拡張
   - OAuth2Repository
   - CredentialShareRepository

### Phase 2: OAuth2フロー（2週間目安）

1. OAuth2Service
   - StartAuthorization
   - HandleCallback
   - RefreshToken
   - RevokeConnection

2. OAuth2Handler
   - API エンドポイント実装

3. プリセットプロバイダー
   - Google, GitHub, Slack, Notion, Linear, Microsoft

### Phase 3: 共有機能（1週間目安）

1. CredentialShareService
   - Share, Unshare, UpdatePermission

2. アクセスチェック統合
   - CredentialResolver への統合

### Phase 4: フロントエンド（2週間目安）

1. 認証情報管理画面
2. OAuth2接続フロー
3. ステップでのバインディングUI
4. 共有UI

---

## セキュリティ考慮事項

1. **PKCE必須化**: Authorization Code Flow では PKCE を強く推奨
2. **State検証**: CSRF攻撃防止のため state パラメータを必須
3. **トークン暗号化**: すべてのトークンをAES-256-GCMで暗号化
4. **スコープ最小化**: 必要最小限のスコープのみ要求
5. **共有時の詳細非表示**: `use`権限では認証情報の詳細を表示しない
6. **監査ログ**: 接続/切断/共有/使用のログを記録
7. **有効期限管理**: 共有に有効期限を設定可能

---

## 関連ドキュメント

- [INTEGRATIONS.md](../INTEGRATIONS.md) - 外部サービス連携
- [UNIFIED_BLOCK_MODEL.md](./UNIFIED_BLOCK_MODEL.md) - ブロックモデル
- [BACKEND.md](../BACKEND.md) - バックエンド実装
