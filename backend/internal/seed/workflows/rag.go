package workflows

import "encoding/json"

func (r *Registry) registerRAGWorkflows() {
	r.register(RAGWorkflow())
}

// RAGWorkflow is a unified RAG workflow with 3 entry points:
// - indexing: Index documents into vector database
// - qa: Simple question answering
// - chat: Interactive chat with knowledge base
func RAGWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000101",
		SystemSlug:  "rag",
		Name:        "RAG Workflows",
		Description: "Retrieval-Augmented Generation workflows: document indexing, question answering, and knowledge base chat",
		Version:     1,
		IsSystem:    true,
		Steps: []SystemStepDefinition{
			// ============================
			// Document Indexing Entry Point (横並び: Y=40固定, X増加)
			// ============================
			{
				TempID:      "start_indexing",
				Name:        "Start: Document Indexing",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "indexing",
					"description": "Index documents into vector database"
				}`),
				PositionX: 40,
				PositionY: 40,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["documents", "collection"],
						"properties": {
							"documents": {
								"type": "array",
								"items": {
									"type": "object",
									"properties": {
										"content": {"type": "string"},
										"metadata": {"type": "object"}
									}
								}
							},
							"collection": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "indexing_split",
				Name:      "Split Documents",
				Type:      "text-splitter",
				PositionX: 160,
				PositionY: 40,
				BlockSlug: "text-splitter",
				Config: json.RawMessage(`{
					"chunk_size": 1000,
					"chunk_overlap": 200
				}`),
			},
			{
				TempID:    "indexing_embed",
				Name:      "Generate Embeddings",
				Type:      "embedding",
				PositionX: 280,
				PositionY: 40,
				BlockSlug: "embedding",
				Config: json.RawMessage(`{
					"provider": "openai",
					"model": "text-embedding-3-small"
				}`),
			},
			{
				TempID:    "indexing_store",
				Name:      "Store in Vector DB",
				Type:      "vector-upsert",
				PositionX: 400,
				PositionY: 40,
				BlockSlug: "vector-upsert",
				Config: json.RawMessage(`{}`),
			},
			{
				TempID:    "indexing_result",
				Name:      "Return Result",
				Type:      "function",
				PositionX: 520,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "return { indexed_count: input.upserted_count, chunk_count: input.upserted_count, collection: input.collection, ids: input.ids };",
					"language": "javascript"
				}`),
			},

			// ============================
			// Question Answering Entry Point (横並び: Y=160固定, X増加)
			// ============================
			{
				TempID:      "start_qa",
				Name:        "Start: Question Answering",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "qa",
					"description": "Answer questions using RAG"
				}`),
				PositionX: 40,
				PositionY: 160,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["query", "collection"],
						"properties": {
							"query": {"type": "string"},
							"collection": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "qa_query",
				Name:      "RAG Query",
				Type:      "rag-query",
				PositionX: 160,
				PositionY: 160,
				BlockSlug: "rag-query",
				Config: json.RawMessage(`{
					"top_k": 5,
					"llm_provider": "openai",
					"llm_model": "gpt-4o-mini",
					"system_prompt": "You are a helpful assistant. Answer questions based on the provided context. If the context does not contain enough information, say so clearly."
				}`),
			},

			// ============================
			// Knowledge Base Chat Entry Point (横並び: Y=280固定, X増加)
			// ============================
			{
				TempID:      "start_chat",
				Name:        "Start: Knowledge Base Chat",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "chat",
					"description": "Interactive chat with knowledge base"
				}`),
				PositionX: 40,
				PositionY: 280,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["query", "collection"],
						"properties": {
							"query": {"type": "string"},
							"collection": {"type": "string"},
							"chat_history": {
								"type": "array",
								"items": {
									"type": "object",
									"properties": {
										"role": {"type": "string"},
										"content": {"type": "string"}
									}
								}
							}
						}
					}
				}`),
			},
			{
				TempID:    "chat_search",
				Name:      "Search Documents",
				Type:      "vector-search",
				PositionX: 160,
				PositionY: 280,
				BlockSlug: "vector-search",
				Config: json.RawMessage(`{
					"top_k": 5,
					"include_content": true
				}`),
			},
			{
				TempID:    "chat_context",
				Name:      "Build Context",
				Type:      "function",
				PositionX: 280,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const context = (input.matches || []).map((m, i) => ` + "`" + `[${i+1}] ${m.content}` + "`" + `).join('\\n\\n---\\n\\n'); const history = (input.chat_history || []).map(h => ` + "`" + `${h.role}: ${h.content}` + "`" + `).join('\\n'); return { context, history, query: input.query, matches: input.matches, chat_history: input.chat_history };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "chat_llm",
				Name:      "Generate Answer",
				Type:      "llm",
				PositionX: 400,
				PositionY: 280,
				Config: json.RawMessage(`{
					"provider": "openai",
					"model": "gpt-4o-mini",
					"system_prompt": "You are a helpful knowledge base assistant. Answer based on the context provided. Cite sources using [N] notation.",
					"user_prompt": "## Previous Conversation\n{{$.history}}\n\n## Retrieved Context\n{{$.context}}\n\n## User Question\n{{$.query}}\n\n## Answer",
					"temperature": 0.3,
					"max_tokens": 2000
				}`),
			},
			{
				TempID:    "chat_format",
				Name:      "Format Response",
				Type:      "function",
				PositionX: 520,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const newHistory = [...(input.chat_history || []), {role: 'user', content: input.query}, {role: 'assistant', content: input.content}]; return { answer: input.content, sources: (input.matches || []).map(m => ({id: m.id, score: m.score, excerpt: (m.content || '').substring(0, 150) + '...'})), chat_history: newHistory };",
					"language": "javascript"
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			// Indexing flow
			{SourceTempID: "start_indexing", TargetTempID: "indexing_split", SourcePort: "output"},
			{SourceTempID: "indexing_split", TargetTempID: "indexing_embed", SourcePort: "output"},
			{SourceTempID: "indexing_embed", TargetTempID: "indexing_store", SourcePort: "output"},
			{SourceTempID: "indexing_store", TargetTempID: "indexing_result", SourcePort: "output"},

			// QA flow
			{SourceTempID: "start_qa", TargetTempID: "qa_query", SourcePort: "output"},

			// Chat flow
			{SourceTempID: "start_chat", TargetTempID: "chat_search", SourcePort: "output"},
			{SourceTempID: "chat_search", TargetTempID: "chat_context", SourcePort: "output"},
			{SourceTempID: "chat_context", TargetTempID: "chat_llm", SourcePort: "output"},
			{SourceTempID: "chat_llm", TargetTempID: "chat_format", SourcePort: "output"},
		},
	}
}
