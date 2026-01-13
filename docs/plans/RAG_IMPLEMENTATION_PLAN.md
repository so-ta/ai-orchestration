# RAG (Retrieval-Augmented Generation) 実装プラン

## ステータス: ✅ Phase 1-3 完了

**作成日**: 2026-01-13
**目的**: LangChainスタイルのRAGワークフローをAI Orchestrationに実装

---

## 1. 概要

### 1.1 目標

LangChainのRAGアーキテクチャを参考に、以下を実現：

- **コンポーネント分離**: Document Loader → Text Splitter → Embedding → Vector Store → Retriever
- **柔軟なワークフロー構築**: 各ブロックを自由に組み合わせ可能
- **統合ブロック**: 簡易RAGを1ブロックで実現
- **マルチテナント分離**: テナント間のデータ分離を厳格に実装

### 1.2 設計原則

| 原則 | 説明 |
|------|------|
| **LangChain互換** | Document型、チェーン構造をLangChainに合わせる |
| **テナント分離** | ベクトルDB・コレクションレベルでの完全分離 |
| **段階的実装** | 基本機能から開始、高度な機能は後続フェーズ |
| **既存パターン準拠** | Unified Block Model、ctx API拡張パターンに従う |

---

## 2. アーキテクチャ

### 2.1 全体構成

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI Orchestration RAG                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ctx.embedding                                                   │
│  └── embed(provider, model, texts[]) → vectors[]                │
│                                                                  │
│  ctx.vector (テナント分離を強制)                                 │
│  ├── upsert(collection, documents[])                            │
│  ├── query(collection, vector, options) → results               │
│  └── delete(collection, ids[])                                  │
│      └─→ 内部で tenant_id を自動付与、他テナントアクセス不可    │
│                                                                  │
│  RAG Blocks                                                      │
│  ├── doc-loader      - ドキュメント読み込み                      │
│  ├── text-splitter   - チャンク分割                             │
│  ├── embedding       - ベクトル化                               │
│  ├── vector-upsert   - ベクトル保存                             │
│  ├── vector-search   - ベクトル検索                             │
│  ├── vector-delete   - ベクトル削除                             │
│  └── rag-query       - 統合RAGクエリ                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 マルチテナント分離設計

**⚠️ 最重要: テナント分離の実装**

```
┌─────────────────────────────────────────────────────────────────┐
│                    テナント分離アーキテクチャ                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ユーザーリクエスト                                              │
│       ↓                                                          │
│  ctx.vector.query("my-docs", vector, {top_k: 5})                │
│       ↓                                                          │
│  ┌─────────────────────────────────────────┐                    │
│  │  VectorService (Go)                     │                    │
│  │  ├── tenant_id を ExecutionContext から取得                  │
│  │  ├── collection → {tenant_id}_{collection} に変換           │
│  │  └── すべてのクエリに tenant_id フィルタを強制               │
│  └─────────────────────────────────────────┘                    │
│       ↓                                                          │
│  ┌─────────────────────────────────────────┐                    │
│  │  pgvector / 外部ベクトルDB              │                    │
│  │  WHERE tenant_id = $1 AND collection = $2                   │
│  └─────────────────────────────────────────┘                    │
│                                                                  │
│  ❌ 他テナントのデータにはアクセス不可                           │
│  ❌ collection名の直接指定でもテナント境界を越えられない         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**分離の実装箇所**:

| 層 | 分離方法 |
|----|---------|
| **ctx API層** | tenant_idをExecutionContextから取得（ユーザー指定不可） |
| **VectorService層** | collection名に tenant_id プレフィックス強制 |
| **DB層** | 全クエリに `WHERE tenant_id = $1` 強制 |
| **pgvector** | `tenant_id` カラム + インデックス |
| **外部VectorDB** | namespace/collection = `{tenant_id}_{collection}` |

---

## 3. データベーススキーマ

### 3.1 vector_collections テーブル（コレクション管理）

```sql
-- ベクトルコレクションのメタデータ管理
CREATE TABLE vector_collections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    embedding_provider VARCHAR(50) NOT NULL DEFAULT 'openai',
    embedding_model VARCHAR(100) NOT NULL DEFAULT 'text-embedding-3-small',
    dimension INTEGER NOT NULL DEFAULT 1536,
    document_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_collection_per_tenant UNIQUE (tenant_id, name)
);

