# ブロック設定ブラッシュアップ案

**Status**: 📋 設計中
**Created**: 2026-01-12
**Related Documents**:
- [UNIFIED_BLOCK_MODEL.md](./UNIFIED_BLOCK_MODEL.md)
- [BLOCK_REGISTRY.md](../BLOCK_REGISTRY.md)
- [FRONTEND.md](../FRONTEND.md)

---

## 1. 現状分析

### 1.1 現在の問題点

| # | 問題 | 影響度 | 詳細 |
|---|------|--------|------|
| 1 | **configSchemaが未活用** | 🔴 高 | DBに保存されているがフロントエンドで使用されていない |
| 2 | **UIがハードコード化** | 🔴 高 | PropertiesPanel.vueが1,956行。新ブロック追加時にコード変更必須 |
| 3 | **動的フォーム生成がない** | 🔴 高 | JSON Schemaから自動的にフォームを生成する仕組みが不在 |
| 4 | **型定義が貧弱** | 🟡 中 | 単純な型のみ。複雑な入力タイプ（配列、条件付き等）が困難 |
| 5 | **バリデーション不足** | 🟡 中 | クライアント側でJSON Schemaベースの検証がない |
| 6 | **ui_config未活用** | 🟡 中 | アイコン・色のみ使用。configSchema部分は未使用 |

### 1.2 現在のアーキテクチャ

```
┌─────────────────────────────────────────────────────────────┐
│  PropertiesPanel.vue (1,956行)                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  v-if="formType === 'llm'"     → LLM設定セクション    │   │
│  │  v-if="formType === 'tool'"    → ツール設定セクション  │   │
│  │  v-if="formType === 'condition'"→ 条件設定セクション  │   │
│  │  ... (18ブロックタイプ分)                            │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                           ↑
                    ハードコード化
                           ↓
┌─────────────────────────────────────────────────────────────┐
│  block_definitions テーブル                                 │
│  ├── config_schema (JSONB) ← 未使用                        │
│  └── ui_config (JSONB)     ← icon/colorのみ使用            │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. 改善案

### 2.1 目標アーキテクチャ

```
┌─────────────────────────────────────────────────────────────┐
│  PropertiesPanel.vue (軽量化)                               │
│  ├── 共通ヘッダー（ブロック名、説明）                         │
│  └── <DynamicConfigForm :schema="configSchema" />          │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│  DynamicConfigForm.vue (新規)                               │
│  ├── JSON Schema解析                                        │
│  ├── **型推論によるウィジェット自動選択**                     │
│  ├── ajvによるバリデーション                                │
│  └── 条件付きフィールド表示                                  │
└─────────────────────────────────────────────────────────────┘
                           ↑
              標準JSON Schemaのみで動作
                           ↓
┌─────────────────────────────────────────────────────────────┐
│  block_definitions テーブル                                 │
│  └── config_schema (標準JSON Schema)                        │
│      ├── type, enum, minimum, maximum 等                   │
│      ├── title, description (ラベル・説明に自動利用)         │
│      └── format (uri, email等の標準フォーマット)             │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 設計方針: 標準JSON Schema優先

**カスタムブロック作成のユーザビリティを考慮し、標準JSON Schemaだけで基本的なフォームが生成される設計とする。**

#### 基本原則

| 優先度 | 方針 |
|--------|------|
| 1 | 標準JSON Schemaのみで基本フォームが動作 |
| 2 | 型から適切なウィジェットを自動推論 |
| 3 | `title`/`description`をラベル・説明に自動利用 |
| 4 | 拡張プロパティは使わなくても問題なし |

#### カスタムブロック作成例（最小構成）

ユーザーがカスタムブロックを作成する際は、標準JSON Schemaだけで十分：

```json
{
  "type": "object",
  "properties": {
    "webhook_url": {
      "type": "string",
      "format": "uri",
      "title": "Webhook URL",
      "description": "通知先のWebhook URL"
    },
    "message": {
      "type": "string",
      "title": "メッセージ",
      "maxLength": 2000
    },
    "retry_count": {
      "type": "integer",
      "title": "リトライ回数",
      "minimum": 0,
      "maximum": 5,
      "default": 3
    },
    "enabled": {
      "type": "boolean",
      "title": "有効化",
      "default": true
    }
  },
  "required": ["webhook_url", "message"]
}
```

上記スキーマから自動生成されるUI：
- `webhook_url` → URL入力フィールド（`format: uri`から推論）
- `message` → テキストエリア（`maxLength`が長いstringから推論）
- `retry_count` → 数値入力（`type: integer`から推論）
- `enabled` → チェックボックス（`type: boolean`から推論）

