# コンポーネント分割ルール

大規模なVueコンポーネントを分割するための標準パターン。

---

## 分割基準

| セクション | 閾値 | 対応 |
|------------|------|------|
| **総行数** | >500行 | 分割検討 |
| **総行数** | >1000行 | 分割必須 |
| **script setup** | >200行 | composable抽出 |
| **template** | >300行 | 子コンポーネント分離 |
| **style** | >400行 | 共通スタイル抽出 |

---

## 分割優先順位

1. **Composables抽出** - ロジックをcomposableに分離（最も安全）
2. **読み取り専用セクション** - 表示のみのセクションを分離
3. **フォームセクション** - v-modelパターンで分離
4. **スタイル分離** - 共通スタイルを外部化

---

## ディレクトリ構造パターン

```
components/feature-name/
├── FeatureName.vue           # メインコンポーネント
├── composables/              # ロジック分離
│   ├── useFeatureLogic.ts
│   └── useFeatureState.ts
├── sections/                 # UIセクション分離
│   ├── FeatureHeader.vue
│   ├── FeatureBody.vue
│   └── FeatureFooter.vue
└── legacy/                   # レガシーフォールバック
    └── LegacyForm.vue
```

---

## コンポーネント設計パターン

### 1. 表示専用コンポーネント

Props のみ受け取り、Emit なし。

```typescript
interface Props {
  data: SomeType
}
const props = defineProps<Props>()
// Emits なし
```

### 2. フォームコンポーネント（v-model）

双方向バインディングパターン。

```typescript
interface Props {
  modelValue: ConfigType
  disabled?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: ConfigType): void
}>()

// 内部状態
const localValue = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
```

### 3. アクションコンポーネント

イベントを発火するコンポーネント。

```typescript
interface Props {
  item: ItemType
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'action-name', payload: PayloadType): void
}>()
```

---

## Composable 設計パターン

### 状態管理 composable

```typescript
// composables/useFeatureState.ts
export function useFeatureState() {
  const items = ref<Item[]>([])
  const selectedId = ref<string | null>(null)

  const selectedItem = computed(() =>
    items.value.find(i => i.id === selectedId.value)
  )

  function selectItem(id: string) {
    selectedId.value = id
  }

  return {
    items: readonly(items),
    selectedItem,
    selectItem
  }
}
```

### ロジック composable

```typescript
// composables/useFeatureLogic.ts
export function useFeatureLogic(config: Ref<Config>) {
  function processData(input: Input): Output {
    // 純粋なロジック
    return transform(input, config.value)
  }

  const derivedValue = computed(() =>
    calculateSomething(config.value)
  )

  return {
    processData,
    derivedValue
  }
}
```

---

## 分離時の注意点

### 外部インターフェース維持

| ルール | 説明 |
|--------|------|
| Props維持 | 親コンポーネントのpropsを変更しない |
| Emits維持 | 親コンポーネントのemitsを変更しない |
| 下位互換性 | リファクタリングで外部挙動を変えない |

### スタイルの扱い

| パターン | 用途 |
|----------|------|
| scoped style | コンポーネント固有スタイル |
| 共通スタイル | `assets/css/` に抽出 |
| CSS変数 | テーマ対応が必要な場合 |

---

## 検証方法

1. **TypeScript**: `npm run check` でエラーなし
2. **視覚テスト**: ブラウザで表示確認
3. **機能テスト**: 操作が正常に動作

---

## 参照

- [docs/FRONTEND.md](../FRONTEND.md) - フロントエンド全般
- [.claude/rules/frontend.md](../../.claude/rules/frontend.md) - フロントエンドルール
