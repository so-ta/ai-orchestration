# Phase 8: Variables System 実装計画

## 概要

**目的**: ワークフロー全体で使用可能な変数システムを提供し、設定値・シークレット・実行時データを一元管理する。

**ユースケース例**:
- APIキーをシークレット変数として安全に管理
- ワークフロー固有の設定値（タイムアウト、リトライ回数）を変数化
- 実行時に動的に値を変更可能にする
- テナント共通の設定をシステム変数として管理

---

## 機能要件

### 1. 変数スコープ

| スコープ | 説明 | 例 | アクセス |
|----------|------|-----|----------|
| `system` | システム全体で共有 | デフォルトLLMモデル | 全ワークフロー |
| `tenant` | テナント内で共有 | テナント固有のAPIキー | テナント内全WF |
| `workflow` | ワークフロー固有 | WF固有の設定値 | 単一WF |
| `run` | 実行時のみ有効 | 入力パラメータ | 単一実行 |

### 2. 変数タイプ

| タイプ | 説明 | 例 |
|--------|------|-----|
| `string` | 文字列 | `"gpt-4o"` |
| `number` | 数値 | `30000` |
| `boolean` | 真偽値 | `true` |
| `json` | JSONオブジェクト | `{"key": "value"}` |
| `secret` | 暗号化された機密値 | APIキー |

### 3. 変数参照構文

```
${var.system.default_model}      -- システム変数
${var.tenant.api_timeout}        -- テナント変数
${var.workflow.max_retries}      -- ワークフロー変数
${var.run.user_input}            -- 実行時変数
${secret.openai_api_key}         -- シークレット（自動マスク）
```

### 4. UI表示

- シークレット値は `••••••••` でマスク表示
- 変数一覧で使用箇所を表示
- 未定義変数の参照をハイライト

---

## 技術設計

### データモデル

```sql
-- 変数テーブル
CREATE TABLE variables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),  -- NULLでシステム変数
    workflow_id UUID REFERENCES workflows(id),  -- NULLでテナント/システム変数
    name VARCHAR(255) NOT NULL,
    description TEXT,
    value_type VARCHAR(50) NOT NULL,  -- string|number|boolean|json|secret
    value TEXT,  -- 暗号化される場合あり
    is_secret BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(id),

    -- スコープに応じた一意制約
    CONSTRAINT unique_system_var UNIQUE NULLS NOT DISTINCT (tenant_id, workflow_id, name)
);

CREATE INDEX idx_variables_tenant ON variables(tenant_id);
CREATE INDEX idx_variables_workflow ON variables(workflow_id);
CREATE INDEX idx_variables_name ON variables(name);

-- 実行時変数（一時的、Redis推奨だがDBでも可）
CREATE TABLE run_variables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    value_type VARCHAR(50) NOT NULL,
    value TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(run_id, name)
);
```

### Domain層

**ファイル**: `backend/internal/domain/variable.go`

```go
package domain

type VariableScope string

const (
    VariableScopeSystem   VariableScope = "system"
    VariableScopeTenant   VariableScope = "tenant"
    VariableScopeWorkflow VariableScope = "workflow"
    VariableScopeRun      VariableScope = "run"
)

type VariableType string

const (
    VariableTypeString  VariableType = "string"
    VariableTypeNumber  VariableType = "number"
    VariableTypeBoolean VariableType = "boolean"
    VariableTypeJSON    VariableType = "json"
    VariableTypeSecret  VariableType = "secret"
)

type Variable struct {
    ID          uuid.UUID     `json:"id"`
    TenantID    *uuid.UUID    `json:"tenant_id,omitempty"`
    WorkflowID  *uuid.UUID    `json:"workflow_id,omitempty"`
    Name        string        `json:"name"`
    Description string        `json:"description,omitempty"`
    ValueType   VariableType  `json:"value_type"`
    Value       string        `json:"value"`  // シークレットは暗号化
    IsSecret    bool          `json:"is_secret"`
    CreatedAt   time.Time     `json:"created_at"`
    UpdatedAt   time.Time     `json:"updated_at"`
    CreatedBy   *uuid.UUID    `json:"created_by,omitempty"`
}

func (v *Variable) Scope() VariableScope {
    if v.TenantID == nil {
        return VariableScopeSystem
    }
    if v.WorkflowID == nil {
        return VariableScopeTenant
    }
    return VariableScopeWorkflow
}

func (v *Variable) MaskedValue() string {
    if v.IsSecret {
        return "••••••••"
    }
    return v.Value
}
```

