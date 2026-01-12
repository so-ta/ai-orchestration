-- Migration: 013_llm_block_config_schema.sql
-- Purpose: Update LLM block with rich config_schema for DynamicConfigForm
-- See: docs/designs/BLOCK_CONFIG_IMPROVEMENT.md

-- Update LLM block config_schema with proper JSON Schema
UPDATE block_definitions
SET config_schema = '{
  "type": "object",
  "properties": {
    "provider": {
      "type": "string",
      "title": "プロバイダー",
      "description": "使用するLLMプロバイダーを選択",
      "enum": ["openai", "anthropic", "mock"],
      "default": "openai"
    },
    "model": {
      "type": "string",
      "title": "モデル",
      "description": "使用するモデル名"
    },
    "system_prompt": {
      "type": "string",
      "title": "システムプロンプト",
      "description": "AIの振る舞いを定義するプロンプト",
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
      "description": "出力の多様性を制御（0: 決定的、2: 創造的）",
      "minimum": 0,
      "maximum": 2,
      "default": 0.7
    },
    "max_tokens": {
      "type": "integer",
      "title": "最大トークン数",
      "description": "生成する最大トークン数",
      "minimum": 1,
      "maximum": 128000,
      "default": 4096
    }
  },
  "required": ["provider", "model", "user_prompt"]
}'::jsonb,
ui_config = '{
  "icon": "brain",
  "color": "#8B5CF6",
  "fieldOverrides": {
    "system_prompt": {
      "widget": "textarea",
      "rows": 4
    },
    "user_prompt": {
      "widget": "textarea",
      "rows": 8
    }
  },
  "groups": [
    { "id": "model", "title": "モデル設定", "icon": "🤖" },
    { "id": "prompt", "title": "プロンプト", "icon": "💬" },
    { "id": "params", "title": "パラメータ", "icon": "⚙️", "collapsed": true }
  ],
  "fieldGroups": {
    "provider": "model",
    "model": "model",
    "system_prompt": "prompt",
    "user_prompt": "prompt",
    "temperature": "params",
    "max_tokens": "params"
  }
}'::jsonb
WHERE slug = 'llm' AND tenant_id IS NULL;
