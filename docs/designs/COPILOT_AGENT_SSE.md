# Copilot Agent SSE ストリーミング設計

> **Status**: 実装済み
> **Created**: 2026-01-21
> **Updated**: 2026-01-21

## 概要

Copilot機能をポーリングベースからSSE（Server-Sent Events）ストリーミングベースに移行し、自律的なエージェントによるワークフロー構築を実現する。

## 背景と決定経緯

### 課題

従来のCopilot実装（PHASE10_COPILOT.md参照）では以下の課題があった：

| 課題 | 詳細 |
|------|------|
| レイテンシ | ポーリング間隔（1秒）による遅延 |
| UX | 処理中の進捗が見えない |
| 柔軟性 | 単発の生成のみ、対話的な改善が困難 |
| ツール実行 | ブロック情報取得などの中間ステップが不透明 |

### 決定事項

1. **SSEストリーミング採用**
   - リアルタイムでの応答表示
   - ツール実行状況の可視化
   - 中断可能な処理フロー

2. **エージェントアーキテクチャ**
   - ツール呼び出しによる自律的なワークフロー構築
   - 反復的な思考・実行ループ
   - コンテキストを保持したセッション管理

3. **インライン提案カード（Claude Code風）**
   - 変更内容をカード形式で提示
   - 適用/却下/修正依頼のアクション
   - 適用済み状態の永続化

## アーキテクチャ

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Copilot Agent SSE アーキテクチャ                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Frontend (CopilotTab.vue)                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │  EventSource API                                                     ││
│  │    ↓                                                                 ││
│  │  onThinking → 思考中UI表示                                           ││
│  │  onToolCall → ツール実行中UI表示                                     ││
│  │  onToolResult → ツール結果UI更新                                     ││
│  │  onPartialText → ストリーミングテキスト表示                          ││
│  │  onComplete → 提案カード表示                                         ││
│  │  onError → エラー表示                                                ││
│  └─────────────────────────────────────────────────────────────────────┘│
│       │ GET /workflows/{id}/copilot/agent/sessions/{sid}/stream         │
│       ▼                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │  Backend (copilot_agent.go)                                          ││
│  │                                                                       ││
│  │  StreamAgentMessage Handler                                          ││
│  │    ↓                                                                 ││
│  │  AgentUsecase.RunAgentLoop()                                         ││
│  │    ↓                                                                 ││
│  │  ┌─────────────────────────────────────────────────────────────────┐││
│  │  │  Agent Loop (最大10イテレーション)                               │││
│  │  │                                                                   │││
│  │  │  1. LLM呼び出し（Anthropic Claude）                              │││
│  │  │     ↓                                                             │││
│  │  │  2. レスポンス解析                                                │││
│  │  │     ├─ thinking → SSE: event:thinking                            │││
│  │  │     ├─ tool_use → SSE: event:tool_call → ツール実行              │││
│  │  │     │              → SSE: event:tool_result                      │││
│  │  │     └─ text → SSE: event:partial_text                            │││
│  │  │     ↓                                                             │││
│  │  │  3. end_turn または max iterations で終了                        │││
│  │  │     → SSE: event:complete                                        │││
│  │  └─────────────────────────────────────────────────────────────────┘││
│  └─────────────────────────────────────────────────────────────────────┘│
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## API設計

### エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| POST | `/workflows/{id}/copilot/agent/sessions` | セッション作成 |
| GET | `/workflows/{id}/copilot/agent/sessions/active` | アクティブセッション取得 |
| GET | `/workflows/{id}/copilot/agent/sessions/{sid}` | セッション取得 |
| POST | `/workflows/{id}/copilot/agent/sessions/{sid}/messages` | メッセージ送信（非ストリーム） |
| GET | `/workflows/{id}/copilot/agent/sessions/{sid}/stream` | SSEストリーム開始 |
| POST | `/workflows/{id}/copilot/agent/sessions/{sid}/cancel` | ストリームキャンセル |

### SSEイベント形式

