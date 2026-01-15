# Block Group Redesign - グループブロック再設計

> **Status**: ✅ Implemented (Phase A Complete)
> **Created**: 2025-01-15
> **Updated**: 2026-01-15
> **Author**: AI Agent

---

## 概要

グループブロックを抜本的に見直し、以下の原則で再設計する：

1. **グループは body のみ** - 内部にロール分割を持たない
2. **IN/OUT で制御** - 分岐・エラー処理は外部ポートで表現
3. **統廃合** - 6種類 → 4種類に整理
4. **pre_process / post_process** - 通常ブロックと同じ仕組みで入出力変換

---

## 現状の問題点

### 1. 並列処理の曖昧さ

```
現状：2つの方法で並列処理が発生
├── 暗黙の並列: 1つのブロックから複数ブロックに線を引く
└── 明示の並列: Parallel グループを使う

→ どちらが並列なのか分かりにくい
```

### 2. グループ内のロール分割が複雑

```
現状の Try-Catch:
┌─ Try-Catch ─────────────────┐
│  ┌─ TRY ────┐ ┌─ CATCH ──┐  │
│  │ [ブロック] │ │ [ブロック] │  │
│  └──────────┘ └──────────┘  │
└─────────────────────────────┘

→ グループ内にロール（try/catch/then/else）が混在
→ 視覚的に複雑、理解しづらい
```

### 3. 条件分岐グループと条件分岐ブロックの重複

```
現状：
├── if_else グループブロック
├── switch_case グループブロック
├── condition システムブロック
└── switch システムブロック

→ 同じ機能が2つの形式で存在
```

---

## 新設計

### グループブロック一覧（4種類）

| グループ | 用途 | IN | OUT |
|----------|------|-----|-----|
| `parallel` | 異なる処理を同時実行 | 単一オブジェクト | 集約結果 / error |
| `try_catch` | エラーハンドリング | 単一オブジェクト | success / error |
| `foreach` | 同じ処理を配列要素に適用 | 配列 | 結果配列 / error |
| `while` | 条件ループ | 単一オブジェクト | ループ結果 / error |

### 廃止するグループ

| グループ | 廃止理由 | 代替 |
|----------|----------|------|
| `if_else` | システムブロックで代替可能 | `condition` ブロック |
| `switch_case` | システムブロックで代替可能 | `switch` ブロック |

### 廃止するシステムブロック

| ブロック | 廃止理由 | 代替 |
|----------|----------|------|
| `loop` | グループブロックで代替可能 | `while` グループ |

---

## 新しいグループ構造

### 基本原則

```
【外部】
   │
   ▼ 外部からの IN
┌─────────────────────────────────────────────────┐
│  BlockGroup                                      │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │ pre_process (JS)                           │ │
│  │ 外部 IN → 内部 IN への変換                  │ │
│  └────────────────────────────────────────────┘ │
│                     │                            │
│                     ▼ 内部への IN                │
│  ┌────────────────────────────────────────────┐ │
│  │ body                                       │ │
│  │ [ブロック] ──> [ブロック] ──> [ブロック]    │ │
│  │ [ブロック] ──> [ブロック]                   │ │
│  └────────────────────────────────────────────┘ │
│                     │                            │
│                     ▼ 内部からの OUT             │
│  ┌────────────────────────────────────────────┐ │
│  │ post_process (JS)                          │ │
│  │ 内部 OUT → 外部 OUT への変換                │ │
│  └────────────────────────────────────────────┘ │
│                                                  │
└─────────────────────────────────────────────────┘
   │                              │
   ▼ 外部への OUT (success)       ▼ 外部への ERROR
【外部】                        【外部】
```

---

## 各グループの詳細設計

### 1. Parallel（並列実行）

**用途**: 異なる処理を同時に実行し、結果を集約

```
外部 IN: { userId: "123", options: {...} }
    │
    ▼ pre_process（各フローに同じ入力を配布）

┌─ Parallel ─────────────────────────────────────────┐
│                                                     │
│  【フロー1】                                        │
│  [API呼び出し] ──> [データ変換]                     │
│                                                     │
│  【フロー2】                                        │
│  ┌─ While ──────────────┐                          │
│  │  [ループ処理]         │ ──> [集計]              │
│  └──────────────────────┘                          │
│                                                     │
│  【フロー3】                                        │
│  [LLM呼び出し]                                      │
│                                                     │
└─────────────────────────────────────────────────────┘
    │
    ▼ post_process（結果を集約）

外部 OUT: {
  flow1: { ... },
  flow2: { ... },
  flow3: { ... }
}
```

