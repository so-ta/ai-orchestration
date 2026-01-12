# Phase 9: Cost Tracking 実装計画

> **Status: ✅ 実装済み**
>
> このフェーズは `usage_records`, `usage_budgets`, `usage_daily_aggregates` テーブルとして実装済みです。
> 実装の詳細は以下を参照:
> - `backend/internal/domain/usage.go`
> - `backend/internal/handler/usage.go`
> - `backend/internal/usecase/usage.go`
> - `backend/internal/repository/postgres/usage.go`
> - APIエンドポイント: `/api/v1/usage/*`

## 概要

**目的**: LLM API使用量とコストを追跡・可視化し、コスト管理と予算制御を可能にする。

**ユースケース例**:
- 月次コストレポートの自動生成
- ワークフロー別のコスト分析
- 予算超過時のアラート通知
- モデル別の使用量比較と最適化提案
- テナント別の課金基盤

---

## 機能要件

### 1. トラッキング対象

| 項目 | 説明 |
|------|------|
| トークン使用量 | input_tokens, output_tokens |
| コスト | USD換算（プロバイダー・モデル別単価） |
| レイテンシ | API応答時間（ms） |
| 成功/失敗 | API呼び出しの成否 |

### 2. 集計軸

| 軸 | 説明 |
|----|------|
| 時間 | 日別、週別、月別 |
| テナント | テナントごとの使用量 |
| ワークフロー | WFごとの使用量 |
| モデル | GPT-4o, Claude-3等 |
| プロバイダー | OpenAI, Anthropic等 |

### 3. ダッシュボード表示

- 今月のコスト合計
- 日別コスト推移グラフ
- ワークフロー別コストランキング
- モデル別使用量内訳（円グラフ）
- トークン使用量推移
- 予算消化率

---

## 技術設計

### データモデル

```sql
-- 使用量レコード
CREATE TABLE usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID REFERENCES workflows(id),
    run_id UUID REFERENCES runs(id),
    step_run_id UUID REFERENCES step_runs(id),

    -- プロバイダー情報
    provider VARCHAR(50) NOT NULL,  -- openai|anthropic|google|...
    model VARCHAR(100) NOT NULL,     -- gpt-4o|claude-3-opus|...
    operation VARCHAR(50) NOT NULL,  -- chat|completion|embedding|moderation|...

    -- 使用量
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,

    -- コスト
    input_cost_usd DECIMAL(12, 8) NOT NULL DEFAULT 0,
    output_cost_usd DECIMAL(12, 8) NOT NULL DEFAULT 0,
    total_cost_usd DECIMAL(12, 8) NOT NULL DEFAULT 0,

    -- メタデータ
    latency_ms INT,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- インデックス
CREATE INDEX idx_usage_tenant_date ON usage_records(tenant_id, created_at);
CREATE INDEX idx_usage_workflow ON usage_records(workflow_id);
CREATE INDEX idx_usage_run ON usage_records(run_id);
CREATE INDEX idx_usage_provider_model ON usage_records(provider, model);

-- 日次集計テーブル（パフォーマンス最適化）
CREATE TABLE usage_daily_aggregates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID REFERENCES workflows(id),
    date DATE NOT NULL,
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,

    total_requests INT NOT NULL DEFAULT 0,
    total_input_tokens BIGINT NOT NULL DEFAULT 0,
    total_output_tokens BIGINT NOT NULL DEFAULT 0,
    total_cost_usd DECIMAL(12, 6) NOT NULL DEFAULT 0,
    avg_latency_ms INT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, workflow_id, date, provider, model)
);

-- 予算設定
CREATE TABLE usage_budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID REFERENCES workflows(id),  -- NULLでテナント全体
    budget_type VARCHAR(50) NOT NULL,  -- daily|monthly
    budget_amount_usd DECIMAL(12, 2) NOT NULL,
    alert_threshold DECIMAL(3, 2) NOT NULL DEFAULT 0.80,  -- 80%で警告
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 価格設定

**ファイル**: `backend/internal/domain/pricing.go`

```go
package domain

// TokenPricing represents pricing per 1000 tokens
type TokenPricing struct {
    Provider    string
    Model       string
    InputPer1K  float64  // USD per 1K input tokens
    OutputPer1K float64  // USD per 1K output tokens
}

var DefaultPricing = []TokenPricing{
    // OpenAI
    {"openai", "gpt-4o", 0.0025, 0.01},
    {"openai", "gpt-4o-mini", 0.00015, 0.0006},
    {"openai", "gpt-4-turbo", 0.01, 0.03},
    {"openai", "gpt-3.5-turbo", 0.0005, 0.0015},

    // Anthropic
    {"anthropic", "claude-3-opus", 0.015, 0.075},
    {"anthropic", "claude-3-sonnet", 0.003, 0.015},
    {"anthropic", "claude-3-haiku", 0.00025, 0.00125},
    {"anthropic", "claude-3-5-sonnet", 0.003, 0.015},

    // Google
    {"google", "gemini-1.5-pro", 0.00125, 0.005},
    {"google", "gemini-1.5-flash", 0.000075, 0.0003},
}

