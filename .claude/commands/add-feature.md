# Add Feature Workflow

新機能追加時のワークフロー。

## ステップ

### 1. 関連ドキュメントを読む

```
1. docs/INDEX.md で関連ドキュメントを特定
2. 該当ドキュメントを読む
3. 既存の実装パターンを確認
```

### 2. 影響範囲を特定

| 変更対象 | 対応方針 |
|---------|---------|
| 単一ファイルのみ | 直接修正 |
| 複数ファイル（同一パッケージ） | パッケージ内で完結させる |
| 複数パッケージ | 影響範囲を全て確認してから着手 |

### 3. 実装

既存パターンに従って実装:

**Backend:**
- Handler → Usecase → Repository の順
- domain/ でエンティティ定義
- テストを同時に書く

**Frontend:**
- composables/ でロジック
- components/ でUI
- pages/ でルーティング

### 4. テスト

```bash
# Backend
cd backend && go test ./...

# Frontend
cd frontend && npm run check
```

### 5. サービス再起動（Docker環境）

```bash
docker compose restart api worker
```

### 6. 動作確認

ブラウザで動作確認。

### 7. ドキュメント更新

| 変更内容 | 更新対象 |
|---------|---------|
| API追加 | API.md, openapi.yaml |
| DB変更 | DATABASE.md |
| 新ブロック | BLOCK_REGISTRY.md |
| Backend構造 | BACKEND.md |
| Frontend構造 | FRONTEND.md |

## チェックリスト

```
[ ] 関連ドキュメントを読んだ
[ ] 影響範囲を特定した
[ ] 既存パターンに従って実装した
[ ] テストを書いた
[ ] テストが通ることを確認した
[ ] サービスを再起動した
[ ] 動作確認した
[ ] ドキュメントを更新した
```

## 参考

- [docs/rules/WORKFLOW_RULES.md](docs/rules/WORKFLOW_RULES.md)
- [docs/TESTING.md](docs/TESTING.md)
- [docs/DOCUMENTATION.md](docs/DOCUMENTATION.md)
