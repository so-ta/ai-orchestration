# AI Orchestration - ドキュメント索引

> **AI駆動開発**: このプロジェクトはすべてAIエージェントが実装・保守します。
> 新しいセッション開始時は必ず [CLAUDE.md](../CLAUDE.md) を最初に読んでください。

## セッション開始チェックリスト

```
1. [ ] ../CLAUDE.md を読む（プロジェクト概要）
2. [ ] このファイルで関連ドキュメントを特定
3. [ ] 作業対象のドキュメントを読む
4. [ ] 既存コードパターンを確認
```

---

## 技術ドキュメント

| ドキュメント | 目的 | 読むタイミング |
|----------|---------|--------------|
| [BACKEND.md](./BACKEND.md) | Goバックエンドの構造、インターフェース、パターン | バックエンドコードの修正時 |
| [FRONTEND.md](./FRONTEND.md) | Nuxt/Vueの構造、Composables、コンポーネント | フロントエンドコードの修正時 |
| [API.md](./API.md) | RESTエンドポイント、リクエスト/レスポンススキーマ | API連携、エンドポイント追加時 |
| [DATABASE.md](./DATABASE.md) | スキーマ、クエリ | データベース操作時 |
| [DEPLOYMENT.md](./DEPLOYMENT.md) | Docker、Kubernetes、環境設定 | DevOps、デプロイ時 |
| [BLOCK_REGISTRY.md](./BLOCK_REGISTRY.md) | ブロック定義、エラーコード | **新規ブロック追加時** |
| [INTEGRATIONS.md](./INTEGRATIONS.md) | 外部サービス連携一覧 | 連携ブロック追加・利用時 |
| [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) | エラー対処法 | エラー発生時 |

## 開発ルール

作業種類に応じて必要なルールを参照:

| ルールドキュメント | 目的 | 読むタイミング |
|---------------|---------|--------------|
| [WORKFLOW_RULES](./rules/WORKFLOW_RULES.md) | 開発ワークフロー全般（Why/過去の失敗例あり） | すべての開発作業 |
| [GIT_RULES](./rules/GIT_RULES.md) | コミット、PR、コンフリクト解消 | コミット・PR作成時 |
| [TESTING](./TESTING.md) | テスト作成・実行（統合ガイド） | テスト作成・実行時 |
| [DOCUMENTATION](./DOCUMENTATION.md) | ドキュメント作成・同期（統合ガイド） | ドキュメント更新時 |
| [CODEX_REVIEW](./rules/CODEX_REVIEW.md) | PRレビューフロー | PR push後 |

## テストドキュメント

| ドキュメント | 目的 | 読むタイミング |
|----------|---------|--------------|
| [TESTING.md](./TESTING.md) | テスト統合ガイド（優先度マトリックス含む） | テスト作成・実行時 |
| [BACKEND_TESTING.md](./BACKEND_TESTING.md) | Goバックエンドのテストパターン | バックエンドテスト実装時 |
| [frontend/docs/TESTING.md](../frontend/docs/TESTING.md) | フロントエンドテストルール | フロントエンドコード変更時 |

---

## アーキテクチャ設計

| 設計 | 説明 | ステータス | ドキュメント |
|--------|-------------|--------|----------|
| 統一ブロックモデル | ブロック実行の統一モデル | ✅ 実装済み | [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) |
| ブロック設定改善 | ブロック設定UI改善 | 📋 設計中 | [BLOCK_CONFIG_IMPROVEMENT.md](./designs/BLOCK_CONFIG_IMPROVEMENT.md) |

## 実装ステータス（Source of Truth）

> この表が実装状態の正（Source of Truth）です。各計画書内の記載は補助情報です。