func GetPricing(provider, model string) *TokenPricing {
    for _, p := range DefaultPricing {
        if p.Provider == provider && p.Model == model {
            return &p
        }
    }
    return nil
}

func CalculateCost(provider, model string, inputTokens, outputTokens int) (inputCost, outputCost, totalCost float64) {
    pricing := GetPricing(provider, model)
    if pricing == nil {
        return 0, 0, 0
    }

    inputCost = float64(inputTokens) / 1000 * pricing.InputPer1K
    outputCost = float64(outputTokens) / 1000 * pricing.OutputPer1K
    totalCost = inputCost + outputCost
    return
}
```

### 使用量レコーダー

**ファイル**: `backend/internal/engine/usage_recorder.go`

```go
package engine

type UsageRecorder struct {
    repo repository.UsageRepository
}

type UsageRecord struct {
    TenantID     uuid.UUID
    WorkflowID   uuid.UUID
    RunID        uuid.UUID
    StepRunID    uuid.UUID
    Provider     string
    Model        string
    Operation    string
    InputTokens  int
    OutputTokens int
    LatencyMs    int
    Success      bool
    ErrorMessage string
}

func (r *UsageRecorder) Record(ctx context.Context, record UsageRecord) error {
    // コスト計算
    inputCost, outputCost, totalCost := domain.CalculateCost(
        record.Provider, record.Model,
        record.InputTokens, record.OutputTokens,
    )

    usage := &domain.UsageRecord{
        ID:            uuid.New(),
        TenantID:      record.TenantID,
        WorkflowID:    record.WorkflowID,
        RunID:         record.RunID,
        StepRunID:     record.StepRunID,
        Provider:      record.Provider,
        Model:         record.Model,
        Operation:     record.Operation,
        InputTokens:   record.InputTokens,
        OutputTokens:  record.OutputTokens,
        TotalTokens:   record.InputTokens + record.OutputTokens,
        InputCostUSD:  inputCost,
        OutputCostUSD: outputCost,
        TotalCostUSD:  totalCost,
        LatencyMs:     record.LatencyMs,
        Success:       record.Success,
        ErrorMessage:  record.ErrorMessage,
        CreatedAt:     time.Now(),
    }

    return r.repo.Create(ctx, usage)
}
```

### Adapter統合

**ファイル**: `backend/internal/adapter/openai.go`（変更）

```go
func (a *OpenAIAdapter) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
    startTime := time.Now()

    // API呼び出し
    response, err := a.client.CreateChatCompletion(ctx, req)

    // 使用量記録
    latencyMs := int(time.Since(startTime).Milliseconds())

    usageRecord := engine.UsageRecord{
        TenantID:     middleware.GetTenantID(ctx),
        Provider:     "openai",
        Model:        config.Model,
        Operation:    "chat",
        InputTokens:  response.Usage.PromptTokens,
        OutputTokens: response.Usage.CompletionTokens,
        LatencyMs:    latencyMs,
        Success:      err == nil,
    }
    if err != nil {
        usageRecord.ErrorMessage = err.Error()
    }

    a.usageRecorder.Record(ctx, usageRecord)

    return output, err
}
```

---

## API設計

### エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| GET | `/api/v1/usage/summary` | 使用量サマリー |
| GET | `/api/v1/usage/daily` | 日別使用量 |
| GET | `/api/v1/usage/by-workflow` | ワークフロー別 |
| GET | `/api/v1/usage/by-model` | モデル別 |
| GET | `/api/v1/runs/{id}/usage` | Run詳細 |
| GET | `/api/v1/usage/budgets` | 予算一覧 |
| POST | `/api/v1/usage/budgets` | 予算設定 |
| PUT | `/api/v1/usage/budgets/{id}` | 予算更新 |

### Request/Response

**使用量サマリー**:
```json
// GET /api/v1/usage/summary?period=month
{
  "period": "2024-01",
  "total_cost_usd": 152.34,
  "total_requests": 12500,
  "total_input_tokens": 2500000,
  "total_output_tokens": 1200000,
  "by_provider": {
    "openai": {"cost_usd": 120.50, "requests": 10000},
    "anthropic": {"cost_usd": 31.84, "requests": 2500}
  },
  "by_model": {
    "gpt-4o": {"cost_usd": 80.00, "requests": 5000},
    "gpt-4o-mini": {"cost_usd": 40.50, "requests": 5000},
    "claude-3-sonnet": {"cost_usd": 31.84, "requests": 2500}
  },
  "budget": {
    "monthly_limit_usd": 200.00,
    "consumed_percent": 76.17
  }
}
```

**日別使用量**:
```json
// GET /api/v1/usage/daily?start=2024-01-01&end=2024-01-31
{
  "daily": [
    {
      "date": "2024-01-01",
      "total_cost_usd": 5.20,
      "total_requests": 450,
      "total_tokens": 120000
    },
    // ...
  ]
}
```

---

## フロントエンド設計

### ダッシュボードページ

**ファイル**: `frontend/pages/usage/index.vue`

```vue
<template>
  <div class="usage-dashboard">
    <!-- サマリーカード -->
    <div class="summary-cards">
      <SummaryCard
        :title="$t('usage.totalCost')"
        :value="formatCurrency(summary.total_cost_usd)"
        :trend="summary.cost_trend"
      />
      <SummaryCard
        :title="$t('usage.totalRequests')"
        :value="formatNumber(summary.total_requests)"
      />
      <SummaryCard
        :title="$t('usage.budgetUsed')"
        :value="formatPercent(summary.budget.consumed_percent)"
        :alert="summary.budget.consumed_percent > 80"
      />
    </div>

    <!-- 日別コストグラフ -->
    <CostChart :data="dailyData" />

    <!-- モデル別内訳 -->
    <ModelBreakdownChart :data="modelData" />

    <!-- ワークフロー別ランキング -->
    <WorkflowCostTable :data="workflowData" />
  </div>
