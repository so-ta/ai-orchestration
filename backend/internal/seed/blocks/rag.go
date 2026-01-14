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
		Name:        "Document Loader",
		Description: "Load documents from URL, text, or JSON",
		Category:    domain.BlockCategoryData,
		Icon:        "file-text",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"source_type": {"type": "string", "enum": ["url", "text", "json"], "default": "url", "title": "Source Type"},
				"url": {"type": "string", "title": "URL"},
				"content": {"type": "string", "title": "Text Content"},
				"strip_html": {"type": "boolean", "default": true, "title": "Strip HTML Tags"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {"type": "string"},
				"content": {"type": "string"},
				"text": {"type": "string"}
			}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"documents": {"type": "array"}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "object"}`), Required: false, Description: "Optional source data"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Loaded documents"},
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
		UIConfig: json.RawMessage(`{"icon": "file-text", "color": "#F59E0B"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "DOC_001", Name: "FETCH_ERROR", Description: "Failed to fetch URL", Retryable: true},
			{Code: "DOC_002", Name: "EMPTY_CONTENT", Description: "No content provided", Retryable: false},
		},
		Enabled: true,
	}
}

func TextSplitterBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "text-splitter",
		Version:     1,
		Name:        "Text Splitter",
		Description: "Split documents into smaller chunks",
		Category:    domain.BlockCategoryData,
		Icon:        "scissors",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"chunk_size": {"type": "integer", "default": 1000, "minimum": 100, "maximum": 8000, "title": "Chunk Size (chars)"},
				"chunk_overlap": {"type": "integer", "default": 200, "minimum": 0, "title": "Overlap (chars)"},
				"separator": {"type": "string", "default": "\\n\\n", "title": "Separator"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"documents": {"type": "array"},
				"content": {"type": "string"},
				"text": {"type": "string"}
			}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"documents": {"type": "array"},
				"chunk_count": {"type": "integer"}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "documents", Label: "Documents", Schema: json.RawMessage(`{"type": "array"}`), Required: true, Description: "Documents to split"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Split documents"},
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
		UIConfig: json.RawMessage(`{"icon": "scissors", "color": "#06B6D4"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "SPLIT_001", Name: "NO_CONTENT", Description: "No content to split", Retryable: false},
		},
		Enabled: true,
	}
}

func RAGQueryBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "rag-query",
		Version:     1,
		Name:        "RAG Query",
		Description: "Search documents and generate answer with LLM",
		Category:    domain.BlockCategoryAI,
		Icon:        "message-square",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["collection"],
			"properties": {
				"collection": {"type": "string", "title": "Collection Name"},
				"top_k": {"type": "integer", "default": 5, "title": "Search Results"},
				"embedding_provider": {"type": "string", "default": "openai"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small"},
				"llm_provider": {"type": "string", "enum": ["openai", "anthropic"], "default": "openai", "title": "LLM Provider"},
				"llm_model": {"type": "string", "default": "gpt-4", "title": "LLM Model"},
				"system_prompt": {"type": "string", "title": "System Prompt"},
				"temperature": {"type": "number", "default": 0.3, "minimum": 0, "maximum": 2},
				"max_tokens": {"type": "integer", "default": 2000}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["query"],
			"properties": {
				"query": {"type": "string"},
				"question": {"type": "string"}
			}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"answer": {"type": "string"},
				"sources": {"type": "array"}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "query", Label: "Query", Schema: json.RawMessage(`{"type": "string"}`), Required: true, Description: "Question to answer"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Answer with sources"},
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
		UIConfig: json.RawMessage(`{"icon": "message-square", "color": "#8B5CF6"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "RAG_001", Name: "QUERY_REQUIRED", Description: "Query is required", Retryable: false},
			{Code: "RAG_002", Name: "COLLECTION_REQUIRED", Description: "Collection is required", Retryable: false},
		},
		Enabled: true,
	}
}
