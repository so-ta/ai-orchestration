# Bug Fix Workflow

バグ修正時のワークフロー。

## ステップ

### 1. 再現テストを書く

バグを修正する**前に**、まずバグを再現するテストを書く。

```go
func TestBugXXX_Reproduction(t *testing.T) {
    // Setup
    // Execute
    // Assert - このテストは最初は失敗するはず
}
```

### 2. テスト失敗を確認

```bash
# Backend
cd backend && go test ./path/to/package/... -v -run TestBugXXX

# Frontend
cd frontend && npm run test:run -- --grep "bug description"
```

テストが**失敗する**ことを確認。

### 3. コードを修正

バグの原因を特定し、最小限の変更で修正。

### 4. テスト成功を確認

```bash
# Backend
cd backend && go test ./...

# Frontend
cd frontend && npm run check
```

### 5. 関連テストを追加

エッジケースのテストも追加：

- 境界値
- null/undefined
- 空配列/空文字列
- 異常系

### 6. サービス再起動（Docker環境）

```bash
docker compose restart api worker
```

### 7. 動作確認

ブラウザまたはcurlで実際に動作確認。

## チェックリスト

```
[ ] 再現テストを書いた
[ ] テストが失敗することを確認した
[ ] コードを修正した
[ ] テストが成功することを確認した
[ ] エッジケースのテストを追加した
[ ] サービスを再起動した
[ ] 動作確認した
[ ] 関連ドキュメントを更新した（必要な場合）
```

## よくあるバグパターン

| パターン | 確認ポイント |
|---------|-------------|
| nil/undefined | nullチェック漏れ |
| 型変換 | 暗黙の型変換、JSON.parse |
| 非同期 | await漏れ、Promise handling |
| 境界値 | off-by-one、空配列 |
| 状態管理 | race condition、stale state |

## 参考

- [docs/TESTING.md](docs/TESTING.md)
- [docs/rules/WORKFLOW_RULES.md](docs/rules/WORKFLOW_RULES.md)
