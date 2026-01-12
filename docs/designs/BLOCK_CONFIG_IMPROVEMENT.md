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
│  ├── フィールドタイプ別レンダリング                          │
│  ├── ajvによるバリデーション                                │
│  └── 条件付きフィールド表示                                  │
└─────────────────────────────────────────────────────────────┘
                           ↑
                    スキーマ駆動
                           ↓
┌─────────────────────────────────────────────────────────────┐
│  block_definitions テーブル                                 │
│  └── ui_config.configSchema (拡張JSON Schema)              │
│      ├── 標準JSON Schema (type, enum, minimum等)           │
│      └── UI拡張 (x-ui-*)                                   │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 拡張configSchemaフォーマット

標準JSON Schemaに`x-ui-*`プレフィックスでUI制御を追加：

```json
{
  "type": "object",
  "properties": {
    "provider": {
      "type": "string",
      "enum": ["openai", "anthropic", "mock"],
      "default": "openai",
      "x-ui-widget": "select",
      "x-ui-label": "プロバイダー",
      "x-ui-description": "使用するLLMプロバイダーを選択",
      "x-ui-order": 1
    },
    "model": {
      "type": "string",
      "x-ui-widget": "select",
      "x-ui-label": "モデル",
      "x-ui-order": 2,
      "x-ui-depends-on": {
        "field": "provider",
        "options": {
          "openai": ["gpt-4o", "gpt-4o-mini", "gpt-4-turbo"],
          "anthropic": ["claude-sonnet-4-20250514", "claude-3-5-haiku-20241022"],
          "mock": ["mock-model"]
        }
      }
    },
    "system_prompt": {
      "type": "string",
      "x-ui-widget": "textarea",
      "x-ui-label": "システムプロンプト",
      "x-ui-rows": 4,
      "x-ui-placeholder": "AIの役割を定義...",
      "x-ui-order": 3
    },
    "user_prompt": {
      "type": "string",
      "x-ui-widget": "template-editor",
      "x-ui-label": "ユーザープロンプト",
      "x-ui-description": "{{変数名}}で入力データを参照可能",
      "x-ui-rows": 6,
      "x-ui-order": 4
    },
    "temperature": {
      "type": "number",
      "minimum": 0,
      "maximum": 2,
      "default": 0.7,
      "x-ui-widget": "slider",
      "x-ui-label": "Temperature",
      "x-ui-step": 0.1,
      "x-ui-order": 5
    },
    "max_tokens": {
      "type": "integer",
      "minimum": 1,
      "maximum": 128000,
      "default": 4096,
      "x-ui-widget": "number",
      "x-ui-label": "最大トークン数",
      "x-ui-order": 6,
      "x-ui-collapsed": true
    }
  },
  "required": ["provider", "model", "user_prompt"]
}
```

### 2.3 サポートするUIウィジェット

| ウィジェット | 用途 | オプション |
|-------------|------|-----------|
| `text` | 単一行テキスト | `placeholder`, `maxLength` |
| `textarea` | 複数行テキスト | `rows`, `placeholder` |
| `number` | 数値入力 | `step`, `min`, `max` |
| `slider` | スライダー | `step`, `min`, `max`, `showValue` |
| `select` | ドロップダウン | `options`, `allowCustom` |
| `radio` | ラジオボタン | `options`, `inline` |
| `checkbox` | チェックボックス | - |
| `switch` | トグルスイッチ | - |
| `color` | カラーピッカー | `presets` |
| `datetime` | 日時選択 | `type` (date/time/datetime) |
| `code` | コードエディタ | `language`, `height` |
| `template-editor` | テンプレートエディタ | `variables`, `rows` |
| `json` | JSONエディタ | `schema` |
| `key-value` | キー・バリューペア | `keyLabel`, `valueLabel` |
| `array` | 配列エディタ | `itemSchema`, `addLabel` |
| `object` | ネストオブジェクト | `properties` |
| `secret` | シークレット入力 | `envKey` |

### 2.4 条件付きフィールド表示