CREATE INDEX idx_vector_collections_tenant ON vector_collections(tenant_id);
```

### 3.2 vector_documents テーブル（ドキュメント+ベクトル）

```sql
-- pgvector拡張の有効化
CREATE EXTENSION IF NOT EXISTS vector;

-- ベクトルドキュメント（チャンク単位）
CREATE TABLE vector_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    collection_id UUID NOT NULL REFERENCES vector_collections(id) ON DELETE CASCADE,

    -- ドキュメントコンテンツ（LangChain Document型に対応）
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',

    -- ベクトル埋め込み
    embedding vector(1536),  -- OpenAI text-embedding-3-small

    -- ソース情報
    source_url TEXT,
    source_type VARCHAR(50),  -- 'url', 'file', 'text', 'api'
    chunk_index INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- テナント分離を強制するインデックス
    CONSTRAINT fk_tenant_collection
        FOREIGN KEY (tenant_id, collection_id)
        REFERENCES vector_collections(tenant_id, id)
);

-- テナント分離 + 類似検索用インデックス
CREATE INDEX idx_vector_documents_tenant_collection
    ON vector_documents(tenant_id, collection_id);

-- ベクトル検索用インデックス（IVFFlat）
CREATE INDEX idx_vector_documents_embedding
    ON vector_documents
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- メタデータ検索用GINインデックス
CREATE INDEX idx_vector_documents_metadata
    ON vector_documents USING GIN (metadata);
```

### 3.3 使用量トラッキング拡張

```sql
-- usage_records に embedding 操作を追加
-- 既存の operation COMMENT を拡張
COMMENT ON COLUMN usage_records.operation IS
    'Operation type: chat, completion, embedding, vector_search, vector_upsert';
```

---

## 4. ctx API設計

### 4.1 ctx.embedding インターフェース

```typescript
interface EmbeddingService {
  /**
   * テキストをベクトル埋め込みに変換
   * @param provider - 'openai' | 'cohere' | 'voyage'
   * @param model - 'text-embedding-3-small' | 'text-embedding-3-large' | etc.
   * @param texts - 埋め込み対象のテキスト（文字列または配列）
   * @returns ベクトル配列と使用量情報
   */
  embed(
    provider: string,
    model: string,
    texts: string | string[]
  ): Promise<{
    vectors: number[][];
    model: string;
    dimension: number;
    usage: { total_tokens: number };
  }>;
}
```

### 4.2 ctx.vector インターフェース

```typescript
interface VectorService {
  /**
   * ドキュメントをコレクションに追加（ベクトル化込み）
   * ⚠️ tenant_id は自動付与、ユーザー指定不可
   */
  upsert(
    collection: string,
    documents: Array<{
      id?: string;
      content: string;
      metadata?: Record<string, any>;
      vector?: number[];  // 省略時は自動ベクトル化
    }>,
    options?: {
      embedding_provider?: string;
      embedding_model?: string;
    }
  ): Promise<{
    upserted_count: number;
    ids: string[];
  }>;

  /**
   * 類似ベクトル検索
   * ⚠️ tenant_id フィルタ自動適用、他テナントデータ参照不可
   */
  query(
    collection: string,
    vector: number[],
    options?: {
      top_k?: number;           // デフォルト: 5
      threshold?: number;       // 類似度閾値
      filter?: Record<string, any>;  // メタデータフィルタ
      include_content?: boolean;     // コンテンツを含めるか
    }
  ): Promise<{
    matches: Array<{
      id: string;
      score: number;
      content?: string;
      metadata?: Record<string, any>;
    }>;
  }>;

  /**
   * ドキュメント削除
   * ⚠️ 自テナントのドキュメントのみ削除可能
   */
  delete(
    collection: string,
    ids: string[]
  ): Promise<{
    deleted_count: number;
  }>;