| フェーズ | 機能 | ステータス | ドキュメント | 関連PR/コミット |
|-------|---------|--------|----------|-------------------|
| コア | 統一ブロックモデル | ✅ 完了 | [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) | - |
| コア | ブロックグループ再設計 | ✅ 完了 | [BLOCK_GROUP_REDESIGN.md](./designs/BLOCK_GROUP_REDESIGN.md) | - |
| コア | ブロック設定改善 | 🚧 Phase 3完了 | [BLOCK_CONFIG_IMPROVEMENT.md](./designs/BLOCK_CONFIG_IMPROVEMENT.md) | - |
| コア | リッチビュー出力 | ✅ 完了 | [RICH_VIEW_OUTPUT.md](./designs/RICH_VIEW_OUTPUT.md) | - |
| 6 | ガードレール | 📋 未着手 | [PHASE6_GUARDRAILS.md](./plans/PHASE6_GUARDRAILS.md) | - |
| 7 | 評価器 | 📋 未着手 | [PHASE7_EVALUATOR.md](./plans/PHASE7_EVALUATOR.md) | - |
| 8 | 変数システム | ✅ 完了 | [PHASE8_VARIABLES.md](./plans/PHASE8_VARIABLES.md) | - |
| 9 | コストトラッキング | ✅ 完了 | [PHASE9_COST_TRACKING.md](./plans/PHASE9_COST_TRACKING.md) | - |
| 10 | コパイロット | 📋 未着手 | [PHASE10_COPILOT.md](./plans/PHASE10_COPILOT.md) | - |
| 特別 | RAG実装 | 🚧 進行中 | [RAG_IMPLEMENTATION_PLAN.md](./plans/RAG_IMPLEMENTATION_PLAN.md) | - |

**ステータス凡例**:
- ✅ 完了: 実装完了
- 🚧 進行中 / Phase X完了: 実装中（フェーズ完了）
- 📋 未着手: 未実装

**推奨実装順序**: Phase 6 → 7 → 10

---

## システム概要

```
アーキテクチャ: クリーンアーキテクチャ (Handler -> Usecase -> Domain -> Repository)
テナント: マルチテナント（tenant_id による分離）
認証: Keycloak OIDC (JWT)
キュー: Redisベースのジョブキュー
トレーシング: OpenTelemetry -> Jaeger
```

## コアコンセプト（クイックリファレンス）

### ワークフローの状態

```
draft -> published (不変)
```

### 実行（Run）の状態

```
pending -> running -> completed | failed | cancelled
```

### ステップタイプ

詳細は [BACKEND.md](./BACKEND.md#domain-models) を参照。

| タイプ | 目的 |
|------|---------|
| `start` | エントリーポイント |
| `llm` | LLM API呼び出し |
| `tool` | アダプター実行 |
| `condition` | 分岐ルーティング（2方向） |
| `switch` | 多分岐ルーティング |
| `map` | 配列の並列/逐次処理 |
| `join` | 分岐のマージ |
| `subflow` | ネストされたワークフロー |
| `loop` | 反復処理 |
| `filter` | アイテムのフィルタリング |
| `log` | デバッグログ |

### アダプター

詳細は [BACKEND.md](./BACKEND.md#adapter-implementations) を参照。

| ID | 目的 |
|----|---------|
| `mock` | テスト用 |
| `openai` | GPT API |
| `anthropic` | Claude API |
| `http` | 汎用HTTP |

---

## よくある操作

### 新規ブロック/連携の追加

**スラッシュコマンドを使用**: `/add-block`

または [.claude/commands/add-block.md](../.claude/commands/add-block.md) を参照。

### 新規APIエンドポイントの追加

1. `backend/internal/handler/` にハンドラーを追加
2. `cmd/api/main.go` にルートを追加
3. 新しいビジネスロジックが必要な場合はUsecaseを追加
4. `docs/API.md` と `docs/openapi.yaml` を更新

### 新規ステップタイプの追加

1. `backend/internal/domain/step.go` に定義を追加
2. `backend/internal/engine/executor.go` に実行ロジックを追加
3. フロントエンドのステップ設定UIを更新
4. `docs/BACKEND.md` を更新

### バグ修正

**スラッシュコマンドを使用**: `/fix-bug`

---

## テストコマンド

```bash
# バックエンド
cd backend && go test ./...
cd backend && go test ./tests/e2e/... -v

# フロントエンド（コミット前に必須）
cd frontend && npm run check
```

---

## URL（開発環境）

| サービス | URL |
|---------|-----|
| API | http://localhost:8080 |
| フロントエンド | http://localhost:3000 |
| Keycloak管理画面 | http://localhost:8180/admin |
| Jaeger UI | http://localhost:16686 |