**設定**:
```typescript
interface ParallelConfig {
  maxConcurrent?: number  // 最大同時実行数（0 = 無制限）
  failFast?: boolean      // 最初のエラーで全停止
}
```

**並列単位の判定**: グループ内の「連結成分」（繋がっていないフロー）を自動判定

**出力ポート**:
- `out`: すべてのフローが成功した場合
- `error`: いずれかのフローが失敗した場合（failFast時）

---

### 2. Try-Catch（エラーハンドリング）

**用途**: エラー発生時に error ポートへ分岐

```
外部 IN: { data: ... }
    │
    ▼ pre_process

┌─ Try-Catch ─────────────────────────────────────────┐
│                                                      │
│  [API呼び出し] ──> [データ検証] ──> [保存]          │
│                                                      │
└──────────────────────────────────────────────────────┘
    │                              │
    ▼ post_process (成功時)        ▼ エラー発生時

外部 OUT (success)              外部 OUT (error)
{ result: ... }                 { error: "...", input: {...} }
    │                              │
    ▼                              ▼
[次の処理]                     [エラー処理ブロック]
```

**設定**:
```typescript
interface TryCatchConfig {
  retryCount?: number   // リトライ回数（デフォルト: 0）
  retryDelay?: number   // リトライ間隔（ms）
}
```

**出力ポート**:
- `out`: 正常完了
- `error`: エラー発生（エラー情報 + 元の入力を出力）

**重要な変更点**:
- catch ロールを廃止
- エラー処理は外部ブロックで行う
- グループは body のみ

---

### 3. ForEach（配列反復）

**用途**: 配列の各要素に同じ処理を適用

```
外部 IN: { items: [A, B, C], context: {...} }
    │
    ▼ pre_process（配列を展開）

┌─ ForEach ───────────────────────────────────────────┐
│                                                      │
│  iteration 0: { item: A, index: 0, context: {...} } │
│  iteration 1: { item: B, index: 1, context: {...} } │
│  iteration 2: { item: C, index: 2, context: {...} } │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │  [処理ブロック] ──> [変換ブロック]            │   │
│  └──────────────────────────────────────────────┘   │
│                                                      │
└──────────────────────────────────────────────────────┘
    │
    ▼ post_process（結果を配列に集約）

外部 OUT: {
  results: [A', B', C'],
  _meta: { iterations: 3, ... }
}
```

**設定**:
```typescript
interface ForEachConfig {
  inputPath?: string    // 配列パス（デフォルト: "$.items"）
  parallel?: boolean    // 並列実行するか
  maxWorkers?: number   // 最大ワーカー数
}
```

**出力ポート**:
- `out`: すべての反復が成功
- `error`: いずれかの反復が失敗

**pre_process のデフォルト動作**:
```javascript
// 外部 IN から配列を取得し、各要素を内部 IN に変換
const items = getPath(input, config.inputPath || '$.items');
return items.map((item, index) => ({
  item,
  index,
  context: input.context || {}
}));
```

**post_process のデフォルト動作**:
```javascript
// 各反復の結果を配列に集約
return {
  results: outputs,
  _meta: {
    iterations: outputs.length,
    completedAt: new Date().toISOString()
  }
};
```

---

### 4. While（条件ループ）

**用途**: 条件が満たされる間、処理を繰り返す

```
外部 IN: { counter: 0, target: 5, data: [] }
    │
    ▼ pre_process

┌─ While ─────────────────────────────────────────────┐
│  condition: "$.counter < $.target"                   │
│                                                      │
│  iteration 0: { counter: 0, ... }                   │
│  iteration 1: { counter: 1, ... }                   │
│  iteration 2: { counter: 2, ... }                   │
│  ...                                                 │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │  [処理] ──> [カウンター更新]                  │   │
│  └──────────────────────────────────────────────┘   │
│                                                      │
└──────────────────────────────────────────────────────┘
    │
    ▼ post_process

外部 OUT: {
  result: { counter: 5, data: [...] },
  _meta: { iterations: 5, ... }
}
```

**設定**:
```typescript
interface WhileConfig {
  condition: string       // 条件式（例: "$.counter < $.target"）
  maxIterations?: number  // 安全制限（デフォルト: 100）
  doWhile?: boolean       // do-while形式（先に実行、後で判定）
}
```

**出力ポート**:
- `out`: 条件が false になりループ終了
- `error`: maxIterations 到達 または body 内でエラー

