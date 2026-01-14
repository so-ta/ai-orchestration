package workflows

import "encoding/json"

func (r *Registry) registerRAGWorkflows() {
	r.register(RAGDocumentIndexingWorkflow())
	r.register(RAGQuestionAnsweringWorkflow())
	r.register(RAGKnowledgeBaseChatWorkflow())
}

// RAGDocumentIndexingWorkflow indexes documents into vector database for RAG queries
func RAGDocumentIndexingWorkflow() *SystemWorkflowDefinition {
	// Block definition IDs for reference
	textSplitterBlockID := "0a900006-0000-0000-0000-000000000001"
	embeddingBlockID := "0a900001-0000-0000-0000-000000000001"
	vectorUpsertBlockID := "0a900002-0000-0000-0000-000000000001"

	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000101",
		SystemSlug:  "rag-document-indexing",
		Name:        "RAG: Document Indexing Pipeline",
		Description: "Index documents into vector database for RAG queries. Split documents into chunks, generate embeddings, and store in vector DB.",
		Version:     1,
		IsSystem:    true,
		InputSchema: json.RawMessage(`{
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
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"indexed_count": {"type": "integer"},
				"chunk_count": {"type": "integer"}
			}
		}`),
		Steps: []SystemStepDefinition{
			{
				TempID:    "step_1",
				Name:      "Start",
				Type:      "start",
				PositionX: 400,
				PositionY: 50,
				Config:    json.RawMessage(`{}`),
			},
			{
				TempID:     "step_2",
				Name:       "Split Documents",
				Type:       "text-splitter",
				PositionX:  400,
				PositionY:  200,
				BlockDefID: &textSplitterBlockID,
				Config: json.RawMessage(`{
					"chunk_size": 1000,
					"chunk_overlap": 200
				}`),
			},
			{
				TempID:     "step_3",
				Name:       "Generate Embeddings",
				Type:       "embedding",
				PositionX:  400,
				PositionY:  350,
				BlockDefID: &embeddingBlockID,
				Config: json.RawMessage(`{
					"provider": "openai",
					"model": "text-embedding-3-small"
				}`),
			},
			{
				TempID:     "step_4",
				Name:       "Store in Vector DB",
				Type:       "vector-upsert",
				PositionX:  400,
				PositionY:  500,
				BlockDefID: &vectorUpsertBlockID,
				Config: json.RawMessage(`{
					"collection": "{{$.collection}}"
				}`),
			},
			{
				TempID:    "step_5",
				Name:      "Return Result",
				Type:      "function",
				PositionX: 400,
				PositionY: 650,
				Config: json.RawMessage(`{
					"code": "return { indexed_count: input.upserted_count, chunk_count: input.upserted_count, collection: input.collection, ids: input.ids };",
					"language": "javascript"
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			{SourceTempID: "step_1", TargetTempID: "step_2", SourcePort: "output"},
			{SourceTempID: "step_2", TargetTempID: "step_3", SourcePort: "output"},
			{SourceTempID: "step_3", TargetTempID: "step_4", SourcePort: "output"},
			{SourceTempID: "step_4", TargetTempID: "step_5", SourcePort: "output"},
		},
	}
}

// RAGQuestionAnsweringWorkflow answers questions using RAG
func RAGQuestionAnsweringWorkflow() *SystemWorkflowDefinition {
	ragQueryBlockID := "0a900007-0000-0000-0000-000000000001"

	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000102",
		SystemSlug:  "rag-question-answering",
		Name:        "RAG: Question Answering",
		Description: "Answer questions using RAG. Searches vector database for relevant documents and generates answer using LLM.",
		Version:     1,
		IsSystem:    true,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["query", "collection"],
			"properties": {
				"query": {"type": "string"},
				"collection": {"type": "string"}
			}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"answer": {"type": "string"},
				"sources": {"type": "array"}
			}
		}`),
		Steps: []SystemStepDefinition{
			{
				TempID:    "step_1",
				Name:      "Start",
				Type:      "start",
				PositionX: 400,
				PositionY: 50,
				Config:    json.RawMessage(`{}`),
			},
			{
				TempID:     "step_2",
				Name:       "RAG Query",
				Type:       "rag-query",
				PositionX:  400,
				PositionY:  200,
				BlockDefID: &ragQueryBlockID,
				Config: json.RawMessage(`{
					"collection": "{{$.collection}}",
					"top_k": 5,
					"llm_provider": "openai",
					"llm_model": "gpt-4o-mini",
					"system_prompt": "You are a helpful assistant. Answer questions based on the provided context. If the context does not contain enough information, say so clearly."
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			{SourceTempID: "step_1", TargetTempID: "step_2", SourcePort: "output"},
		},
	}
}

// RAGKnowledgeBaseChatWorkflow provides interactive chat with knowledge base
func RAGKnowledgeBaseChatWorkflow() *SystemWorkflowDefinition {
	vectorSearchBlockID := "0a900003-0000-0000-0000-000000000001"

	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000103",
		SystemSlug:  "rag-knowledge-base-chat",
		Name:        "RAG: Knowledge Base Chat",
		Description: "Interactive chat with knowledge base. Maintains conversation context and retrieves relevant documents for each query.",
		Version:     1,
		IsSystem:    true,
		InputSchema: json.RawMessage(`{
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
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"answer": {"type": "string"},
				"sources": {"type": "array"},
				"chat_history": {"type": "array"}
			}
		}`),
		Steps: []SystemStepDefinition{
			{
				TempID:    "step_1",
				Name:      "Start",
				Type:      "start",
				PositionX: 400,
				PositionY: 50,
				Config:    json.RawMessage(`{}`),
			},
			{
				TempID:     "step_2",
				Name:       "Search Documents",
				Type:       "vector-search",
				PositionX:  400,
				PositionY:  200,
				BlockDefID: &vectorSearchBlockID,
				Config: json.RawMessage(`{
					"collection": "{{$.collection}}",
					"top_k": 5,
					"include_content": true
				}`),
			},
			{
				TempID:    "step_3",
				Name:      "Build Context",
				Type:      "function",
				PositionX: 400,
				PositionY: 350,
				Config: json.RawMessage(`{
					"code": "const context = (input.matches || []).map((m, i) => ` + "`" + `[${i+1}] ${m.content}` + "`" + `).join('\\n\\n---\\n\\n'); const history = (input.chat_history || []).map(h => ` + "`" + `${h.role}: ${h.content}` + "`" + `).join('\\n'); return { context, history, query: input.query, matches: input.matches };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "step_4",
				Name:      "Generate Answer",
				Type:      "llm",
				PositionX: 400,
				PositionY: 500,
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
				TempID:    "step_5",
				Name:      "Format Response",
				Type:      "function",
				PositionX: 400,
				PositionY: 650,
				Config: json.RawMessage(`{
					"code": "const newHistory = [...(input.chat_history || []), {role: 'user', content: input.query}, {role: 'assistant', content: input.content}]; return { answer: input.content, sources: (input.matches || []).map(m => ({id: m.id, score: m.score, excerpt: (m.content || '').substring(0, 150) + '...'})), chat_history: newHistory };",
					"language": "javascript"
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			{SourceTempID: "step_1", TargetTempID: "step_2", SourcePort: "output"},
			{SourceTempID: "step_2", TargetTempID: "step_3", SourcePort: "output"},
			{SourceTempID: "step_3", TargetTempID: "step_4", SourcePort: "output"},
			{SourceTempID: "step_4", TargetTempID: "step_5", SourcePort: "output"},
		},
	}
}