### 2.3 型推論ルール

標準JSON Schemaの属性からウィジェットを自動決定：

| JSON Schema | 推論されるウィジェット |
|-------------|----------------------|
| `type: "string"` | テキスト入力 |
| `type: "string"` + `enum` | セレクトボックス |
| `type: "string"` + `maxLength > 100` | テキストエリア |
| `type: "string"` + `format: "uri"` | URL入力 |
| `type: "string"` + `format: "date-time"` | 日時ピッカー |
| `type: "number"` / `type: "integer"` | 数値入力 |
| `type: "boolean"` | チェックボックス |
| `type: "array"` | 配列エディタ |
| `type: "object"` | ネストフォーム |

### 2.4 標準属性の活用

| JSON Schema属性 | UI上の用途 |
|----------------|-----------|
| `title` | フィールドラベル（なければプロパティ名を表示） |
| `description` | ヘルプテキスト |
| `default` | 初期値 |
| `enum` | 選択肢 |
| `minimum` / `maximum` | 入力制限 |
| `minLength` / `maxLength` | 文字数制限 |
| `pattern` | 正規表現バリデーション |
| `format` | 入力タイプのヒント（uri, email, date-time等） |

### 2.5 オプション: UI拡張（上級者向け）

システムブロックや高度なカスタマイズが必要な場合のみ、`ui_config`で追加設定可能：

```json
{
  "ui_config": {
    "icon": "send",
    "color": "#5865F2",
    "fieldOverrides": {
      "message": {
        "widget": "template-editor",
        "rows": 6
      }
    },
    "groups": [
      { "id": "basic", "title": "基本設定" },
      { "id": "advanced", "title": "詳細設定", "collapsed": true }
    ],
    "fieldGroups": {
      "webhook_url": "basic",
      "message": "basic",
      "retry_count": "advanced"
    }
  }
}
```

**重要**: `ui_config`は完全にオプショナル。指定しなくても標準JSON Schemaから適切なUIが生成される。

### 2.6 条件付きフィールド表示

JSON Schemaの`if`/`then`/`else`または`allOf`+`if`で実現可能（標準仕様）：

```json
{
  "type": "object",
  "properties": {
    "loop_type": {
      "type": "string",
      "title": "ループタイプ",
      "enum": ["for", "forEach", "while"]
    },
    "count": {
      "type": "integer",
      "title": "繰り返し回数"
    },
    "input_path": {
      "type": "string",
      "title": "入力パス"
    }
  },
  "allOf": [
    {
      "if": { "properties": { "loop_type": { "const": "for" } } },
      "then": { "required": ["count"] }
    },
    {
      "if": { "properties": { "loop_type": { "const": "forEach" } } },
      "then": { "required": ["input_path"] }
    }
  ]
}
```

フロントエンドは`required`になっていないフィールドを折りたたみ表示または非表示にすることで、条件付き表示を実現。

---

## 3. 実装計画

### Phase 1: 基盤整備（優先度：高）

#### 3.1.1 DynamicConfigFormコンポーネント作成

```
frontend/components/workflow-editor/config/
├── DynamicConfigForm.vue       # メインコンポーネント
├── ConfigFieldRenderer.vue     # フィールドレンダラー
├── widgets/
│   ├── TextWidget.vue
│   ├── TextareaWidget.vue
│   ├── NumberWidget.vue
│   ├── SliderWidget.vue
│   ├── SelectWidget.vue
│   ├── CheckboxWidget.vue
│   ├── CodeWidget.vue
│   ├── TemplateEditorWidget.vue
│   ├── ArrayWidget.vue
│   └── ObjectWidget.vue
├── composables/
│   ├── useSchemaParser.ts      # スキーマ解析
│   ├── useValidation.ts        # ajvバリデーション
│   └── useConditionalFields.ts # 条件付き表示
└── types/
    └── config-schema.ts        # 型定義
```

#### 3.1.2 型定義