**ループの動作**:
1. 条件を評価
2. true の場合、body を実行
3. body の出力を次の反復の入力として使用
4. 1 に戻る

---

## データモデル

### BlockGroup エンティティ

```go
type BlockGroup struct {
    ID        uuid.UUID `json:"id"`
    TenantID  uuid.UUID `json:"tenantId"`
    WorkflowID uuid.UUID `json:"workflowId"`

    // 基本情報
    Name        string         `json:"name"`
    Type        BlockGroupType `json:"type"`  // parallel, try_catch, foreach, while
    Description string         `json:"description"`

    // 入出力変換（通常ブロックと同じ仕組み）
    PreProcess  *string `json:"preProcess"`   // JS: 外部IN → 内部IN
    PostProcess *string `json:"postProcess"`  // JS: 内部OUT → 外部OUT

    // グループ固有設定
    Config json.RawMessage `json:"config"`

    // ネスト対応
    ParentGroupID *uuid.UUID `json:"parentGroupId"`

    // 位置・サイズ（UI用）
    PositionX int `json:"positionX"`
    PositionY int `json:"positionY"`
    Width     int `json:"width"`
    Height    int `json:"height"`

    // タイムスタンプ
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

// グループタイプ（4種類のみ）
const (
    BlockGroupTypeParallel = "parallel"
    BlockGroupTypeTryCatch = "try_catch"
    BlockGroupTypeForEach  = "foreach"
    BlockGroupTypeWhile    = "while"
)
```

### 設定構造体

```go
// ParallelConfig 並列実行設定
type ParallelConfig struct {
    MaxConcurrent int  `json:"maxConcurrent"` // 0 = 無制限
    FailFast      bool `json:"failFast"`
}

// TryCatchConfig エラーハンドリング設定
type TryCatchConfig struct {
    RetryCount int `json:"retryCount"`
    RetryDelay int `json:"retryDelay"` // ms
}

// ForEachConfig 配列反復設定
type ForEachConfig struct {
    InputPath  string `json:"inputPath"`  // デフォルト: "$.items"
    Parallel   bool   `json:"parallel"`
    MaxWorkers int    `json:"maxWorkers"`
}

// WhileConfig 条件ループ設定
type WhileConfig struct {
    Condition     string `json:"condition"`
    MaxIterations int    `json:"maxIterations"` // デフォルト: 100
    DoWhile       bool   `json:"doWhile"`
}
```

---

## 出力ポート定義

### フロントエンド設定

```typescript
const groupOutputPorts: Record<BlockGroupType, OutputPort[]> = {
  parallel: [
    { name: 'out', label: 'Complete', color: '#22c55e' },
    { name: 'error', label: 'Error', color: '#ef4444' },
  ],
  try_catch: [
    { name: 'out', label: 'Success', color: '#22c55e' },
    { name: 'error', label: 'Error', color: '#ef4444' },
  ],
  foreach: [
    { name: 'out', label: 'Complete', color: '#22c55e' },
    { name: 'error', label: 'Error', color: '#ef4444' },
  ],
  while: [
    { name: 'out', label: 'Done', color: '#22c55e' },
    { name: 'error', label: 'Error', color: '#ef4444' },
  ],
}
```

---

## 実行エンジンの変更

### Parallel 実行

```go
func (e *BlockGroupExecutor) executeParallel(ctx context.Context, group *BlockGroup, input json.RawMessage) (*BlockGroupResult, error) {
    config := parseParallelConfig(group.Config)

    // 1. pre_process で入力を変換
    internalInput, err := e.runPreProcess(ctx, group, input)
    if err != nil {
        return nil, err
    }

    // 2. body 内の連結成分（独立したフロー）を検出
    flows := e.detectFlows(group.ID)

    // 3. セマフォで並列数を制御
    sem := make(chan struct{}, config.MaxConcurrent)
    if config.MaxConcurrent == 0 {
        sem = make(chan struct{}, len(flows))
    }

    // 4. 各フローを並列実行
    var wg sync.WaitGroup
    results := make(map[string]json.RawMessage)
    var firstError error
    var mu sync.Mutex

    for _, flow := range flows {
        wg.Add(1)
        go func(f *Flow) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            result, err := e.executeFlow(ctx, f, internalInput)
            mu.Lock()
            defer mu.Unlock()

            if err != nil {
                if firstError == nil {
                    firstError = err
                }
                if config.FailFast {
                    return
                }
            }
            results[f.Name] = result
        }(flow)
    }

    wg.Wait()

    // 5. post_process で出力を変換
    internalOutput, _ := json.Marshal(results)
    output, err := e.runPostProcess(ctx, group, internalOutput)

    if firstError != nil && config.FailFast {
        return &BlockGroupResult{
            Output: output,
            Port:   "error",
            Error:  firstError,
        }, nil
    }

    return &BlockGroupResult{
        Output: output,
        Port:   "out",
    }, nil
}
```

