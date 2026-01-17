# デプロイメントリファレンス

Docker、Kubernetes、および開発・本番環境の設定。

## クイックリファレンス

| 項目 | 値 |
|------|-------|
| 開発環境 | Docker Compose |
| 本番環境 | ECS（別途構築） |
| コンテナレジストリ | ローカルビルド / カスタム |
| API ポート | 8080 |
| フロントエンドポート | 3000 |
| Keycloak ポート | 8180 |
| Jaeger ポート | 16686 |
| ヘルスエンドポイント | `/health`, `/ready` |

## 開発環境

### Docker Compose サービス

| サービス | イメージ | ポート | 説明 |
|---------|-------|------|-------------|
| postgres | postgres:16-alpine | 5432 | PostgreSQL データベース |
| redis | redis:7-alpine | 6379 | キャッシュ & ジョブキュー |
| keycloak | keycloak:24.0 | 8180 | OIDC 認証 |
| api | ./backend | 8080 | Go API サーバー |
| worker | ./backend | - | ジョブプロセッサー |
| frontend | ./frontend | 3000 | Nuxt Web UI |
| jaeger | jaegertracing/all-in-one | 16686 | 分散トレーシング |

### コマンド

```bash
# 全サービス起動
docker compose up -d

# ビルドして起動
docker compose up -d --build

# ログ表示
docker compose logs -f api
docker compose logs -f worker
docker compose logs -f frontend

# 単一サービス再起動
docker compose restart api

# 全停止
docker compose down

# 停止してボリュームも削除
docker compose down -v

# 単一サービスを再ビルド
docker compose up -d --build api
```

### 環境変数

プロジェクトルートに `.env` ファイルを作成:

```bash
# LLM API キー
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...

# テレメトリ有効化
TELEMETRY_ENABLED=true
```

### サービス URL

| サービス | URL |
|---------|-----|
| API | http://localhost:8080 |
| フロントエンド | http://localhost:3000 |
| Keycloak 管理画面 | http://localhost:8180/admin (admin/admin) |
| Jaeger UI | http://localhost:16686 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |

### デフォルト認証情報

| サービス | ユーザー | パスワード |
|---------|------|----------|
| PostgreSQL | aio | aio_password |
| Keycloak 管理者 | admin | admin |
| テストユーザー（admin） | admin@example.com | admin123 |
| テストユーザー（builder） | builder@example.com | builder123 |

---

## 本番用 Dockerfile

### バックエンド

```dockerfile
# ビルドステージ
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker

# 本番ステージ
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /api /worker ./
USER appuser
EXPOSE 8080
CMD ["./api"]
```

### フロントエンド

```dockerfile
# ビルドステージ
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# 本番ステージ
FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.output ./.output
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
```

---

## ヘルスエンドポイント

### Liveness (/health)

- 即座にレスポンス
- プロセスが生存しているか確認
- 用途: K8s livenessProbe

```json
{"status": "ok"}
```

### Readiness (/ready)

- 依存関係をチェック
- 異常時は 503 を返す
- 用途: K8s readinessProbe

```json
{
  "status": "ok",
  "components": {
    "database": "ok",
    "redis": "ok"
  }
}
```

---

## モニタリング

### OpenTelemetry

有効化: `TELEMETRY_ENABLED=true`

トレースのエクスポート先: `OTEL_EXPORTER_OTLP_ENDPOINT`

### Jaeger 設定

```yaml
# docker-compose
jaeger:
  image: jaegertracing/all-in-one:1.54
  ports:
    - "16686:16686"  # UI
    - "4317:4317"    # OTLP gRPC
    - "4318:4318"    # OTLP HTTP
```

### 主要メトリクス

- リクエストレイテンシ（P50, P95, P99）
- エラーレート
- DAG 実行時間
- ステップ実行時間
- キュー深度
- アクティブ Run 数

---

## トラブルシューティング

### よくある問題

| 問題 | 原因 | 解決策 |
|-------|-------|-----|
| API 502 | DB 接続失敗 | DATABASE_URL、postgres の状態を確認 |
| Worker が処理しない | Redis 接続 | REDIS_URL、redis の状態を確認 |
| 認証エラー | Keycloak 利用不可 | KEYCLOAK_URL、keycloak の状態を確認 |
| クエリが遅い | インデックス不足 | データベースログを確認、EXPLAIN |
| OOM killed | メモリ制限が低すぎる | デプロイメントの制限を増加 |

### デバッグコマンド

```bash
# Docker Compose ログ
docker compose logs -f api
docker compose logs -f worker

# コンテナに入る
docker compose exec api sh

# DB 接続確認
docker compose exec api psql $DATABASE_URL -c "SELECT 1"

# Redis 確認
docker compose exec api redis-cli -u $REDIS_URL PING
```

---

## スケーリング考慮事項

| コンポーネント | 戦略 | 注記 |
|-----------|----------|-------|
| API | 水平スケーリング | ステートレス、CPU ベースのオートスケーリング推奨 |
| Worker | キューベースのスケーリング | 各 Worker は順次処理、Worker 数増加 = 並列性向上 |
| データベース | コネクションプーリング | PgBouncer 推奨、読み取り重視の場合はリードレプリカ |
| Redis | クラスターモード | キャッシュ用とキュー用で別インスタンス（オプション） |

## 関連ドキュメント

- [BACKEND.md](./BACKEND.md) - バックエンドアーキテクチャ
- [FRONTEND.md](./FRONTEND.md) - フロントエンドアーキテクチャ
- [DATABASE.md](./DATABASE.md) - データベーススキーマと接続設定
- [API.md](./API.md) - ヘルスチェックエンドポイント
