package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerRAGBlocks() {
	r.register(DocLoaderBlock())
	r.register(TextSplitterBlock())
	r.register(RAGQueryBlock())
}

func DocLoaderBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "doc-loader",
		Version:     1,
		Name:        LText("Document Loader", "ドキュメントローダー"),
		Description: LText("Load documents from URL, text, or JSON", "URL、テキスト、またはJSONからドキュメントを読み込み"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "file-text",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"source_type": {"type": "string", "enum": ["url", "text", "json"], "default": "url", "title": "Source Type", "description": "Document source type"},
				"url": {"type": "string", "title": "URL", "description": "URL to load document from"},
				"content": {"type": "string", "title": "Text Content", "description": "Text content to load"},
				"strip_html": {"type": "boolean", "default": true, "title": "Strip HTML Tags", "description": "Remove HTML tags from content"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"source_type": {"type": "string", "enum": ["url", "text", "json"], "default": "url", "title": "ソースタイプ", "description": "ドキュメントのソースタイプ"},
				"url": {"type": "string", "title": "URL", "description": "ドキュメントを読み込むURL"},
				"content": {"type": "string", "title": "テキストコンテンツ", "description": "読み込むテキストコンテンツ"},
				"strip_html": {"type": "boolean", "default": true, "title": "HTMLタグを除去", "description": "コンテンツからHTMLタグを除去"}
			}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"documents": {"type": "array"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Loaded documents", "読み込まれたドキュメント", true),
		},
		Code: `function isPrivateIp(hostname) {
  const parts = hostname.split('.').map(Number);
  if (parts.length !== 4 || parts.some((p) => Number.isNaN(p) || p < 0 || p > 255)) return false;
  const [a, b] = parts;
  if (a === 10) return true;
  if (a === 127) return true;
  if (a === 0) return true;
  if (a === 169 && b === 254) return true;
  if (a === 172 && b >= 16 && b <= 31) return true;
  if (a === 192 && b === 168) return true;
  return false;
}
function validateExternalUrl(rawUrl) {
  let parsed;
  try { parsed = new URL(rawUrl); } catch { throw new Error('[DOC_001] Invalid URL format'); }
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') throw new Error('[DOC_001] Only HTTP(S) URLs are allowed');
  const hostname = parsed.hostname.toLowerCase();
  if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1' || hostname === '0.0.0.0') throw new Error('[DOC_001] Access to local addresses is not allowed');
  if (/^(?:\d{1,3}\.){3}\d{1,3}$/.test(hostname) && isPrivateIp(hostname)) throw new Error('[DOC_001] Access to private network addresses is not allowed');
  if (hostname.endsWith('.local') || hostname.endsWith('.internal')) throw new Error('[DOC_001] Access to internal hostnames is not allowed');
  return parsed.toString();
}
const sourceType = config.source_type || 'url';
let content, metadata;
if (sourceType === 'url') {
  const rawUrl = config.url || input.url;
  if (!rawUrl) throw new Error('[DOC_002] URL is required for url source type');
  const url = validateExternalUrl(rawUrl);
  const response = ctx.http.get(url);
  content = typeof response.data === 'string' ? response.data : JSON.stringify(response.data);
  metadata = {source: url, source_type: 'url', content_type: response.headers['Content-Type'], fetched_at: new Date().toISOString()};
} else if (sourceType === 'text') {
  content = config.content || input.content || input.text;
  if (!content) throw new Error('[DOC_002] No content provided');
  metadata = {source_type: 'text'};
} else if (sourceType === 'json') {
  const data = input.data || input;
  content = config.content_path ? getPath(data, config.content_path) : JSON.stringify(data);
  metadata = {source_type: 'json'};
}
if (config.strip_html && content && content.includes('<')) {
  content = content.replace(/<script[^>]*>[\s\S]*?<\/script>/gi, '').replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '').replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim();
}
return {documents: [{content, metadata, char_count: content.length}]};`,
		UIConfig: LSchema(`{"icon": "file-text", "color": "#F59E0B"}`, `{"icon": "file-text", "color": "#F59E0B"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("DOC_001", "FETCH_ERROR", "取得エラー", "Failed to fetch URL", "URLの取得に失敗しました", true),
			LError("DOC_002", "EMPTY_CONTENT", "空のコンテンツ", "No content provided", "コンテンツが提供されていません", false),
		},
		Enabled: true,
	}
}

func TextSplitterBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "text-splitter",
		Version:     1,
		Name:        LText("Text Splitter", "テキスト分割"),
		Description: LText("Split documents into smaller chunks", "ドキュメントを小さなチャンクに分割"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "scissors",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"chunk_size": {"type": "integer", "default": 1000, "minimum": 100, "maximum": 8000, "title": "Chunk Size (chars)", "description": "Maximum characters per chunk"},
				"chunk_overlap": {"type": "integer", "default": 200, "minimum": 0, "title": "Overlap (chars)", "description": "Character overlap between chunks"},
				"separator": {"type": "string", "default": "\\n\\n", "title": "Separator", "description": "Text separator for splitting"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"chunk_size": {"type": "integer", "default": 1000, "minimum": 100, "maximum": 8000, "title": "チャンクサイズ（文字数）", "description": "チャンクあたりの最大文字数"},
				"chunk_overlap": {"type": "integer", "default": 200, "minimum": 0, "title": "オーバーラップ（文字数）", "description": "チャンク間の文字オーバーラップ"},
				"separator": {"type": "string", "default": "\\n\\n", "title": "区切り文字", "description": "分割用のテキスト区切り文字"}
			}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"documents": {"type": "array"},
				"chunk_count": {"type": "integer"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Split documents", "分割されたドキュメント", true),
		},
		Code: `const documents = input.documents || [{content: input.content || input.text}];
if (!documents || documents.length === 0) throw new Error('[SPLIT_001] No content to split');
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
      const words = current.split(/\s+/);
      const overlapWords = Math.ceil(overlap / 6);
      current = words.slice(-overlapWords).join(' ') + sep + segment;
    } else {
      current = combined;
    }
  }
  if (current.trim()) chunks.push(current.trim());
  return chunks;
}
const result = [];
for (const doc of documents) {
  const chunks = splitText(doc.content || '', chunkSize, chunkOverlap, separator);
  for (let i = 0; i < chunks.length; i++) {
    result.push({content: chunks[i], metadata: {...(doc.metadata || {}), chunk_index: i, chunk_total: chunks.length}, char_count: chunks[i].length});
  }
}
return {documents: result, chunk_count: result.length, original_count: documents.length};`,
		UIConfig: LSchema(`{"icon": "scissors", "color": "#06B6D4"}`, `{"icon": "scissors", "color": "#06B6D4"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("SPLIT_001", "NO_CONTENT", "コンテンツなし", "No content to split", "分割するコンテンツがありません", false),
		},
		Enabled: true,
	}
}

func RAGQueryBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "rag-query",
		Version:     1,
		Name:        LText("RAG Query", "RAGクエリ"),
		Description: LText("Search documents and generate answer with LLM", "ドキュメントを検索しLLMで回答を生成"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "message-square",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"collection": {"type": "string", "title": "Collection Name", "description": "Collection name (can also be provided via input.collection)"},
				"top_k": {"type": "integer", "default": 5, "title": "Search Results", "description": "Number of search results to retrieve"},
				"embedding_provider": {"type": "string", "default": "openai", "title": "Embedding Provider"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small", "title": "Embedding Model"},
				"llm_provider": {"type": "string", "enum": ["openai", "anthropic"], "default": "openai", "title": "LLM Provider", "description": "LLM provider for answer generation"},
				"llm_model": {"type": "string", "default": "gpt-4", "title": "LLM Model", "description": "LLM model for answer generation"},
				"system_prompt": {"type": "string", "title": "System Prompt", "description": "System prompt for LLM"},
				"temperature": {"type": "number", "default": 0.3, "minimum": 0, "maximum": 2, "title": "Temperature"},
				"max_tokens": {"type": "integer", "default": 2000, "title": "Max Tokens"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"collection": {"type": "string", "title": "コレクション名", "description": "コレクション名（input.collectionでも指定可能）"},
				"top_k": {"type": "integer", "default": 5, "title": "検索結果数", "description": "取得する検索結果の数"},
				"embedding_provider": {"type": "string", "default": "openai", "title": "埋め込みプロバイダー"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small", "title": "埋め込みモデル"},
				"llm_provider": {"type": "string", "enum": ["openai", "anthropic"], "default": "openai", "title": "LLMプロバイダー", "description": "回答生成用のLLMプロバイダー"},
				"llm_model": {"type": "string", "default": "gpt-4", "title": "LLMモデル", "description": "回答生成用のLLMモデル"},
				"system_prompt": {"type": "string", "title": "システムプロンプト", "description": "LLM用のシステムプロンプト"},
				"temperature": {"type": "number", "default": 0.3, "minimum": 0, "maximum": 2, "title": "温度"},
				"max_tokens": {"type": "integer", "default": 2000, "title": "最大トークン数"}
			}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"answer": {"type": "string"},
				"sources": {"type": "array"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Answer with sources", "ソース付きの回答", true),
		},
		Code: `const query = input.query || input.question;
if (!query) throw new Error('[RAG_001] Query is required');
const collection = config.collection || input.collection;
if (!collection) throw new Error('[RAG_002] Collection is required');
const embeddingProvider = config.embedding_provider || 'openai';
const embeddingModel = config.embedding_model || 'text-embedding-3-small';
const llmProvider = config.llm_provider || 'openai';
const llmModel = config.llm_model || 'gpt-4';
const topK = config.top_k || 5;
const embedResult = ctx.embedding.embed(embeddingProvider, embeddingModel, [query]);
const queryVector = embedResult.vectors[0];
const searchResult = ctx.vector.query(collection, queryVector, {top_k: topK, include_content: true});
const context = searchResult.matches.map((m, i) => '[' + (i + 1) + '] ' + m.content).join('\n\n---\n\n');
const systemPrompt = config.system_prompt || 'You are a helpful assistant. Answer based on the provided context. Cite sources using [N]. If context lacks relevant info, say so.';
const userPrompt = '## Context\n\n' + context + '\n\n## Question\n\n' + query + '\n\n## Answer';
const llmResponse = ctx.llm.chat(llmProvider, llmModel, {messages: [{role: 'system', content: systemPrompt}, {role: 'user', content: userPrompt}], temperature: config.temperature || 0.3, max_tokens: config.max_tokens || 2000});
return {answer: llmResponse.content, sources: searchResult.matches.map(m => ({id: m.id, score: m.score, content: (m.content || '').substring(0, 200) + '...', metadata: m.metadata})), usage: {embedding: embedResult.usage, llm: llmResponse.usage}};`,
		UIConfig: LSchema(`{"icon": "message-square", "color": "#8B5CF6"}`, `{"icon": "message-square", "color": "#8B5CF6"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("RAG_001", "QUERY_REQUIRED", "クエリ必須", "Query is required", "クエリが必要です", false),
			LError("RAG_002", "COLLECTION_REQUIRED", "コレクション必須", "Collection is required", "コレクションが必要です", false),
		},
		Enabled: true,
	}
}