### Try-Catch 実行

```go
func (e *BlockGroupExecutor) executeTryCatch(ctx context.Context, group *BlockGroup, input json.RawMessage) (*BlockGroupResult, error) {
    config := parseTryCatchConfig(group.Config)

    // 1. pre_process で入力を変換
    internalInput, err := e.runPreProcess(ctx, group, input)
    if err != nil {
        return nil, err
    }

    // 2. body を実行（リトライ付き）
    var lastError error
    for attempt := 0; attempt <= config.RetryCount; attempt++ {
        if attempt > 0 {
            time.Sleep(time.Duration(config.RetryDelay) * time.Millisecond)
        }

        result, err := e.executeBody(ctx, group, internalInput)
        if err == nil {
            // 3. 成功時: post_process で出力を変換
            output, _ := e.runPostProcess(ctx, group, result)
            return &BlockGroupResult{
                Output: output,
                Port:   "out",
            }, nil
        }
        lastError = err
    }

    // 4. 失敗時: エラー情報を出力
    errorOutput, _ := json.Marshal(map[string]any{
        "error": lastError.Error(),
        "input": json.RawMessage(input),
    })

    return &BlockGroupResult{
        Output: errorOutput,
        Port:   "error",
        Error:  lastError,
    }, nil
}
```

---

## マイグレーション計画

### Phase 1: データベース変更

```sql
-- 1. block_groups テーブルに pre_process, post_process カラム追加
ALTER TABLE block_groups
ADD COLUMN pre_process TEXT,
ADD COLUMN post_process TEXT;

-- 2. group_role カラムを steps から削除（既存データのマイグレーション後）
-- ALTER TABLE steps DROP COLUMN group_role;
```

### Phase 2: バックエンド変更

1. `domain/block_group.go` - 新しい構造体定義
2. `engine/block_group_executor.go` - 新しい実行ロジック
3. 廃止グループ（if_else, switch_case）の削除
4. loop システムブロックの削除

### Phase 3: フロントエンド変更

1. `composables/useBlockGroups.ts` - API更新
2. `components/dag-editor/DagEditor.vue` - ゾーン分割UIの削除
3. グループの pre_process / post_process 編集UI追加

### Phase 4: データマイグレーション

1. 既存の if_else グループ → condition ブロックに変換
2. 既存の switch_case グループ → switch ブロックに変換
3. 既存の loop ブロック → while グループに変換
4. try/catch/then/else ロールのステップ → 外部ブロックに移動

---

## 移行ガイド

### if_else グループ → condition ブロック

```
【Before】
┌─ If-Else ─────────────────────────────────┐
│  condition: "$.status == 'active'"         │
│  ┌─ THEN ─────┐  ┌─ ELSE ─────┐           │
│  │ [処理A]     │  │ [処理B]     │           │
│  └────────────┘  └────────────┘           │
└────────────────────────────────────────────┘

【After】
                  ┌──> [処理A] ──> ...
[condition] ─────┤
  $.status ==    └──> [処理B] ──> ...
  'active'
```

### try_catch グループ → 新 try_catch グループ + 外部ブロック

```
【Before】
┌─ Try-Catch ────────────────────────────────┐
│  ┌─ TRY ──────┐  ┌─ CATCH ────┐            │
│  │ [API呼出]   │  │ [エラー処理] │            │
│  └────────────┘  └────────────┘            │
└────────────────────────────────────────────┘

【After】
┌─ Try-Catch ────────────────┐
│  [API呼出]                  │──> (out) ──> [次の処理]
└────────────────────────────┘
              │
              └──> (error) ──> [エラー処理]
```

---

## 関連ドキュメント

- [UNIFIED_BLOCK_MODEL.md](./UNIFIED_BLOCK_MODEL.md) - 統一ブロックモデル
- [BLOCK_REGISTRY.md](../BLOCK_REGISTRY.md) - ブロック定義一覧
- [BACKEND.md](../BACKEND.md) - バックエンド実装パターン