```json
{
  "properties": {
    "loop_type": {
      "type": "string",
      "enum": ["for", "forEach", "while", "doWhile"],
      "x-ui-widget": "select",
      "x-ui-label": "ループタイプ"
    },
    "count": {
      "type": "integer",
      "x-ui-widget": "number",
      "x-ui-label": "繰り返し回数",
      "x-ui-visible-if": {
        "field": "loop_type",
        "value": "for"
      }
    },
    "input_path": {
      "type": "string",
      "x-ui-widget": "text",
      "x-ui-label": "入力パス",
      "x-ui-visible-if": {
        "field": "loop_type",
        "value": "forEach"
      }
    },
    "condition": {
      "type": "string",
      "x-ui-widget": "text",
      "x-ui-label": "継続条件",
      "x-ui-visible-if": {
        "field": "loop_type",
        "values": ["while", "doWhile"]
      }
    }
  }
}
```

### 2.5 グループ化とセクション

```json
{
  "x-ui-groups": [
    {
      "id": "basic",
      "label": "基本設定",
      "collapsed": false
    },
    {
      "id": "advanced",
      "label": "詳細設定",
      "collapsed": true
    }
  ],
  "properties": {
    "provider": {
      "x-ui-group": "basic",
      "x-ui-order": 1
    },
    "model": {
      "x-ui-group": "basic",
      "x-ui-order": 2
    },
    "temperature": {
      "x-ui-group": "advanced",
      "x-ui-order": 1
    },
    "max_tokens": {
      "x-ui-group": "advanced",
      "x-ui-order": 2
    }
  }
}
```

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

export type UIWidget =
  | 'text'
  | 'textarea'
  | 'number'
  | 'slider'
  | 'select'
  | 'radio'
  | 'checkbox'
  | 'switch'
  | 'color'
  | 'datetime'
  | 'code'
  | 'template-editor'
  | 'json'
  | 'key-value'
  | 'array'
  | 'object'
  | 'secret';

export interface UIExtensions {
  'x-ui-widget'?: UIWidget;
  'x-ui-label'?: string;
  'x-ui-description'?: string;
  'x-ui-placeholder'?: string;
  'x-ui-order'?: number;
  'x-ui-group'?: string;
  'x-ui-collapsed'?: boolean;
  'x-ui-rows'?: number;
  'x-ui-step'?: number;
  'x-ui-visible-if'?: VisibilityCondition;
  'x-ui-depends-on'?: DependsOnConfig;
}

export interface VisibilityCondition {
  field: string;
  value?: string | number | boolean;
  values?: (string | number | boolean)[];
  operator?: 'eq' | 'ne' | 'in' | 'notIn' | 'gt' | 'lt';
}

export interface DependsOnConfig {
  field: string;
  options: Record<string, string[]>;
}

export interface ConfigSchemaProperty extends UIExtensions {
  type: 'string' | 'number' | 'integer' | 'boolean' | 'array' | 'object';
  enum?: (string | number)[];
  default?: unknown;
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  items?: ConfigSchemaProperty;
  properties?: Record<string, ConfigSchemaProperty>;
  required?: string[];
}

export interface ConfigSchema {
  type: 'object';
  properties: Record<string, ConfigSchemaProperty>;
  required?: string[];
  'x-ui-groups'?: UIGroup[];
}