```typescript
// frontend/components/workflow-editor/config/types/config-schema.ts

// 標準JSON Schema型定義（シンプルに保つ）
export interface JSONSchemaProperty {
  type: 'string' | 'number' | 'integer' | 'boolean' | 'array' | 'object';
  title?: string;           // フィールドラベル
  description?: string;     // ヘルプテキスト
  default?: unknown;
  enum?: (string | number)[];
  const?: string | number | boolean;

  // 数値制約
  minimum?: number;
  maximum?: number;

  // 文字列制約
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  format?: 'uri' | 'email' | 'date-time' | 'date' | 'time' | 'uuid';

  // 配列
  items?: JSONSchemaProperty;
  minItems?: number;
  maxItems?: number;

  // オブジェクト
  properties?: Record<string, JSONSchemaProperty>;
  required?: string[];
  additionalProperties?: boolean | JSONSchemaProperty;
}

export interface ConfigSchema {
  type: 'object';
  properties: Record<string, JSONSchemaProperty>;
  required?: string[];
  allOf?: ConditionalSchema[];
}

// 条件付きスキーマ（標準JSON Schema）
export interface ConditionalSchema {
  if?: { properties: Record<string, { const: unknown }> };
  then?: { required?: string[] };
  else?: { required?: string[] };
}

// UI設定（オプショナル、ui_configに格納）
export interface UIConfig {
  icon?: string;
  color?: string;
  fieldOverrides?: Record<string, FieldOverride>;
  groups?: UIGroup[];
  fieldGroups?: Record<string, string>;
}

export interface FieldOverride {
  widget?: 'textarea' | 'code' | 'template-editor' | 'slider' | 'secret' | 'key-value';
  rows?: number;
  language?: string;
}

export interface UIGroup {
  id: string;
  title: string;
  collapsed?: boolean;
}
```

### Phase 2: コアウィジェット実装（優先度：高）

| ウィジェット | 複雑度 | 依存関係 |
|-------------|--------|---------|
| TextWidget | 低 | - |
| TextareaWidget | 低 | - |
| NumberWidget | 低 | - |
| SelectWidget | 中 | DependsOn対応 |
| CheckboxWidget | 低 | - |
| SliderWidget | 中 | - |
| CodeWidget | 高 | monaco-editor |
| ArrayWidget | 高 | 再帰レンダリング |
| ObjectWidget | 高 | 再帰レンダリング |

### Phase 3: PropertiesPanel統合（優先度：高）

```vue
<!-- PropertiesPanel.vue (リファクタリング後) -->
<template>
  <div class="properties-panel">
    <!-- 共通ヘッダー -->
    <div class="panel-header">
      <input v-model="nodeName" class="node-name-input" />
      <p class="node-description">{{ blockDefinition?.description }}</p>
    </div>

    <!-- 動的フォーム -->
    <DynamicConfigForm
      v-if="configSchema"
      :schema="configSchema"
      :value="nodeConfig"
      :variables="availableVariables"
      @update:value="handleConfigUpdate"
      @validation-error="handleValidationError"
    />

    <!-- フォールバック（スキーマがない場合） -->
    <LegacyConfigForm
      v-else
      :type="formType"
      :config="nodeConfig"
      @update="handleConfigUpdate"
    />
  </div>
</template>
```

### Phase 4: マイグレーション（優先度：中）

#### 4.1 既存ブロックのスキーマ定義

各既存ブロック（llm, tool, condition等）のconfigSchemaを拡張形式で再定義：

```sql
-- Example: LLM block schema update
UPDATE block_definitions
SET ui_config = jsonb_set(
  ui_config,
  '{configSchema}',
  '{
    "type": "object",
    "x-ui-groups": [
      {"id": "model", "label": "モデル設定"},
      {"id": "prompt", "label": "プロンプト"},
      {"id": "params", "label": "パラメータ", "collapsed": true}
    ],
    "properties": {
      "provider": {
        "type": "string",
        "enum": ["openai", "anthropic", "mock"],
        "default": "openai",
        "x-ui-widget": "select",
        "x-ui-label": "プロバイダー",
        "x-ui-group": "model",
        "x-ui-order": 1
      }
    },
    "required": ["provider", "model", "user_prompt"]
  }'::jsonb
)
WHERE slug = 'llm';
```

### Phase 5: 高度な機能（優先度：低）

1. **テンプレートエディタ強化**
   - 変数の自動補完
   - シンタックスハイライト
   - プレビュー機能

2. **バリデーション強化**
   - カスタムバリデータ
   - 非同期バリデーション（API呼び出し等）
   - エラーメッセージのローカライズ

3. **スキーマビルダーUI**
   - 管理画面でGUIによるスキーマ定義
   - プレビュー機能

---

## 4. 各ブロックのconfigSchema定義（標準JSON Schemaのみ）

カスタムブロック作成の参考として、システムブロックも標準JSON Schemaのみで定義。

### 4.1 LLMブロック