```
event: thinking
data: {"type":"thinking","data":{"content":"ワークフローを分析中...","iteration":1}}

event: tool_call
data: {"type":"tool_call","data":{"tool":"search_blocks","arguments":{"query":"llm"}}}

event: tool_result
data: {"type":"tool_result","data":{"tool":"search_blocks","result":{...},"is_error":false}}

event: partial_text
data: {"type":"partial_text","data":{"content":"LLMブロックを"}}

event: complete
data: {"type":"complete","data":{"response":"...","tools_used":["search_blocks","create_step"],"iterations":3}}

event: error
data: {"type":"error","data":{"error":"API error message"}}

event: done
data: {}
```

## ツール定義

エージェントが利用可能なツール：

| ツール名 | 説明 | 引数 |
|---------|------|------|
| `list_blocks` | ブロック一覧取得 | `category?` |
| `search_blocks` | ブロック検索 | `query` |
| `get_block_schema` | ブロックスキーマ取得 | `slug` |
| `get_workflow` | ワークフロー取得 | - |
| `list_workflow_steps` | ステップ一覧取得 | - |
| `create_step` | ステップ作成 | `name`, `type`, `config`, `position` |
| `update_step` | ステップ更新 | `step_id`, `patch` |
| `delete_step` | ステップ削除 | `step_id` |
| `create_edge` | エッジ作成 | `source_id`, `target_id`, `ports?` |
| `delete_edge` | エッジ削除 | `edge_id` |
| `validate_workflow` | ワークフロー検証 | - |
| `search_docs` | ドキュメント検索 | `query` |

## フロントエンド実装

### useCopilot.ts

```typescript
// SSEストリーミング関数
function streamAgentMessage(
  projectId: string,
  sessionId: string,
  message: string,
  callbacks: StreamCallbacks
): { cancel: () => void } {
  const url = `${baseUrl}/workflows/${projectId}/copilot/agent/sessions/${sessionId}/stream?message=${encodeURIComponent(message)}`
  const eventSource = new EventSource(url, { withCredentials: true })

  eventSource.addEventListener('thinking', (e) => {
    const data = JSON.parse(e.data)
    callbacks.onThinking?.(data.data.content)
  })

  eventSource.addEventListener('tool_call', (e) => {
    const data = JSON.parse(e.data)
    callbacks.onToolCall?.(data.data.tool, data.data.arguments)
  })

  // ... 他のイベントハンドラ

  return {
    cancel: () => {
      eventSource.close()
      // キャンセルAPIを呼び出し
    }
  }
}
```

### CopilotTab.vue 状態管理

```typescript
// Agent state - 他のstateと一緒に先に定義（重要）
const agentStreamState = ref<AgentStreamState>({
  isStreaming: false,
  currentThinking: '',
  toolSteps: [],
  partialResponse: '',
  finalResponse: null,
  error: null,
})
const currentCancelFn = ref<(() => void) | null>(null)
```

**重要**: `agentStreamState`は`watch`や`onMounted`より前に定義する必要がある。
これはVue 3のscript setupにおける変数の初期化順序に関連する。

### 提案カード（CopilotProposalCard.vue）

Claude Code風のインライン提案カード：

```vue
<template>
  <div class="proposal-card" :class="proposal.status">
    <header>
      <span class="badge">+{{ proposal.changes.length }}</span>
      提案された変更
      <span v-if="proposal.status === 'applied'" class="status">✓ 適用済み</span>
    </header>

    <div class="changes">
      <CopilotChangeItem
        v-for="change in proposal.changes"
        :key="change.id"
        :change="change"
      />
    </div>

    <footer v-if="proposal.status === 'pending'">
      <button @click="$emit('discard', proposal.id)">却下</button>
      <button @click="showModifyDialog = true">修正依頼</button>
      <button @click="$emit('apply', proposal.id)" class="primary">適用</button>
    </footer>
  </div>
</template>
```

## データモデル

### CopilotSession

```go
type CopilotSession struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    WorkflowID  uuid.UUID
    UserID      uuid.UUID
    Status      SessionStatus  // active, completed, cancelled
    Messages    []CopilotMessage
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type CopilotMessage struct {
    ID            uuid.UUID
    SessionID     uuid.UUID
    Role          string  // user, assistant
    Content       string
    ExtractedData json.RawMessage  // tool_executions, proposal
    CreatedAt     time.Time
}
```

### ExtractedData構造