export interface UIGroup {
  id: string;
  label: string;
  collapsed?: boolean;
  icon?: string;
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

## 4. 各ブロックのconfigSchema定義

### 4.1 LLMブロック

```json
{
  "type": "object",
  "x-ui-groups": [
    { "id": "model", "label": "モデル設定" },
    { "id": "prompt", "label": "プロンプト" },
    { "id": "params", "label": "パラメータ", "collapsed": true }
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
    },
    "model": {
      "type": "string",
      "x-ui-widget": "select",
      "x-ui-label": "モデル",
      "x-ui-group": "model",
      "x-ui-order": 2,
      "x-ui-depends-on": {
        "field": "provider",
        "options": {
          "openai": ["gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"],
          "anthropic": ["claude-sonnet-4-20250514", "claude-3-5-haiku-20241022", "claude-3-opus-20240229"],
          "mock": ["mock-model"]
        }
      }
    },
    "system_prompt": {
      "type": "string",
      "x-ui-widget": "textarea",
      "x-ui-label": "システムプロンプト",
      "x-ui-group": "prompt",
      "x-ui-rows": 4,
      "x-ui-order": 1
    },
    "user_prompt": {
      "type": "string",
      "x-ui-widget": "template-editor",
      "x-ui-label": "ユーザープロンプト",
      "x-ui-description": "{{変数名}}で入力データを参照可能",
      "x-ui-group": "prompt",
      "x-ui-rows": 6,
      "x-ui-order": 2
    },
    "temperature": {
      "type": "number",
      "minimum": 0,
      "maximum": 2,
      "default": 0.7,
      "x-ui-widget": "slider",
      "x-ui-label": "Temperature",
      "x-ui-step": 0.1,
      "x-ui-group": "params",
      "x-ui-order": 1
    },
    "max_tokens": {
      "type": "integer",
      "minimum": 1,
      "maximum": 128000,
      "default": 4096,
      "x-ui-widget": "number",
      "x-ui-label": "最大トークン数",
      "x-ui-group": "params",
      "x-ui-order": 2
    }
  },
  "required": ["provider", "model", "user_prompt"]
}
```

### 4.2 HTTPツールブロック

```json
{
  "type": "object",
  "x-ui-groups": [
    { "id": "request", "label": "リクエスト設定" },
    { "id": "auth", "label": "認証", "collapsed": true },
    { "id": "advanced", "label": "詳細設定", "collapsed": true }
  ],
  "properties": {
    "url": {
      "type": "string",
      "format": "uri",
      "x-ui-widget": "text",
      "x-ui-label": "URL",
      "x-ui-placeholder": "https://api.example.com/endpoint",
      "x-ui-group": "request",
      "x-ui-order": 1
    },
    "method": {
      "type": "string",
      "enum": ["GET", "POST", "PUT", "PATCH", "DELETE"],
      "default": "GET",
      "x-ui-widget": "select",
      "x-ui-label": "メソッド",
      "x-ui-group": "request",
      "x-ui-order": 2
    },
    "headers": {
      "type": "object",
      "additionalProperties": { "type": "string" },
      "x-ui-widget": "key-value",
      "x-ui-label": "ヘッダー",
      "x-ui-group": "request",
      "x-ui-order": 3
    },
    "body": {
      "type": "string",
      "x-ui-widget": "json",
      "x-ui-label": "リクエストボディ",
      "x-ui-group": "request",
      "x-ui-order": 4,
      "x-ui-visible-if": {
        "field": "method",
        "values": ["POST", "PUT", "PATCH"]
      }
    },
    "auth_type": {
      "type": "string",
      "enum": ["none", "bearer", "basic", "api_key"],
      "default": "none",
      "x-ui-widget": "select",
      "x-ui-label": "認証タイプ",
      "x-ui-group": "auth",
      "x-ui-order": 1
    },
    "auth_token": {
      "type": "string",
      "x-ui-widget": "secret",
      "x-ui-label": "トークン",
      "x-ui-group": "auth",
      "x-ui-order": 2,
      "x-ui-visible-if": {
        "field": "auth_type",
        "values": ["bearer", "api_key"]
      }
    },
    "timeout_ms": {
      "type": "integer",
      "minimum": 1000,
      "maximum": 300000,
      "default": 30000,
      "x-ui-widget": "number",
      "x-ui-label": "タイムアウト (ms)",
      "x-ui-group": "advanced",
      "x-ui-order": 1
    },
    "retry_count": {
      "type": "integer",
      "minimum": 0,
      "maximum": 5,
      "default": 0,
      "x-ui-widget": "number",
      "x-ui-label": "リトライ回数",
      "x-ui-group": "advanced",
      "x-ui-order": 2
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
      "x-ui-widget": "code",
      "x-ui-label": "条件式",
      "x-ui-description": "JSONPath式で条件を記述 (例: $.status == \"success\")",
      "x-ui-language": "jsonpath",
      "x-ui-order": 1
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
      "x-ui-widget": "text",
      "x-ui-label": "評価式",
      "x-ui-description": "分岐の基準となる値 (例: $.category)",
      "x-ui-order": 1
    },
    "cases": {
      "type": "array",
      "x-ui-widget": "array",
      "x-ui-label": "分岐条件",
      "x-ui-add-label": "条件を追加",
      "x-ui-order": 2,
      "items": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "x-ui-widget": "text",
            "x-ui-label": "ラベル"
          },
          "expression": {
            "type": "string",
            "x-ui-widget": "text",
            "x-ui-label": "条件式"
          },
          "is_default": {
            "type": "boolean",
            "default": false,
            "x-ui-widget": "checkbox",
            "x-ui-label": "デフォルト"
          }
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
  "x-ui-groups": [
    { "id": "type", "label": "ループタイプ" },
    { "id": "settings", "label": "設定" }
  ],
  "properties": {
    "loop_type": {
      "type": "string",
      "enum": ["for", "forEach", "while", "doWhile"],
      "default": "for",
      "x-ui-widget": "radio",
      "x-ui-label": "ループタイプ",
      "x-ui-group": "type",
      "x-ui-inline": true,
      "x-ui-order": 1
    },
    "count": {
      "type": "integer",
      "minimum": 1,
      "default": 10,
      "x-ui-widget": "number",
      "x-ui-label": "繰り返し回数",
      "x-ui-group": "settings",
      "x-ui-order": 1,
      "x-ui-visible-if": {
        "field": "loop_type",
        "value": "for"
      }
    },
    "input_path": {
      "type": "string",
      "x-ui-widget": "text",
      "x-ui-label": "入力パス",
      "x-ui-placeholder": "$.items",
      "x-ui-group": "settings",
      "x-ui-order": 2,
      "x-ui-visible-if": {
        "field": "loop_type",
        "value": "forEach"
      }
    },
    "condition": {
      "type": "string",
      "x-ui-widget": "text",
      "x-ui-label": "継続条件",
      "x-ui-placeholder": "$.hasMore == true",
      "x-ui-group": "settings",
      "x-ui-order": 3,
      "x-ui-visible-if": {
        "field": "loop_type",
        "values": ["while", "doWhile"]
      }
    },
    "max_iterations": {
      "type": "integer",
      "minimum": 1,
      "maximum": 10000,
      "default": 100,
      "x-ui-widget": "number",
      "x-ui-label": "最大繰り返し回数",
      "x-ui-group": "settings",
      "x-ui-order": 4
    }
  },
  "required": ["loop_type"]
}
```

### 4.6 Functionブロック

```json
{
  "type": "object",
  "properties": {
    "code": {
      "type": "string",
      "x-ui-widget": "code",
      "x-ui-label": "コード",
      "x-ui-language": "javascript",
      "x-ui-height": "300px",
      "x-ui-order": 1
    },
    "timeout_ms": {
      "type": "integer",
      "minimum": 100,
      "maximum": 60000,
      "default": 5000,
      "x-ui-widget": "number",
      "x-ui-label": "タイムアウト (ms)",
      "x-ui-order": 2
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
      "x-ui-widget": "textarea",
      "x-ui-label": "承認者への指示",
      "x-ui-rows": 4,
      "x-ui-order": 1
    },
    "timeout_hours": {
      "type": "number",
      "minimum": 0.1,
      "maximum": 168,
      "default": 24,
      "x-ui-widget": "number",
      "x-ui-label": "タイムアウト (時間)",
      "x-ui-step": 0.5,
      "x-ui-order": 2
    },
    "require_comment": {
      "type": "boolean",
      "default": false,
      "x-ui-widget": "checkbox",
      "x-ui-label": "コメント必須",
      "x-ui-order": 3
    },
    "approval_options": {
      "type": "array",
      "x-ui-widget": "array",
      "x-ui-label": "承認オプション",
      "x-ui-add-label": "オプションを追加",
      "x-ui-order": 4,
      "items": {
        "type": "object",
        "properties": {
          "label": {
            "type": "string",
            "x-ui-widget": "text",
            "x-ui-label": "ラベル"
          },
          "value": {
            "type": "string",
            "x-ui-widget": "text",
            "x-ui-label": "値"
          },
          "color": {
            "type": "string",
            "x-ui-widget": "color",
            "x-ui-label": "色"
          }
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