  /**
   * コレクション一覧取得
   * ⚠️ 自テナントのコレクションのみ返却
   */
  listCollections(): Promise<Array<{
    name: string;
    document_count: number;
    dimension: number;
    created_at: string;
  }>>;
}
```

---

## 5. ブロック定義

### 5.1 doc-loader（ドキュメント読み込み）

```javascript
// Category: data
// Description: URL、テキスト、APIからドキュメントを読み込み

const sourceType = config.source_type || 'url';
let content, metadata;

switch (sourceType) {
  case 'url':
    const url = config.url || input.url;
    const response = await ctx.http.get(url);
    content = typeof response.data === 'string'
      ? response.data
      : JSON.stringify(response.data);
    metadata = {
      source: url,
      source_type: 'url',
      content_type: response.headers['Content-Type'],
      fetched_at: new Date().toISOString()
    };
    break;

  case 'text':
    content = config.content || input.content || input.text;
    metadata = { source_type: 'text' };
    break;

  case 'json':
    const jsonData = input.data || input;
    content = config.content_path
      ? getPath(jsonData, config.content_path)
      : JSON.stringify(jsonData);
    metadata = { source_type: 'json', ...config.metadata };
    break;
}

// HTMLストリッピング（オプション）
if (config.strip_html && content.includes('<')) {
  content = content
    .replace(/<script[^>]*>[\s\S]*?<\/script>/gi, '')
    .replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '')
    .replace(/<[^>]+>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

return {
  documents: [{
    content,
    metadata,
    char_count: content.length
  }]
};
```

**Config Schema**:
```json
{
  "type": "object",
  "properties": {
    "source_type": {
      "type": "string",
      "enum": ["url", "text", "json"],
      "default": "url",
      "title": "ソースタイプ"
    },
    "url": {
      "type": "string",
      "title": "URL",
      "description": "source_type=url の場合に使用"
    },
    "content": {
      "type": "string",
      "title": "テキストコンテンツ",
      "description": "source_type=text の場合に使用"
    },
    "strip_html": {
      "type": "boolean",
      "default": true,
      "title": "HTMLタグを除去"
    }
  }
}
```

### 5.2 text-splitter（チャンク分割）

```javascript
// Category: data
// Description: テキストを指定サイズのチャンクに分割

const documents = input.documents || [{ content: input.content || input.text }];
const chunkSize = config.chunk_size || 1000;
const chunkOverlap = config.chunk_overlap || 200;
const separator = config.separator || '\n\n';

function splitText(text, size, overlap, sep) {
  const chunks = [];
  const segments = text.split(sep);
  let current = '';

  for (const segment of segments) {
    const combined = current ? current + sep + segment : segment;

    if (combined.length > size && current) {
      chunks.push(current.trim());
      // オーバーラップ：前のチャンクの末尾を保持
      const words = current.split(/\s+/);
      const overlapWords = Math.ceil(overlap / 6); // 平均6文字/単語と仮定
      current = words.slice(-overlapWords).join(' ') + sep + segment;
    } else {
      current = combined;
    }
  }

  if (current.trim()) {
    chunks.push(current.trim());
  }

  return chunks;
}

const result = [];

for (const doc of documents) {
  const chunks = splitText(doc.content, chunkSize, chunkOverlap, separator);

  for (let i = 0; i < chunks.length; i++) {
    result.push({
      content: chunks[i],
      metadata: {
        ...doc.metadata,
        chunk_index: i,
        chunk_total: chunks.length
      },
      char_count: chunks[i].length
    });
  }
}

return {
  documents: result,
  chunk_count: result.length,
  original_count: documents.length
};
```

**Config Schema**:
```json
{
  "type": "object",
  "properties": {
    "chunk_size": {
      "type": "integer",
      "default": 1000,
      "minimum": 100,
      "maximum": 8000,
      "title": "チャンクサイズ（文字数）"
    },
    "chunk_overlap": {
      "type": "integer",
      "default": 200,
      "minimum": 0,
      "title": "オーバーラップ（文字数）"
    },
    "separator": {
      "type": "string",
      "default": "\n\n",
      "title": "区切り文字"
    }
  }
}
```

### 5.3 embedding（ベクトル化）

```javascript
// Category: ai
// Description: テキストをベクトル埋め込みに変換

const documents = input.documents || [{ content: input.content || input.text }];
const provider = config.provider || 'openai';
const model = config.model || 'text-embedding-3-small';

// テキスト抽出
const texts = documents.map(d => d.content);

// バッチ埋め込み
const result = await ctx.embedding.embed(provider, model, texts);

// ドキュメントにベクトルを追加
const docsWithVectors = documents.map((doc, i) => ({
  ...doc,
  vector: result.vectors[i]
}));

return {
  documents: docsWithVectors,
  model: result.model,
  dimension: result.dimension,
  usage: result.usage
};
```

**Config Schema**:
```json
{
  "type": "object",
  "properties": {
    "provider": {
      "type": "string",
      "enum": ["openai", "cohere", "voyage"],
      "default": "openai",
      "title": "Embeddingプロバイダー"
    },
    "model": {
      "type": "string",
      "default": "text-embedding-3-small",
      "title": "モデル",
      "description": "OpenAI: text-embedding-3-small, text-embedding-3-large"
    }
  }
}
```

### 5.4 vector-upsert（ベクトル保存）

```javascript
// Category: data
// Description: ドキュメントをベクトルDBに保存

const collection = config.collection || input.collection;
const documents = input.documents;

if (!collection) {
  throw new Error('[VECTOR_001] collection is required');
}

if (!documents || documents.length === 0) {
  throw new Error('[VECTOR_002] documents array is required');
}

// ctx.vector.upsert は tenant_id を自動付与
const result = await ctx.vector.upsert(collection, documents, {
  embedding_provider: config.embedding_provider,
  embedding_model: config.embedding_model
});

return {
  collection,
  upserted_count: result.upserted_count,
  ids: result.ids
};
```

**Config Schema**:
```json
{
  "type": "object",
  "properties": {
    "collection": {
      "type": "string",
      "title": "コレクション名",
      "description": "ベクトルを保存するコレクション"
    },
    "embedding_provider": {
      "type": "string",
      "enum": ["openai", "cohere"],
      "default": "openai",
      "title": "Embeddingプロバイダー",
      "description": "ベクトルが未設定の場合に使用"
    },
    "embedding_model": {
      "type": "string",
      "default": "text-embedding-3-small",
      "title": "Embeddingモデル"
    }
  },
  "required": ["collection"]
}
```

### 5.5 vector-search（ベクトル検索）

```javascript
// Category: data
// Description: ベクトル類似検索を実行

const collection = config.collection || input.collection;
const vector = input.vector || input.vectors?.[0];
const query = input.query;

if (!collection) {
  throw new Error('[VECTOR_001] collection is required');
}

// クエリテキストの場合は先にベクトル化
let searchVector = vector;
if (!searchVector && query) {
  const provider = config.embedding_provider || 'openai';
  const model = config.embedding_model || 'text-embedding-3-small';
  const embedResult = await ctx.embedding.embed(provider, model, query);
  searchVector = embedResult.vectors[0];
}

if (!searchVector) {
  throw new Error('[VECTOR_003] vector or query is required');
}

// ctx.vector.query は tenant_id フィルタを自動適用
const result = await ctx.vector.query(collection, searchVector, {
  top_k: config.top_k || 5,
  threshold: config.threshold,
  filter: config.filter,
  include_content: config.include_content !== false
});

return {
  matches: result.matches,
  count: result.matches.length,
  collection
};
```

**Config Schema**:
```json
{
  "type": "object",
  "properties": {
    "collection": {
      "type": "string",
      "title": "コレクション名"
    },
    "top_k": {
      "type": "integer",
      "default": 5,
      "minimum": 1,
      "maximum": 100,
      "title": "取得件数"
    },
    "threshold": {
      "type": "number",
      "minimum": 0,
      "maximum": 1,
      "title": "類似度閾値",
      "description": "この値以上のスコアのみ返却"
    },
    "include_content": {
      "type": "boolean",
      "default": true,
      "title": "コンテンツを含める"
    },
    "embedding_provider": {
      "type": "string",
      "default": "openai",
      "title": "Embeddingプロバイダー",
      "description": "queryテキストのベクトル化に使用"
    },
    "embedding_model": {
      "type": "string",
      "default": "text-embedding-3-small",
      "title": "Embeddingモデル"
    }
  },
  "required": ["collection"]
}
```

### 5.6 vector-delete（ベクトル削除）

```javascript
// Category: data
// Description: ベクトルDBからドキュメントを削除

const collection = config.collection || input.collection;
const ids = input.ids || (input.id ? [input.id] : null);

if (!collection) {
  throw new Error('[VECTOR_001] collection is required');
}

if (!ids || ids.length === 0) {
  throw new Error('[VECTOR_004] ids array is required');
}

// ctx.vector.delete は tenant_id フィルタを自動適用
const result = await ctx.vector.delete(collection, ids);

return {
  collection,
  deleted_count: result.deleted_count,
  requested_ids: ids
};
```

### 5.7 rag-query（統合RAGクエリ）

```javascript
// Category: ai
// Description: RAGパイプライン（検索→コンテキスト構築→LLM回答）

const query = input.query || input.question;
const collection = config.collection || input.collection;

if (!query) {
  throw new Error('[RAG_001] query is required');
}

if (!collection) {
  throw new Error('[RAG_002] collection is required');
}

// 設定
const embeddingProvider = config.embedding_provider || 'openai';
const embeddingModel = config.embedding_model || 'text-embedding-3-small';
const llmProvider = config.llm_provider || 'openai';
const llmModel = config.llm_model || 'gpt-4';
const topK = config.top_k || 5;

// Step 1: クエリをベクトル化
const embedResult = await ctx.embedding.embed(embeddingProvider, embeddingModel, query);
const queryVector = embedResult.vectors[0];

// Step 2: 類似ドキュメント検索（tenant分離自動適用）
const searchResult = await ctx.vector.query(collection, queryVector, {
  top_k: topK,
  include_content: true
});

// Step 3: コンテキスト構築
const context = searchResult.matches
  .map((m, i) => `[${i + 1}] ${m.content}`)
  .join('\n\n---\n\n');

// Step 4: システムプロンプト
const systemPrompt = config.system_prompt ||
  `You are a helpful assistant. Answer the question based on the provided context.
If the context does not contain relevant information, say so clearly.
Always cite the source number [N] when using information from the context.`;

// Step 5: LLM呼び出し
const userPrompt = `## Context

${context}

## Question

${query}

## Answer`;

const llmResponse = await ctx.llm.chat(llmProvider, llmModel, {
  messages: [
    { role: 'system', content: systemPrompt },
    { role: 'user', content: userPrompt }
  ],
  temperature: config.temperature || 0.3,
  max_tokens: config.max_tokens || 2000
});

return {
  answer: llmResponse.content,
  sources: searchResult.matches.map(m => ({
    id: m.id,
    score: m.score,
    content: m.content?.substring(0, 200) + '...',
    metadata: m.metadata
  })),
  usage: {
    embedding: embedResult.usage,
    llm: llmResponse.usage
  }
};
```

**Config Schema**:
```json
{
  "type": "object",
  "properties": {
    "collection": {
      "type": "string",
      "title": "コレクション名"
    },
    "top_k": {
      "type": "integer",
      "default": 5,
      "title": "検索件数"
    },
    "embedding_provider": {
      "type": "string",
      "enum": ["openai", "cohere"],
      "default": "openai"
    },
    "embedding_model": {
      "type": "string",
      "default": "text-embedding-3-small"
    },
    "llm_provider": {
      "type": "string",
      "enum": ["openai", "anthropic"],
      "default": "openai"
    },
    "llm_model": {
      "type": "string",
      "default": "gpt-4"
    },
    "system_prompt": {
      "type": "string",
      "title": "システムプロンプト",
      "description": "カスタムシステムプロンプト"
    },
    "temperature": {
      "type": "number",
      "default": 0.3,
      "minimum": 0,
      "maximum": 2
    },
    "max_tokens": {
      "type": "integer",
      "default": 2000
    }
  },
  "required": ["collection"]
}
```

---

## 6. Go実装コンポーネント

### 6.1 EmbeddingService

```go
// backend/internal/block/sandbox/embedding.go

package sandbox

import (
    "context"
    "fmt"
)

// EmbeddingService provides embedding capabilities to sandbox scripts
type EmbeddingService interface {
    Embed(provider, model string, texts []string) (*EmbeddingResult, error)
}

type EmbeddingResult struct {
    Vectors   [][]float32 `json:"vectors"`
    Model     string      `json:"model"`
    Dimension int         `json:"dimension"`
    Usage     struct {
        TotalTokens int `json:"total_tokens"`
    } `json:"usage"`
}

// EmbeddingServiceImpl implements EmbeddingService
type EmbeddingServiceImpl struct {
    openaiAdapter    *OpenAIEmbeddingAdapter
    // cohereAdapter *CohereEmbeddingAdapter  // Phase 2
    tenantID         string
    usageRecorder    UsageRecorder
}

func (s *EmbeddingServiceImpl) Embed(provider, model string, texts []string) (*EmbeddingResult, error) {
    switch provider {
    case "openai":
        return s.openaiAdapter.Embed(model, texts)
    default:
        return nil, fmt.Errorf("unsupported embedding provider: %s", provider)
    }
}
```

### 6.2 VectorService（テナント分離実装）

```go
// backend/internal/block/sandbox/vector.go

package sandbox

import (
    "context"
    "fmt"

    "github.com/google/uuid"
)

// VectorService provides vector DB operations with tenant isolation
type VectorService interface {
    Upsert(collection string, documents []VectorDocument, opts *UpsertOptions) (*UpsertResult, error)
    Query(collection string, vector []float32, opts *QueryOptions) (*QueryResult, error)
    Delete(collection string, ids []string) (*DeleteResult, error)
    ListCollections() ([]CollectionInfo, error)
}

type VectorDocument struct {
    ID       string                 `json:"id,omitempty"`
    Content  string                 `json:"content"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
    Vector   []float32              `json:"vector,omitempty"`
}

// VectorServiceImpl implements VectorService with strict tenant isolation
type VectorServiceImpl struct {
    backend        VectorBackend  // pgvector, pinecone, etc.
    embeddingService EmbeddingService
    tenantID       uuid.UUID      // ⚠️ ExecutionContextから注入、ユーザー変更不可
}

// ⚠️ tenantIDは外部から設定不可、コンストラクタでのみ設定
func NewVectorService(tenantID uuid.UUID, backend VectorBackend, embedding EmbeddingService) *VectorServiceImpl {
    return &VectorServiceImpl{
        tenantID:         tenantID,
        backend:          backend,
        embeddingService: embedding,
    }
}

func (s *VectorServiceImpl) Upsert(collection string, documents []VectorDocument, opts *UpsertOptions) (*UpsertResult, error) {
    // ⚠️ tenant_id を強制的に付与
    for i := range documents {
        if documents[i].ID == "" {
            documents[i].ID = uuid.New().String()
        }
    }

    // バックエンドに tenant_id 付きで保存
    return s.backend.Upsert(s.tenantID, collection, documents, opts)
}

func (s *VectorServiceImpl) Query(collection string, vector []float32, opts *QueryOptions) (*QueryResult, error) {
    // ⚠️ tenant_id フィルタを強制適用
    return s.backend.Query(s.tenantID, collection, vector, opts)
}

func (s *VectorServiceImpl) Delete(collection string, ids []string) (*DeleteResult, error) {
    // ⚠️ 自テナントのドキュメントのみ削除可能
    return s.backend.Delete(s.tenantID, collection, ids)
}
```

### 6.3 PGVectorBackend（テナント分離クエリ）

```go
// backend/internal/adapter/vector_pgvector.go

package adapter

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
    pgvector "github.com/pgvector/pgvector-go"
)

type PGVectorBackend struct {
    pool *pgxpool.Pool
}

func (b *PGVectorBackend) Query(tenantID uuid.UUID, collection string, vector []float32, opts *QueryOptions) (*QueryResult, error) {
    // ⚠️ tenant_id フィルタを強制
    query := `
        SELECT
            vd.id,
            vd.content,
            vd.metadata,
            1 - (vd.embedding <=> $3) as score
        FROM vector_documents vd
        JOIN vector_collections vc ON vd.collection_id = vc.id
        WHERE vc.tenant_id = $1          -- ⚠️ 必須フィルタ
          AND vc.name = $2
          AND vd.tenant_id = $1          -- ⚠️ 二重チェック
        ORDER BY vd.embedding <=> $3
        LIMIT $4
    `

    rows, err := b.pool.Query(context.Background(), query,
        tenantID,                    // $1: tenant_id (強制)
        collection,                  // $2: collection name
        pgvector.NewVector(vector),  // $3: query vector
        opts.TopK,                   // $4: limit
    )
    // ... 結果処理
}

func (b *PGVectorBackend) Upsert(tenantID uuid.UUID, collection string, documents []VectorDocument, opts *UpsertOptions) (*UpsertResult, error) {
    // まずコレクションを取得/作成（tenant_id付き）
    collectionID, err := b.getOrCreateCollection(tenantID, collection, opts)
    if err != nil {
        return nil, err
    }

    // ⚠️ すべてのドキュメントに tenant_id を設定
    query := `
        INSERT INTO vector_documents (id, tenant_id, collection_id, content, metadata, embedding)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO UPDATE SET
            content = EXCLUDED.content,
            metadata = EXCLUDED.metadata,
            embedding = EXCLUDED.embedding
    `

    for _, doc := range documents {
        _, err := b.pool.Exec(context.Background(), query,
            doc.ID,
            tenantID,      // ⚠️ 強制設定
            collectionID,
            doc.Content,
            doc.Metadata,
            pgvector.NewVector(doc.Vector),
        )
        // ...
    }
}
```

---

## 7. 実装フェーズ

### Phase 1: 基盤（優先度: 高）

**目標**: 基本的なRAGワークフローを実現

| タスク | 詳細 |
|--------|------|
| 1.1 スキーマ追加 | `vector_collections`, `vector_documents` テーブル作成、pgvector有効化 |
| 1.2 ctx.embedding実装 | EmbeddingService インターフェース、OpenAIアダプター |
| 1.3 ctx.vector実装 | VectorService インターフェース、PGVectorBackend |
| 1.4 テナント分離実装 | tenant_id強制フィルタ、コレクション分離 |
| 1.5 ブロック追加 | `embedding`, `vector-search`, `vector-upsert` |
| 1.6 使用量記録 | embedding操作の使用量トラッキング |
| 1.7 テスト | 単体テスト、テナント分離テスト |

**成果物**:
- 基本RAGワークフロー動作
- テナント分離保証

### Phase 2: ドキュメント処理（優先度: 中）

**目標**: ドキュメント取り込みパイプライン

| タスク | 詳細 |
|--------|------|
| 2.1 doc-loaderブロック | URL/テキスト/JSONからドキュメント読み込み |
| 2.2 text-splitterブロック | チャンク分割、オーバーラップ対応 |
| 2.3 vector-deleteブロック | ドキュメント削除 |
| 2.4 rag-queryブロック | 統合RAGクエリブロック |

**成果物**:
- 完全なIndexingワークフロー
- 1ブロックRAGクエリ

### Phase 3: 高度な機能（✅ 実装完了）

| タスク | 詳細 | ステータス |
|--------|------|----------|
| 3.1 メタデータフィルタ | 検索時のメタデータフィルタリング ($eq, $ne, $gt, $gte, $lt, $lte, $in, $nin, $and, $or, $exists, $contains) | ✅ 完了 |
| 3.2 ハイブリッド検索 | キーワード + ベクトル検索（RRF方式） | ✅ 完了 |
| 3.3 追加Embeddingプロバイダー | Cohere, Voyage対応 | ✅ 完了 |
| 3.4 外部ベクトルDB | Pinecone, Weaviate, Qdrant対応 | ⏭️ スキップ（PGVectorで十分） |

---

## 8. セキュリティ

### 8.1 テナント分離チェックリスト

```
✅ ctx.vector は tenant_id を ExecutionContext から取得
✅ ユーザーが tenant_id を指定・変更することは不可能
✅ すべてのDBクエリに WHERE tenant_id = $1 を強制
✅ コレクション作成時も tenant_id を自動設定
✅ 削除操作も tenant_id フィルタで保護
✅ ListCollections も自テナントのみ返却
```

### 8.2 APIキー管理

| シークレット | 用途 |
|-------------|------|
| `OPENAI_API_KEY` | OpenAI Embedding API |
| システムクレデンシャル | オペレーター管理、テナント非公開 |

### 8.3 コスト制御

- 使用量記録: `usage_records` に embedding 操作を記録
- バジェットアラート: 既存の `usage_budgets` を活用
- レート制限: embedding APIのバッチ処理で効率化

---

## 9. テスト戦略

### 9.1 単体テスト

| 対象 | テスト内容 |
|------|----------|
| EmbeddingService | モック使用、ベクトル変換確認 |
| VectorService | モックBackend、CRUD操作確認 |
| PGVectorBackend | テストDB使用、クエリ確認 |

### 9.2 テナント分離テスト（必須）

```go
func TestTenantIsolation(t *testing.T) {
    // Tenant A がドキュメントを保存
    tenantA := uuid.New()
    serviceA := NewVectorService(tenantA, backend, embedding)
    serviceA.Upsert("shared-name", docsA, nil)

    // Tenant B が同名コレクションを検索
    tenantB := uuid.New()
    serviceB := NewVectorService(tenantB, backend, embedding)
    result, _ := serviceB.Query("shared-name", vector, nil)

    // ⚠️ Tenant B は Tenant A のデータを見れない
    assert.Empty(t, result.Matches)
}
```

### 9.3 E2Eテスト

- Indexingワークフロー: `doc-loader` → `text-splitter` → `embedding` → `vector-upsert`
- Queryワークフロー: `vector-search` → `llm`
- RAG統合: `rag-query` ブロック単体

---

## 10. ドキュメント更新

実装完了後に更新するドキュメント:

| ドキュメント | 更新内容 |
|-------------|---------|
| `docs/BLOCK_REGISTRY.md` | 新規RAGブロック一覧 |
| `docs/DATABASE.md` | vector_collections, vector_documents テーブル |
| `docs/API.md` | ベクトル関連API（必要に応じて） |
| `docs/BACKEND.md` | EmbeddingService, VectorService |
| `docs/designs/UNIFIED_BLOCK_MODEL.md` | ctx.embedding, ctx.vector 追加 |

---

## 11. 関連ドキュメント

- [UNIFIED_BLOCK_MODEL.md](../designs/UNIFIED_BLOCK_MODEL.md) - ブロックアーキテクチャ
- [BLOCK_REGISTRY.md](../BLOCK_REGISTRY.md) - 既存ブロック一覧
- [DATABASE.md](../DATABASE.md) - スキーマ管理
- [LangChain Retrieval Docs](https://docs.langchain.com/oss/python/langchain/retrieval) - 参考設計

---

## 12. 決定事項ログ

### Decision: pgvector を Phase 1 のデフォルトバックエンドとする
- **Date**: 2026-01-13
- **Context**: ベクトルDB選定
- **Options**: pgvector, Pinecone, Weaviate, Qdrant
- **Decision**: pgvector（PostgreSQL拡張）
- **Rationale**:
  - 既存インフラ活用（追加コスト・運用なし）
  - テナント分離がSQL WHERE句で実装可能
  - 小〜中規模に十分なパフォーマンス
- **Consequences**: 大規模時は外部ベクトルDB検討（Phase 3）

### Decision: ctx.embedding, ctx.vector をサンドボックスAPIとして追加
- **Date**: 2026-01-13
- **Context**: RAG機能の実装方式
- **Options**: ctx API拡張 vs HTTP-only (ctx.http)
- **Decision**: ctx API拡張
- **Rationale**:
  - ctx.llm と同様のパターンで一貫性
  - テナント分離をAPI層で強制可能
  - 使用量トラッキングが容易
- **Consequences**: Go実装が必要（EmbeddingService, VectorService）

### Decision: LangChain互換のDocument型を採用
- **Date**: 2026-01-13
- **Context**: データ構造設計
- **Decision**: `{ content, metadata }` 形式を標準化
- **Rationale**: LangChainとの互換性、シンプルな構造