```json
{
  "type": "object",
  "properties": {
    "provider": {
      "type": "string",
      "title": "プロバイダー",
      "enum": ["openai", "anthropic", "mock"],
      "default": "openai"
    },
    "model": {
      "type": "string",
      "title": "モデル"
    },
    "system_prompt": {
      "type": "string",
      "title": "システムプロンプト",
      "maxLength": 10000
    },
    "user_prompt": {
      "type": "string",
      "title": "ユーザープロンプト",
      "description": "{{変数名}}で入力データを参照可能",
      "maxLength": 50000
    },
    "temperature": {
      "type": "number",
      "title": "Temperature",
      "minimum": 0,
      "maximum": 2,
      "default": 0.7
    },
    "max_tokens": {
      "type": "integer",
      "title": "最大トークン数",
      "minimum": 1,
      "maximum": 128000,
      "default": 4096
    }
  },
  "required": ["provider", "model", "user_prompt"]
}
```

### 4.2 HTTPツールブロック

```json
{
  "type": "object",
  "properties": {
    "url": {
      "type": "string",
      "title": "URL",
      "format": "uri"
    },
    "method": {
      "type": "string",
      "title": "メソッド",
      "enum": ["GET", "POST", "PUT", "PATCH", "DELETE"],
      "default": "GET"
    },
    "headers": {
      "type": "object",
      "title": "ヘッダー",
      "additionalProperties": { "type": "string" }
    },
    "body": {
      "type": "string",
      "title": "リクエストボディ",
      "maxLength": 100000
    },
    "timeout_ms": {
      "type": "integer",
      "title": "タイムアウト (ms)",
      "minimum": 1000,
      "maximum": 300000,
      "default": 30000
    },
    "retry_count": {
      "type": "integer",
      "title": "リトライ回数",
      "minimum": 0,
      "maximum": 5,
      "default": 0
    }
  },
  "required": ["url", "method"]
}
```

### 4.3 Conditionブロック

```json
{
  "type": "object",
  "properties": {
    "expression": {
      "type": "string",
      "title": "条件式",
      "description": "JSONPath式で条件を記述 (例: $.status == \"success\")"
    }
  },
  "required": ["expression"]
}
```

### 4.4 Switchブロック

```json
{
  "type": "object",
  "properties": {
    "expression": {
      "type": "string",
      "title": "評価式",
      "description": "分岐の基準となる値 (例: $.category)"
    },
    "cases": {
      "type": "array",
      "title": "分岐条件",
      "items": {
        "type": "object",
        "properties": {
          "name": { "type": "string", "title": "ラベル" },
          "expression": { "type": "string", "title": "条件式" },
          "is_default": { "type": "boolean", "title": "デフォルト", "default": false }
        },
        "required": ["name"]
      }
    }
  },
  "required": ["cases"]
}
```

### 4.5 Loopブロック

```json
{
  "type": "object",
  "properties": {
    "loop_type": {
      "type": "string",
      "title": "ループタイプ",
      "enum": ["for", "forEach", "while", "doWhile"],
      "default": "for"
    },
    "count": {
      "type": "integer",
      "title": "繰り返し回数",
      "minimum": 1,
      "default": 10
    },
    "input_path": {
      "type": "string",
      "title": "入力パス",
      "description": "forEachで使用 (例: $.items)"
    },
    "condition": {
      "type": "string",
      "title": "継続条件",
      "description": "while/doWhileで使用 (例: $.hasMore == true)"
    },
    "max_iterations": {
      "type": "integer",
      "title": "最大繰り返し回数",
      "minimum": 1,
      "maximum": 10000,
      "default": 100
    }
  },
  "required": ["loop_type"],
  "allOf": [
    {
      "if": { "properties": { "loop_type": { "const": "for" } } },
      "then": { "required": ["count"] }
    },
    {
      "if": { "properties": { "loop_type": { "const": "forEach" } } },
      "then": { "required": ["input_path"] }
    }
  ]
}
```

### 4.6 Functionブロック

```json
{
  "type": "object",
  "properties": {
    "code": {
      "type": "string",
      "title": "コード",
      "description": "JavaScriptコードを記述",
      "maxLength": 100000
    },
    "timeout_ms": {
      "type": "integer",
      "title": "タイムアウト (ms)",
      "minimum": 100,
      "maximum": 60000,
      "default": 5000
    }
  },
  "required": ["code"]
}
```

### 4.7 Human-in-the-Loopブロック