</template>
```

### コンポーネント

| コンポーネント | 説明 |
|---------------|------|
| `SummaryCard.vue` | サマリー数値表示 |
| `CostChart.vue` | 日別コスト折れ線グラフ |
| `ModelBreakdownChart.vue` | モデル別円グラフ |
| `WorkflowCostTable.vue` | WF別コストテーブル |
| `BudgetSettings.vue` | 予算設定フォーム |

---

## 実装ステップ

### Step 1: Domain・Repository実装（1日）

**ファイル**:
- `backend/internal/domain/usage.go`
- `backend/internal/domain/pricing.go`
- `backend/internal/repository/postgres/usage.go`
- `backend/migrations/010_add_usage_records.sql`

### Step 2: 使用量レコーダー実装（0.5日）

**ファイル**: `backend/internal/engine/usage_recorder.go`

### Step 3: Adapter統合（1日）

**ファイル**:
- `backend/internal/adapter/openai.go`
- `backend/internal/adapter/anthropic.go`

### Step 4: Usecase・Handler実装（1日）

**ファイル**:
- `backend/internal/usecase/usage.go`
- `backend/internal/handler/usage.go`

### Step 5: 日次集計バッチ（0.5日）

**ファイル**: `backend/cmd/aggregate/main.go`

```go
// 日次集計ジョブ（cronで実行）
func aggregateDaily(ctx context.Context) error {
    yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

    query := `
        INSERT INTO usage_daily_aggregates (
            tenant_id, workflow_id, date, provider, model,
            total_requests, total_input_tokens, total_output_tokens,
            total_cost_usd, avg_latency_ms
        )
        SELECT
            tenant_id, workflow_id, DATE(created_at), provider, model,
            COUNT(*), SUM(input_tokens), SUM(output_tokens),
            SUM(total_cost_usd), AVG(latency_ms)
        FROM usage_records
        WHERE DATE(created_at) = $1
        GROUP BY tenant_id, workflow_id, DATE(created_at), provider, model
        ON CONFLICT (tenant_id, workflow_id, date, provider, model)
        DO UPDATE SET
            total_requests = EXCLUDED.total_requests,
            total_input_tokens = EXCLUDED.total_input_tokens,
            total_output_tokens = EXCLUDED.total_output_tokens,
            total_cost_usd = EXCLUDED.total_cost_usd,
            avg_latency_ms = EXCLUDED.avg_latency_ms,
            updated_at = NOW()
    `
    _, err := db.Exec(ctx, query, yesterday)
    return err
}
```

### Step 6: フロントエンド実装（3日）

**ファイル**:
- `frontend/composables/useUsage.ts`
- `frontend/pages/usage/index.vue`
- `frontend/components/usage/CostChart.vue`
- `frontend/components/usage/ModelBreakdownChart.vue`
- `frontend/components/usage/WorkflowCostTable.vue`

---

## テスト計画

### ユニットテスト

| テスト | 内容 |
|--------|------|
| 価格計算 | 各モデルのコスト計算 |
| 使用量記録 | レコード作成 |
| 集計クエリ | 日別/モデル別集計 |

### E2Eテスト

1. LLMステップ実行 → 使用量レコード作成確認
2. ダッシュボード表示確認
3. 予算アラート動作確認

---

## 工数見積

| タスク | 工数 |
|--------|------|
| Domain/Repository | 1日 |
| 使用量レコーダー | 0.5日 |
| Adapter統合 | 1日 |
| Usecase/Handler | 1日 |
| 日次集計バッチ | 0.5日 |
| フロントエンド | 3日 |
| テスト | 0.5日 |
| ドキュメント | 0.5日 |
| **合計** | **8日** |