### 暗号化サービス

**ファイル**: `backend/pkg/crypto/encryption.go`

```go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "io"
)

type EncryptionService struct {
    key []byte  // 32 bytes for AES-256
}

func NewEncryptionService(key string) *EncryptionService {
    return &EncryptionService{key: []byte(key)}
}

func (s *EncryptionService) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(s.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *EncryptionService) Decrypt(ciphertext string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(s.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

    plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

### 変数リゾルバ

**ファイル**: `backend/internal/engine/variable_resolver.go`

```go
package engine

import (
    "regexp"
    "strings"
)

var variablePattern = regexp.MustCompile(`\$\{(var|secret)\.([a-zA-Z_][a-zA-Z0-9_.]*)\}`)

type VariableResolver struct {
    systemVars   map[string]string
    tenantVars   map[string]string
    workflowVars map[string]string
    runVars      map[string]string
    secrets      map[string]string
}

func NewVariableResolver(
    systemVars, tenantVars, workflowVars, runVars, secrets map[string]string,
) *VariableResolver {
    return &VariableResolver{
        systemVars:   systemVars,
        tenantVars:   tenantVars,
        workflowVars: workflowVars,
        runVars:      runVars,
        secrets:      secrets,
    }
}

func (r *VariableResolver) Resolve(input string) (string, error) {
    return variablePattern.ReplaceAllStringFunc(input, func(match string) string {
        parts := variablePattern.FindStringSubmatch(match)
        if len(parts) != 3 {
            return match
        }

        varType := parts[1]
        varPath := parts[2]

        if varType == "secret" {
            if val, ok := r.secrets[varPath]; ok {
                return val
            }
            return match  // 未解決はそのまま
        }

        // var.scope.name の形式
        pathParts := strings.SplitN(varPath, ".", 2)
        if len(pathParts) != 2 {
            return match
        }

        scope := pathParts[0]
        name := pathParts[1]

        switch scope {
        case "system":
            if val, ok := r.systemVars[name]; ok {
                return val
            }
        case "tenant":
            if val, ok := r.tenantVars[name]; ok {
                return val
            }
        case "workflow":
            if val, ok := r.workflowVars[name]; ok {
                return val
            }
        case "run":
            if val, ok := r.runVars[name]; ok {
                return val
            }
        }

        return match
    }), nil
}
```

---

## API設計

### エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| GET | `/api/v1/variables` | テナント変数一覧 |
| POST | `/api/v1/variables` | テナント変数作成 |
| GET | `/api/v1/variables/{id}` | 変数詳細 |
| PUT | `/api/v1/variables/{id}` | 変数更新 |
| DELETE | `/api/v1/variables/{id}` | 変数削除 |
| GET | `/api/v1/workflows/{id}/variables` | WF変数一覧 |
| POST | `/api/v1/workflows/{id}/variables` | WF変数作成 |
| GET | `/api/v1/admin/variables` | システム変数一覧（管理者） |
| POST | `/api/v1/admin/variables` | システム変数作成（管理者） |

### Request/Response

**変数作成**:
```json
// POST /api/v1/variables
{
  "name": "openai_api_key",
  "description": "OpenAI API Key for this tenant",
  "value_type": "secret",
  "value": "sk-..."
}