```json
{
  "tool_executions": [
    {
      "tool_name": "create_step",
      "arguments": { "name": "LLM Chat", "type": "llm" },
      "result": { "step_id": "..." },
      "is_error": false,
      "timestamp": "2026-01-21T..."
    }
  ],
  "proposal": {
    "id": "draft-xxx",
    "status": "pending",
    "changes": [
      {
        "type": "step:create",
        "temp_id": "temp-xxx",
        "name": "LLM Chat",
        "step_type": "llm",
        "config": { ... }
      }
    ]
  }
}
```

## 変更適用フロー

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         変更適用フロー                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. Copilot Agent がツール実行                                           │
│     create_step({ name: "LLM", type: "llm", ... })                      │
│     ↓                                                                    │
│  2. useCopilotDraft に変更を蓄積                                         │
│     addToDraft({ type: 'step:create', ... })                            │
│     ↓                                                                    │
│  3. Agent完了時に提案カードとして表示                                     │
│     CopilotProposalCard                                                  │
│     ↓                                                                    │
│  4. ユーザーが「適用」クリック                                            │
│     ↓                                                                    │
│  5. emit('changes:applied', changes)                                    │
│     ↓                                                                    │
│  6. 親コンポーネント（WorkflowEditor）で処理                              │
│     ↓                                                                    │
│  7. useCommandHistory で Undo/Redo対応コマンド実行                        │
│     executeCommand(new CreateStepCommand(...))                           │
│     ↓                                                                    │
│  8. API呼び出し・キャンバス更新                                           │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## バグ修正履歴

### agentStreamState初期化エラー（2026-01-21）

**問題**:
```
ReferenceError: Cannot access 'agentStreamState' before initialization
```

**原因**:
`agentStreamState`が`watch`や`onMounted`より後に定義されていた。
Vue 3のscript setupでは、変数は定義順に初期化されるため、
`watch`の中で参照される変数は事前に定義されている必要がある。

**修正**:
`agentStreamState`と`currentCancelFn`の定義を他のstate変数と一緒に
ファイル上部（64行目付近）に移動。

```typescript
// 修正前
onMounted(() => { ... })  // Line 183

const agentStreamState = ref<...>({...})  // Line 189 ← 後ろで定義

watch([() => agentStreamState.value...])  // Line 201 ← 前の定義を参照


// 修正後
const agentStreamState = ref<...>({...})  // Line 65 ← 先に定義
const currentCancelFn = ref<...>(null)    // Line 73

// ... 他の関数定義 ...

onMounted(() => { ... })  // Line 183

watch([() => agentStreamState.value...])  // Line 190 ← 問題なく参照可能
```

## パフォーマンス考慮事項

| 項目 | 対策 |
|------|------|
| SSE接続タイムアウト | 30秒でタイムアウト、再接続ロジック |
| メモリリーク | コンポーネントunmount時にEventSource.close() |
| 大量メッセージ | チャット履歴の仮想スクロール（将来対応） |
| 同時接続数 | 1ワークフローにつき1アクティブセッション |

## セキュリティ考慮事項

| 項目 | 対策 |
|------|------|
| 認証 | SSE接続時もCookieベース認証を使用 |
| テナント分離 | セッションはテナントIDでスコープ |
| ツール制限 | 読み取り系ツールと書き込み系ツールの分離 |
| レート制限 | 1分あたり10リクエストの制限 |

## 今後の拡張

| 機能 | 説明 | 優先度 |
|------|------|--------|
| マルチターン改善 | コンテキストウィンドウの最適化 | 高 |
| プリセット提案 | よく使うパターンの提案 | 中 |
| 変更プレビュー | 適用前のビジュアルプレビュー | 中 |
| バッチ適用 | 複数提案の一括適用 | 低 |

## 関連ドキュメント

- [PHASE10_COPILOT.md](../plans/PHASE10_COPILOT.md) - 初期Copilot設計
- [AI_WORKFLOW_BUILDER.md](../plans/AI_WORKFLOW_BUILDER.md) - AIワークフロービルダー計画
- [FRONTEND.md](../FRONTEND.md) - フロントエンドアーキテクチャ
- [API.md](../API.md) - API設計