```json
{
  "type": "object",
  "properties": {
    "instructions": {
      "type": "string",
      "title": "承認者への指示",
      "maxLength": 5000
    },
    "timeout_hours": {
      "type": "number",
      "title": "タイムアウト (時間)",
      "minimum": 0.1,
      "maximum": 168,
      "default": 24
    },
    "require_comment": {
      "type": "boolean",
      "title": "コメント必須",
      "default": false
    },
    "approval_options": {
      "type": "array",
      "title": "承認オプション",
      "items": {
        "type": "object",
        "properties": {
          "label": { "type": "string", "title": "ラベル" },
          "value": { "type": "string", "title": "値" }
        }
      }
    }
  },
  "required": ["instructions"]
}
```

---

## 5. バリデーション戦略

### 5.1 クライアント側バリデーション

```typescript
// composables/useValidation.ts
import Ajv from 'ajv';
import addFormats from 'ajv-formats';

const ajv = new Ajv({ allErrors: true, verbose: true });
addFormats(ajv);

// カスタムフォーマット追加
ajv.addFormat('jsonpath', /^\$(\.[a-zA-Z_][a-zA-Z0-9_]*|\[\d+\]|\[\*\])*$/);
ajv.addFormat('template', /.*\{\{.*\}\}.*/);

export function useValidation(schema: ConfigSchema) {
  const validate = ajv.compile(schema);

  function validateConfig(config: Record<string, unknown>): ValidationResult {
    const valid = validate(config);
    if (valid) {
      return { valid: true, errors: [] };
    }

    return {
      valid: false,
      errors: validate.errors?.map(err => ({
        field: err.instancePath.slice(1) || err.params.missingProperty,
        message: formatErrorMessage(err),
        keyword: err.keyword
      })) || []
    };
  }

  return { validateConfig };
}
```

### 5.2 サーバー側バリデーション

バックエンドでも同様のJSON Schemaバリデーションを実施：

```go
// backend/internal/usecase/step_usecase.go
func (u *StepUsecase) ValidateStepConfig(ctx context.Context, blockSlug string, config json.RawMessage) error {
    block, err := u.blockRepo.GetBySlug(ctx, blockSlug)
    if err != nil {
        return err
    }

    // JSON Schema validation using gojsonschema
    schemaLoader := gojsonschema.NewBytesLoader(block.ConfigSchema)
    documentLoader := gojsonschema.NewBytesLoader(config)

    result, err := gojsonschema.Validate(schemaLoader, documentLoader)
    if err != nil {
        return err
    }

    if !result.Valid() {
        return &ValidationError{Errors: result.Errors()}
    }

    return nil
}
```

---

## 6. 移行計画

### 6.1 段階的移行

| 段階 | 内容 | 影響 |
|------|------|------|
| **Phase 1** | DynamicConfigFormコンポーネント作成 | なし（新規追加） |
| **Phase 2** | 基本ウィジェット実装 | なし |
| **Phase 3** | 1-2ブロックで試験導入 | 限定的 |
| **Phase 4** | 全ブロックのスキーマ定義 | DBマイグレーション |
| **Phase 5** | PropertiesPanel統合 | フロントエンド変更 |
| **Phase 6** | レガシーコード削除 | 大規模 |

### 6.2 フォールバック戦略

```vue
<template>
  <!-- configSchemaがある場合は動的フォーム -->
  <DynamicConfigForm
    v-if="hasConfigSchema"
    :schema="configSchema"
    :value="config"
    @update:value="$emit('update:config', $event)"
  />

  <!-- ない場合は従来のハードコードUIにフォールバック -->
  <LegacyConfigForm
    v-else
    :type="blockType"
    :config="config"
    @update:config="$emit('update:config', $event)"
  />
</template>
```

---

## 7. 期待される効果

| 項目 | Before | After |
|------|--------|-------|
| **新規ブロック追加** | フロントエンドコード変更必須 | SQLマイグレーションのみ |
| **PropertiesPanel.vue** | 1,956行 | 〜300行（推定） |
| **バリデーション** | 手動・部分的 | JSON Schema準拠・完全 |
| **型安全性** | 低い | TypeScript型自動生成可能 |
| **保守性** | 低い | 高い（スキーマ駆動） |
| **カスタムブロック** | 管理者がUI定義不可 | configSchemaで完全定義可能 |

---

## 8. 次のアクション

1. [ ] Phase 1: DynamicConfigFormコンポーネント骨組み作成
2. [ ] Phase 2: TextWidget, SelectWidget, NumberWidget実装
3. [ ] Phase 3: LLMブロックで試験導入
4. [ ] フィードバック収集・改善
5. [ ] 残りのウィジェット実装
6. [ ] 全ブロックのスキーマ定義