// Response 201
{
  "id": "uuid",
  "name": "openai_api_key",
  "description": "OpenAI API Key for this tenant",
  "value_type": "secret",
  "value": "••••••••",  // マスク
  "is_secret": true,
  "created_at": "2024-01-01T00:00:00Z"
}
```

**変数一覧**:
```json
// GET /api/v1/variables?scope=tenant
{
  "variables": [
    {
      "id": "uuid",
      "name": "api_timeout",
      "value_type": "number",
      "value": "30000",
      "scope": "tenant"
    },
    {
      "id": "uuid",
      "name": "openai_api_key",
      "value_type": "secret",
      "value": "••••••••",
      "scope": "tenant"
    }
  ],
  "total": 2
}
```

---

## 実装ステップ

### Step 1: 暗号化サービス実装（0.5日）

**ファイル**: `backend/pkg/crypto/encryption.go`

- AES-256-GCM暗号化
- 環境変数からキー取得（`ENCRYPTION_KEY`）

### Step 2: Domain・Repository実装（1日）

**ファイル**:
- `backend/internal/domain/variable.go`
- `backend/internal/repository/interfaces.go`
- `backend/internal/repository/postgres/variable.go`
- `backend/migrations/009_add_variables.sql`

### Step 3: Usecase実装（1日）

**ファイル**: `backend/internal/usecase/variable.go`

```go
type VariableUsecase struct {
    repo       repository.VariableRepository
    encryption *crypto.EncryptionService
}

func (u *VariableUsecase) Create(ctx context.Context, input CreateVariableInput) (*domain.Variable, error) {
    // 名前の重複チェック
    // シークレットの場合は暗号化
    // 作成
}

func (u *VariableUsecase) GetDecrypted(ctx context.Context, id uuid.UUID) (*domain.Variable, error) {
    // 取得
    // シークレットの場合は復号化
}
```

### Step 4: Handler実装（0.5日）

**ファイル**: `backend/internal/handler/variable.go`

### Step 5: 変数リゾルバ統合（1日）

**ファイル**: `backend/internal/engine/executor.go`

```go
func (e *Executor) resolveVariables(ctx context.Context, execCtx *ExecutionContext, input json.RawMessage) (json.RawMessage, error) {
    // 各スコープの変数を取得
    systemVars := e.variableRepo.ListByScope(ctx, domain.VariableScopeSystem, nil, nil)
    tenantVars := e.variableRepo.ListByScope(ctx, domain.VariableScopeTenant, &execCtx.TenantID, nil)
    workflowVars := e.variableRepo.ListByScope(ctx, domain.VariableScopeWorkflow, &execCtx.TenantID, &execCtx.WorkflowID)

    // リゾルバ作成
    resolver := NewVariableResolver(...)

    // 入力内の変数を解決
    resolved, err := resolver.Resolve(string(input))
    return json.RawMessage(resolved), err
}
```

### Step 6: フロントエンド実装（2日）

**ファイル**:
- `frontend/composables/useVariables.ts`
- `frontend/pages/settings/variables.vue` - テナント変数管理
- `frontend/pages/workflows/[id]/variables.vue` - WF変数管理
- `frontend/components/variables/VariableForm.vue`
- `frontend/components/variables/VariableList.vue`

---

## セキュリティ考慮事項

| 考慮事項 | 対策 |
|----------|------|
| シークレット漏洩 | AES-256暗号化、復号化は実行時のみ |
| ログ出力 | シークレット値はログに絶対出力しない |
| UI表示 | マスク表示、コピーボタンなし |
| アクセス制御 | テナント分離、ロールベースアクセス |
| キー管理 | 環境変数、将来的にはVault連携 |

---

## テスト計画

### ユニットテスト

| テスト | 内容 |
|--------|------|
| 暗号化/復号化 | 正常系、エラー系 |
| 変数リゾルバ | 各スコープ、未定義変数 |
| CRUD | 作成、取得、更新、削除 |
| スコープ判定 | システム/テナント/WF |

### E2Eテスト

1. テナント変数作成 → WFで参照可能
2. シークレット作成 → 暗号化保存確認
3. WF実行時の変数解決確認
4. 未定義変数の警告表示

---

## 工数見積

| タスク | 工数 |
|--------|------|
| 暗号化サービス | 0.5日 |
| Domain/Repository | 1日 |
| Usecase | 1日 |
| Handler | 0.5日 |
| 変数リゾルバ統合 | 1日 |
| フロントエンド | 2日 |
| テスト | 1.5日 |
| ドキュメント | 0.5日 |
| **合計** | **8日** |
