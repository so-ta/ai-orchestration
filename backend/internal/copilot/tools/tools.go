package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// Context keys for tool execution
type contextKey string

const (
	TenantIDKey  contextKey = "tenant_id"
	UserIDKey    contextKey = "user_id"
	ProjectIDKey contextKey = "project_id"
)

// Dependencies holds the dependencies required by tools
type Dependencies struct {
	BlockRepo   repository.BlockDefinitionRepository
	ProjectRepo repository.ProjectRepository
	StepRepo    repository.StepRepository
	EdgeRepo    repository.EdgeRepository
	RunRepo     repository.RunRepository
	StepRunRepo repository.StepRunRepository
}

// RegisterCoreTools registers all core tools with the registry
func RegisterCoreTools(reg *Registry, deps *Dependencies) error {
	tools := []*Tool{
		// Context collection tools
		listBlocksTool(deps),
		getBlockSchemaTool(deps),
		searchBlocksTool(deps),
		listWorkflowsTool(deps),
		getWorkflowTool(deps),
		getWorkflowRunsTool(deps),
		searchDocumentationTool(deps),

		// Analysis tools (for Enhance mode)
		diagnoseWorkflowTool(deps),

		// Workflow manipulation tools
		createStepTool(deps),
		updateStepTool(deps),
		deleteStepTool(deps),
		createEdgeTool(deps),
		deleteEdgeTool(deps),

		// Validation tools
		validateWorkflowTool(deps),
	}

	for _, tool := range tools {
		if err := reg.Register(tool); err != nil {
			return fmt.Errorf("register tool %s: %w", tool.Name, err)
		}
	}

	return nil
}

// ============================================================================
// Context Collection Tools
// ============================================================================

func listBlocksTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "list_blocks",
		Description: "利用可能なブロック（ステップタイプ）の一覧を取得します。ワークフローで使用できるブロックの種類を確認するために使用します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"category": {
					"type": "string",
					"description": "フィルタするカテゴリ（ai, flow, apps, custom）",
					"enum": ["ai", "flow", "apps", "custom"]
				},
				"search": {
					"type": "string",
					"description": "名前や説明で検索するキーワード"
				}
			},
			"required": []
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				Category string `json:"category"`
				Search   string `json:"search"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			filter := repository.BlockDefinitionFilter{
				EnabledOnly: true,
			}
			if params.Category != "" {
				category := domain.BlockCategory(params.Category)
				filter.Category = &category
			}
			if params.Search != "" {
				filter.Search = &params.Search
			}

			blocks, err := deps.BlockRepo.List(ctx, &tenantID, filter)
			if err != nil {
				return nil, fmt.Errorf("list blocks: %w", err)
			}

			result := make([]map[string]interface{}, 0, len(blocks))
			for _, b := range blocks {
				result = append(result, map[string]interface{}{
					"slug":        b.Slug,
					"name":        b.Name,
					"description": b.Description,
					"category":    b.Category,
					"subcategory": b.Subcategory,
				})
			}

			return json.Marshal(map[string]interface{}{
				"blocks": result,
				"count":  len(result),
			})
		},
	}
}

func getBlockSchemaTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "get_block_schema",
		Description: "特定のブロックの設定スキーマを取得します。ステップ作成時に必要な設定項目を確認するために使用します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"slug": {
					"type": "string",
					"description": "ブロックのスラッグ（例: llm-chat, http-request）"
				}
			},
			"required": ["slug"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				Slug string `json:"slug"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			block, err := deps.BlockRepo.GetBySlug(ctx, &tenantID, params.Slug)
			if err != nil {
				return nil, fmt.Errorf("get block %s: %w", params.Slug, err)
			}
			if block == nil {
				return nil, fmt.Errorf("block not found: %s", params.Slug)
			}

			return json.Marshal(map[string]interface{}{
				"slug":          block.Slug,
				"name":          block.Name,
				"description":   block.Description,
				"config_schema": block.ConfigSchema,
				"output_schema": block.OutputSchema,
				"input_ports":   block.InputPorts,
				"output_ports":  block.OutputPorts,
			})
		},
	}
}

func searchBlocksTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "search_blocks",
		Description: "ブロックをセマンティック検索します。やりたいことに適したブロックを見つけるために使用します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "検索クエリ（例: Slackにメッセージを送る, データベースからデータを取得）"
				}
			},
			"required": ["query"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			// For now, use simple keyword search
			filter := repository.BlockDefinitionFilter{
				EnabledOnly: true,
				Search:      &params.Query,
			}

			blocks, err := deps.BlockRepo.List(ctx, &tenantID, filter)
			if err != nil {
				return nil, fmt.Errorf("search blocks: %w", err)
			}

			result := make([]map[string]interface{}, 0, len(blocks))
			for _, b := range blocks {
				result = append(result, map[string]interface{}{
					"slug":        b.Slug,
					"name":        b.Name,
					"description": b.Description,
					"category":    b.Category,
					"relevance":   "high",
				})
			}

			return json.Marshal(map[string]interface{}{
				"results": result,
				"count":   len(result),
			})
		},
	}
}

func listWorkflowsTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "list_workflows",
		Description: "テナント内のワークフロー（プロジェクト）一覧を取得します。既存のワークフローを参照するために使用します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"status": {
					"type": "string",
					"description": "ステータスでフィルタ",
					"enum": ["draft", "active", "archived"]
				},
				"limit": {
					"type": "integer",
					"description": "取得件数（デフォルト: 20）"
				}
			},
			"required": []
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				Status string `json:"status"`
				Limit  int    `json:"limit"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			if params.Limit == 0 {
				params.Limit = 20
			}

			filter := repository.ProjectFilter{
				Limit: params.Limit,
			}
			if params.Status != "" {
				status := domain.ProjectStatus(params.Status)
				filter.Status = &status
			}

			projects, _, err := deps.ProjectRepo.List(ctx, tenantID, filter)
			if err != nil {
				return nil, fmt.Errorf("list projects: %w", err)
			}

			result := make([]map[string]interface{}, 0, len(projects))
			for _, p := range projects {
				result = append(result, map[string]interface{}{
					"id":          p.ID.String(),
					"name":        p.Name,
					"description": p.Description,
					"status":      p.Status,
					"version":     p.Version,
					"updated_at":  p.UpdatedAt,
				})
			}

			return json.Marshal(map[string]interface{}{
				"workflows": result,
				"count":     len(result),
			})
		},
	}
}

func getWorkflowTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "get_workflow",
		Description: "特定のワークフローの詳細（ステップ、エッジ）を取得します。ワークフローの構造を確認するために使用します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"workflow_id": {
					"type": "string",
					"description": "ワークフローのUUID"
				}
			},
			"required": ["workflow_id"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				WorkflowID string `json:"workflow_id"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			workflowID, err := uuid.Parse(params.WorkflowID)
			if err != nil {
				return nil, fmt.Errorf("invalid workflow_id: %w", err)
			}

			project, err := deps.ProjectRepo.GetWithStepsAndEdges(ctx, tenantID, workflowID)
			if err != nil {
				return nil, fmt.Errorf("get workflow: %w", err)
			}

			steps := make([]map[string]interface{}, 0, len(project.Steps))
			for _, s := range project.Steps {
				stepInfo := map[string]interface{}{
					"id":         s.ID.String(),
					"name":       s.Name,
					"type":       s.Type,
					"config":     s.Config,
					"position_x": s.PositionX,
					"position_y": s.PositionY,
					"is_entry":   s.IsStartBlock(),
				}
				if s.BlockDefinitionID != nil {
					stepInfo["block_definition_id"] = s.BlockDefinitionID.String()
				}
				steps = append(steps, stepInfo)
			}

			edges := make([]map[string]interface{}, 0, len(project.Edges))
			for _, e := range project.Edges {
				edgeInfo := map[string]interface{}{
					"id":          e.ID.String(),
					"source_port": e.SourcePort,
				}
				if e.SourceStepID != nil {
					edgeInfo["source_step_id"] = e.SourceStepID.String()
				}
				if e.TargetStepID != nil {
					edgeInfo["target_step_id"] = e.TargetStepID.String()
				}
				if e.Condition != nil {
					edgeInfo["condition"] = *e.Condition
				}
				edges = append(edges, edgeInfo)
			}

			return json.Marshal(map[string]interface{}{
				"id":          project.ID.String(),
				"name":        project.Name,
				"description": project.Description,
				"status":      project.Status,
				"version":     project.Version,
				"steps":       steps,
				"edges":       edges,
			})
		},
	}
}

func getWorkflowRunsTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "get_workflow_runs",
		Description: "ワークフローの実行履歴を取得します。エラー診断や改善提案のために使用します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"workflow_id": {
					"type": "string",
					"description": "ワークフローのUUID"
				},
				"status": {
					"type": "string",
					"description": "ステータスでフィルタ",
					"enum": ["pending", "running", "completed", "failed", "cancelled"]
				},
				"limit": {
					"type": "integer",
					"description": "取得件数（デフォルト: 10）"
				}
			},
			"required": ["workflow_id"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				WorkflowID string `json:"workflow_id"`
				Status     string `json:"status"`
				Limit      int    `json:"limit"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			if params.Limit == 0 {
				params.Limit = 10
			}

			workflowID, err := uuid.Parse(params.WorkflowID)
			if err != nil {
				return nil, fmt.Errorf("invalid workflow_id: %w", err)
			}

			filter := repository.RunFilter{
				Limit: params.Limit,
			}
			if params.Status != "" {
				status := domain.RunStatus(params.Status)
				filter.Status = &status
			}

			runs, _, err := deps.RunRepo.ListByProject(ctx, tenantID, workflowID, filter)
			if err != nil {
				return nil, fmt.Errorf("list runs: %w", err)
			}

			result := make([]map[string]interface{}, 0, len(runs))
			for _, r := range runs {
				result = append(result, map[string]interface{}{
					"id":           r.ID.String(),
					"status":       r.Status,
					"error":        r.Error,
					"started_at":   r.StartedAt,
					"completed_at": r.CompletedAt,
				})
			}

			return json.Marshal(map[string]interface{}{
				"runs":  result,
				"count": len(result),
			})
		},
	}
}

func searchDocumentationTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "search_documentation",
		Description: "プラットフォームのドキュメントを検索します。使い方や機能についての質問に答えるために使用します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "検索クエリ"
				},
				"topic": {
					"type": "string",
					"description": "トピックでフィルタ（workflow, blocks, integrations, best-practices）",
					"enum": ["workflow", "blocks", "integrations", "best-practices"]
				}
			},
			"required": ["query"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			var params struct {
				Query string `json:"query"`
				Topic string `json:"topic"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			// Comprehensive static documentation (will be replaced with RAG later)
			allDocs := getPlatformDocumentation()

			// Filter by topic if specified
			var docs []map[string]interface{}
			for _, doc := range allDocs {
				if params.Topic != "" && doc["topic"] != params.Topic {
					continue
				}
				// Simple keyword matching for now
				title, _ := doc["title"].(string)
				content, _ := doc["content"].(string)
				if containsKeyword(title, params.Query) || containsKeyword(content, params.Query) {
					docs = append(docs, doc)
				}
			}

			// If no matches, return all docs for the topic or top results
			if len(docs) == 0 {
				for _, doc := range allDocs {
					if params.Topic == "" || doc["topic"] == params.Topic {
						docs = append(docs, doc)
						if len(docs) >= 5 {
							break
						}
					}
				}
			}

			return json.Marshal(map[string]interface{}{
				"results": docs,
				"count":   len(docs),
			})
		},
	}
}

// containsKeyword checks if text contains keyword (case-insensitive)
func containsKeyword(text, keyword string) bool {
	return len(keyword) > 0 && len(text) > 0 &&
		(contains(text, keyword) || contains(text, strings.ToLower(keyword)))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(strings.ToLower(s), strings.ToLower(substr)))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getPlatformDocumentation returns comprehensive platform documentation
func getPlatformDocumentation() []map[string]interface{} {
	return []map[string]interface{}{
		// Workflow basics
		{
			"topic":   "workflow",
			"title":   "ワークフローの作成",
			"content": "ワークフローはステップとエッジで構成されます。ステップは処理単位、エッジはステップ間の接続を表します。新しいワークフローを作成するには、まずStartブロックを配置し、その後に必要な処理ブロックを追加してエッジで接続します。",
		},
		{
			"topic":   "workflow",
			"title":   "トリガーの設定",
			"content": "ワークフローは複数の方法でトリガーできます：手動実行、Webhook（HTTP POST）、スケジュール（Cron式）、イベント駆動。Webhookトリガーを使用すると外部システムからワークフローを起動できます。",
		},
		{
			"topic":   "workflow",
			"title":   "変数と入力データの扱い",
			"content": "ステップ間でデータを渡すには、前のステップの出力を参照します。{{previous_step.output.field_name}}の形式で参照できます。トリガーの入力は{{trigger.input}}で参照できます。",
		},

		// Blocks
		{
			"topic":   "blocks",
			"title":   "ブロックの種類",
			"content": "利用可能なブロックカテゴリ：AI（LLM、Embedding、Vision）、フロー制御（条件分岐、ループ、スイッチ）、アプリ連携（Slack、Discord、Notion、HTTP）、データ処理（JSON変換、テキスト処理）。",
		},
		{
			"topic":   "blocks",
			"title":   "LLMブロックの使い方",
			"content": "LLMブロック（llm-chat）を使用してAIテキスト生成を行います。設定項目：model（使用するモデル）、prompt（システムプロンプト）、temperature（創造性の度合い）、max_tokens（最大トークン数）。構造化出力が必要な場合はllm-structuredブロックを使用してください。",
		},
		{
			"topic":   "blocks",
			"title":   "条件分岐の設定",
			"content": "conditionブロックで条件分岐を実現します。condition設定に条件式を記述し、trueの場合とfalseの場合で異なるパスに分岐します。条件式は{{previous.output}} == 'value'のような形式で記述します。",
		},
		{
			"topic":   "blocks",
			"title":   "ループ処理（Map）",
			"content": "mapブロックで配列データに対してループ処理を実行します。items設定に配列データを指定し、各アイテムに対して指定したサブワークフローを実行します。並列実行も可能です。",
		},

		// Integrations
		{
			"topic":   "integrations",
			"title":   "Slack通知の送信",
			"content": "slack-sendブロックを使用してSlackチャンネルにメッセージを送信します。必要な設定：channel（チャンネル名または ID）、message（送信するメッセージ）。OAuth2認証が必要です。",
		},
		{
			"topic":   "integrations",
			"title":   "HTTP リクエスト",
			"content": "http-requestブロックで外部APIを呼び出します。設定：url（リクエストURL）、method（GET/POST/PUT/DELETE）、headers（ヘッダー）、body（リクエストボディ）。レスポンスはoutput.responseで参照できます。",
		},
		{
			"topic":   "integrations",
			"title":   "Discord連携",
			"content": "discord-sendブロックでDiscordチャンネルにメッセージを送信します。Webhook URLまたはBot Tokenを使用した認証が可能です。",
		},
		{
			"topic":   "integrations",
			"title":   "Notion連携",
			"content": "notion-create-pageブロックでNotionにページを作成します。database_id、title、contentを指定します。OAuth2認証が必要です。",
		},

		// Best practices
		{
			"topic":   "best-practices",
			"title":   "エラーハンドリング",
			"content": "重要な処理にはエラーハンドリングを追加してください。try-catchパターンとして、メイン処理の後にconditionブロックで{{previous.error}}をチェックし、エラー時の処理を分岐させます。",
		},
		{
			"topic":   "best-practices",
			"title":   "並列処理の活用",
			"content": "独立した複数の処理は並列実行できます。並列実行するには、同じソースステップから複数のエッジを異なるターゲットステップに接続します。これにより処理時間を短縮できます。",
		},
		{
			"topic":   "best-practices",
			"title":   "ワークフローのテスト",
			"content": "本番環境に展開する前に、テストデータでワークフローをテスト実行してください。Run Historyで実行結果を確認し、各ステップの入出力を検証します。",
		},
		{
			"topic":   "best-practices",
			"title":   "パフォーマンス最適化",
			"content": "大量データ処理時のヒント：1)並列処理を活用 2)不要なステップを削除 3)LLMの呼び出し回数を最小化 4)キャッシュ可能な処理はキャッシュを活用。",
		},
	}
}

// ============================================================================
// Analysis Tools (for Enhance mode)
// ============================================================================

func diagnoseWorkflowTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "diagnose_workflow",
		Description: "ワークフローを診断し、改善提案を行います。実行履歴、構造、パフォーマンスを分析して具体的な改善点を提示します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"workflow_id": {
					"type": "string",
					"description": "診断するワークフローのUUID"
				},
				"focus": {
					"type": "string",
					"description": "診断の焦点（all, errors, performance, structure）",
					"enum": ["all", "errors", "performance", "structure"]
				}
			},
			"required": ["workflow_id"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				WorkflowID string `json:"workflow_id"`
				Focus      string `json:"focus"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			if params.Focus == "" {
				params.Focus = "all"
			}

			workflowID, err := uuid.Parse(params.WorkflowID)
			if err != nil {
				return nil, fmt.Errorf("invalid workflow_id: %w", err)
			}

			// Get workflow with steps and edges
			project, err := deps.ProjectRepo.GetWithStepsAndEdges(ctx, tenantID, workflowID)
			if err != nil {
				return nil, fmt.Errorf("get workflow: %w", err)
			}

			// Get recent runs
			filter := repository.RunFilter{
				Limit: 20,
			}
			runs, _, err := deps.RunRepo.ListByProject(ctx, tenantID, workflowID, filter)
			if err != nil {
				return nil, fmt.Errorf("list runs: %w", err)
			}

			// Analyze and generate diagnosis
			diagnosis := analyzeWorkflow(project, runs, params.Focus)

			return json.Marshal(diagnosis)
		},
	}
}

// analyzeWorkflow performs comprehensive workflow analysis
func analyzeWorkflow(project *domain.Project, runs []*domain.Run, focus string) map[string]interface{} {
	result := map[string]interface{}{
		"workflow_id":   project.ID.String(),
		"workflow_name": project.Name,
		"issues":        []map[string]interface{}{},
		"suggestions":   []map[string]interface{}{},
		"metrics":       map[string]interface{}{},
	}

	issues := []map[string]interface{}{}
	suggestions := []map[string]interface{}{}

	// Structure analysis
	if focus == "all" || focus == "structure" {
		structureIssues, structureSuggestions := analyzeStructure(project)
		issues = append(issues, structureIssues...)
		suggestions = append(suggestions, structureSuggestions...)
	}

	// Error analysis
	if focus == "all" || focus == "errors" {
		errorIssues, errorSuggestions := analyzeErrors(runs)
		issues = append(issues, errorIssues...)
		suggestions = append(suggestions, errorSuggestions...)
	}

	// Performance analysis
	if focus == "all" || focus == "performance" {
		perfIssues, perfSuggestions := analyzePerformance(project, runs)
		issues = append(issues, perfIssues...)
		suggestions = append(suggestions, perfSuggestions...)
	}

	// Calculate metrics
	metrics := calculateMetrics(runs)

	result["issues"] = issues
	result["suggestions"] = suggestions
	result["metrics"] = metrics
	result["summary"] = generateSummary(len(issues), len(suggestions), metrics)

	return result
}

// analyzeStructure checks workflow structure for issues
func analyzeStructure(project *domain.Project) ([]map[string]interface{}, []map[string]interface{}) {
	var issues []map[string]interface{}
	var suggestions []map[string]interface{}

	// Check for entry point
	hasEntry := false
	for _, step := range project.Steps {
		if step.IsStartBlock() {
			hasEntry = true
			break
		}
	}
	if !hasEntry && len(project.Steps) > 0 {
		issues = append(issues, map[string]interface{}{
			"type":     "structure",
			"severity": "error",
			"message":  "エントリーポイント（Startブロック）が設定されていません",
			"fix":      "Startブロックを追加してください",
		})
	}

	// Check for disconnected steps
	connectedSteps := make(map[uuid.UUID]bool)
	for _, edge := range project.Edges {
		if edge.SourceStepID != nil {
			connectedSteps[*edge.SourceStepID] = true
		}
		if edge.TargetStepID != nil {
			connectedSteps[*edge.TargetStepID] = true
		}
	}
	for _, step := range project.Steps {
		if !step.IsStartBlock() && !connectedSteps[step.ID] && len(project.Steps) > 1 {
			issues = append(issues, map[string]interface{}{
				"type":     "structure",
				"severity": "warning",
				"message":  fmt.Sprintf("ステップ「%s」が孤立しています（他のステップと接続されていません）", step.Name),
				"step_id":  step.ID.String(),
				"fix":      "このステップをエッジで接続するか、不要な場合は削除してください",
			})
		}
	}

	// Check for potential parallel execution opportunities
	stepOutgoingEdges := make(map[uuid.UUID]int)
	for _, edge := range project.Edges {
		if edge.SourceStepID != nil {
			stepOutgoingEdges[*edge.SourceStepID]++
		}
	}
	parallelOpportunities := 0
	for _, step := range project.Steps {
		if stepOutgoingEdges[step.ID] > 1 {
			parallelOpportunities++
		}
	}
	if parallelOpportunities > 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":    "optimization",
			"message": fmt.Sprintf("%d箇所で並列処理が可能です", parallelOpportunities),
			"benefit": "処理時間を短縮できる可能性があります",
		})
	}

	// Check for error handling
	hasCondition := false
	for _, step := range project.Steps {
		if step.Type == domain.StepTypeCondition {
			hasCondition = true
			break
		}
	}
	if len(project.Steps) > 3 && !hasCondition {
		suggestions = append(suggestions, map[string]interface{}{
			"type":    "reliability",
			"message": "エラーハンドリングの追加を検討してください",
			"benefit": "エラー発生時の回復処理を追加することで、ワークフローの信頼性が向上します",
		})
	}

	return issues, suggestions
}

// analyzeErrors checks run history for error patterns
func analyzeErrors(runs []*domain.Run) ([]map[string]interface{}, []map[string]interface{}) {
	var issues []map[string]interface{}
	var suggestions []map[string]interface{}

	if len(runs) == 0 {
		return issues, suggestions
	}

	// Count failures
	failureCount := 0
	errorMessages := make(map[string]int)
	for _, run := range runs {
		if run.Status == domain.RunStatusFailed {
			failureCount++
			if run.Error != nil && *run.Error != "" {
				errorMessages[*run.Error]++
			}
		}
	}

	// Calculate failure rate
	failureRate := float64(failureCount) / float64(len(runs)) * 100

	if failureRate > 20 {
		issues = append(issues, map[string]interface{}{
			"type":     "reliability",
			"severity": "error",
			"message":  fmt.Sprintf("失敗率が高い（%.1f%%）", failureRate),
			"details":  fmt.Sprintf("直近%d回の実行のうち%d回が失敗しています", len(runs), failureCount),
		})
	} else if failureRate > 5 {
		issues = append(issues, map[string]interface{}{
			"type":     "reliability",
			"severity": "warning",
			"message":  fmt.Sprintf("失敗率がやや高い（%.1f%%）", failureRate),
			"details":  fmt.Sprintf("直近%d回の実行のうち%d回が失敗しています", len(runs), failureCount),
		})
	}

	// Find common errors
	for errMsg, count := range errorMessages {
		if count >= 2 {
			issues = append(issues, map[string]interface{}{
				"type":       "error_pattern",
				"severity":   "warning",
				"message":    fmt.Sprintf("繰り返し発生しているエラー（%d回）", count),
				"error_text": errMsg,
			})
			suggestions = append(suggestions, map[string]interface{}{
				"type":    "reliability",
				"message": "リトライロジックまたはエラーハンドリングの追加を検討してください",
				"reason":  fmt.Sprintf("同じエラーが%d回発生しています", count),
			})
		}
	}

	return issues, suggestions
}

// analyzePerformance checks for performance optimization opportunities
func analyzePerformance(project *domain.Project, runs []*domain.Run) ([]map[string]interface{}, []map[string]interface{}) {
	var issues []map[string]interface{}
	var suggestions []map[string]interface{}

	// Count LLM blocks (can be expensive)
	llmCount := 0
	for _, step := range project.Steps {
		if step.Type == domain.StepTypeLLM || strings.Contains(string(step.Type), "llm") {
			llmCount++
		}
	}

	if llmCount > 5 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":    "cost",
			"message": fmt.Sprintf("LLMブロックが%d個あります。コスト削減の余地があるかもしれません", llmCount),
			"benefit": "一部をより安価なモデル（gpt-4o-mini等）に変更することでコストを削減できます",
		})
	}

	// Check for sequential steps that could be parallelized
	if len(project.Steps) > 4 {
		// Count linear chains
		stepIncoming := make(map[uuid.UUID]int)
		stepOutgoing := make(map[uuid.UUID]int)
		for _, edge := range project.Edges {
			if edge.TargetStepID != nil {
				stepIncoming[*edge.TargetStepID]++
			}
			if edge.SourceStepID != nil {
				stepOutgoing[*edge.SourceStepID]++
			}
		}
		linearSteps := 0
		for _, step := range project.Steps {
			if stepIncoming[step.ID] == 1 && stepOutgoing[step.ID] == 1 {
				linearSteps++
			}
		}
		if linearSteps > 3 {
			suggestions = append(suggestions, map[string]interface{}{
				"type":    "performance",
				"message": "直線的なステップが多いです。並列化で高速化できる可能性があります",
				"benefit": "独立した処理を並列化することで全体の実行時間を短縮できます",
			})
		}
	}

	return issues, suggestions
}

// calculateMetrics calculates run metrics
func calculateMetrics(runs []*domain.Run) map[string]interface{} {
	if len(runs) == 0 {
		return map[string]interface{}{
			"total_runs":   0,
			"success_rate": 0,
		}
	}

	successCount := 0
	failedCount := 0
	for _, run := range runs {
		switch run.Status {
		case domain.RunStatusCompleted:
			successCount++
		case domain.RunStatusFailed:
			failedCount++
		}
	}

	return map[string]interface{}{
		"total_runs":    len(runs),
		"success_count": successCount,
		"failed_count":  failedCount,
		"success_rate":  float64(successCount) / float64(len(runs)) * 100,
	}
}

// generateSummary creates a human-readable summary
func generateSummary(issueCount, suggestionCount int, metrics map[string]interface{}) string {
	successRate, _ := metrics["success_rate"].(float64)

	var parts []string

	if issueCount == 0 {
		parts = append(parts, "問題は検出されませんでした")
	} else {
		parts = append(parts, fmt.Sprintf("%d件の問題が見つかりました", issueCount))
	}

	if suggestionCount > 0 {
		parts = append(parts, fmt.Sprintf("%d件の改善提案があります", suggestionCount))
	}

	if totalRuns, ok := metrics["total_runs"].(int); ok && totalRuns > 0 {
		parts = append(parts, fmt.Sprintf("成功率: %.1f%%", successRate))
	}

	return strings.Join(parts, "。")
}

// ============================================================================
// Workflow Manipulation Tools
// ============================================================================

func createStepTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "create_step",
		Description: "ワークフローに新しいステップを追加します。startタイプでエントリーポイントを作成できます。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"workflow_id": {
					"type": "string",
					"description": "ワークフローのUUID"
				},
				"name": {
					"type": "string",
					"description": "ステップの名前"
				},
				"type": {
					"type": "string",
					"description": "ステップのタイプ（start, llm, tool, condition, switch, map, etc.）。startタイプはエントリーポイントです。"
				},
				"block_definition_id": {
					"type": "string",
					"description": "ブロック定義のUUID（オプション）"
				},
				"config": {
					"type": "object",
					"description": "ステップの設定"
				},
				"position_x": {
					"type": "integer",
					"description": "X座標"
				},
				"position_y": {
					"type": "integer",
					"description": "Y座標"
				}
			},
			"required": ["workflow_id", "name", "type"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				WorkflowID        string                 `json:"workflow_id"`
				Name              string                 `json:"name"`
				Type              string                 `json:"type"`
				BlockDefinitionID string                 `json:"block_definition_id"`
				Config            map[string]interface{} `json:"config"`
				PositionX         int                    `json:"position_x"`
				PositionY         int                    `json:"position_y"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			workflowID, err := uuid.Parse(params.WorkflowID)
			if err != nil {
				return nil, fmt.Errorf("invalid workflow_id: %w", err)
			}

			configJSON, _ := json.Marshal(params.Config)

			step := &domain.Step{
				ID:        uuid.New(),
				TenantID:  tenantID,
				ProjectID: workflowID,
				Name:      params.Name,
				Type:      domain.StepType(params.Type),
				Config:    configJSON,
				PositionX: params.PositionX,
				PositionY: params.PositionY,
			}

			// Set block definition ID if provided
			if params.BlockDefinitionID != "" {
				blockDefID, err := uuid.Parse(params.BlockDefinitionID)
				if err != nil {
					return nil, fmt.Errorf("invalid block_definition_id: %w", err)
				}
				step.BlockDefinitionID = &blockDefID
			}

			if err := deps.StepRepo.Create(ctx, step); err != nil {
				return nil, fmt.Errorf("create step: %w", err)
			}

			return json.Marshal(map[string]interface{}{
				"id":       step.ID.String(),
				"name":     step.Name,
				"type":     step.Type,
				"is_entry": step.IsStartBlock(),
				"message":  "ステップを作成しました",
			})
		},
	}
}

func updateStepTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "update_step",
		Description: "既存のステップを更新します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"workflow_id": {
					"type": "string",
					"description": "ワークフローのUUID"
				},
				"step_id": {
					"type": "string",
					"description": "ステップのUUID"
				},
				"name": {
					"type": "string",
					"description": "新しい名前（オプション）"
				},
				"config": {
					"type": "object",
					"description": "新しい設定（オプション）"
				},
				"position_x": {
					"type": "integer",
					"description": "新しいX座標（オプション）"
				},
				"position_y": {
					"type": "integer",
					"description": "新しいY座標（オプション）"
				}
			},
			"required": ["workflow_id", "step_id"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				WorkflowID string                 `json:"workflow_id"`
				StepID     string                 `json:"step_id"`
				Name       *string                `json:"name"`
				Config     map[string]interface{} `json:"config"`
				PositionX  *int                   `json:"position_x"`
				PositionY  *int                   `json:"position_y"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			workflowID, err := uuid.Parse(params.WorkflowID)
			if err != nil {
				return nil, fmt.Errorf("invalid workflow_id: %w", err)
			}

			stepID, err := uuid.Parse(params.StepID)
			if err != nil {
				return nil, fmt.Errorf("invalid step_id: %w", err)
			}

			step, err := deps.StepRepo.GetByID(ctx, tenantID, workflowID, stepID)
			if err != nil {
				return nil, fmt.Errorf("get step: %w", err)
			}

			if params.Name != nil {
				step.Name = *params.Name
			}
			if params.Config != nil {
				configJSON, _ := json.Marshal(params.Config)
				step.Config = configJSON
			}
			if params.PositionX != nil {
				step.PositionX = *params.PositionX
			}
			if params.PositionY != nil {
				step.PositionY = *params.PositionY
			}

			if err := deps.StepRepo.Update(ctx, step); err != nil {
				return nil, fmt.Errorf("update step: %w", err)
			}

			return json.Marshal(map[string]interface{}{
				"id":      step.ID.String(),
				"name":    step.Name,
				"message": "ステップを更新しました",
			})
		},
	}
}

func deleteStepTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "delete_step",
		Description: "ステップを削除します。関連するエッジも自動的に削除されます。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"workflow_id": {
					"type": "string",
					"description": "ワークフローのUUID"
				},
				"step_id": {
					"type": "string",
					"description": "削除するステップのUUID"
				}
			},
			"required": ["workflow_id", "step_id"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				WorkflowID string `json:"workflow_id"`
				StepID     string `json:"step_id"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			workflowID, err := uuid.Parse(params.WorkflowID)
			if err != nil {
				return nil, fmt.Errorf("invalid workflow_id: %w", err)
			}

			stepID, err := uuid.Parse(params.StepID)
			if err != nil {
				return nil, fmt.Errorf("invalid step_id: %w", err)
			}

			if err := deps.StepRepo.Delete(ctx, tenantID, workflowID, stepID); err != nil {
				return nil, fmt.Errorf("delete step: %w", err)
			}

			return json.Marshal(map[string]interface{}{
				"id":      stepID.String(),
				"message": "ステップを削除しました",
			})
		},
	}
}

func createEdgeTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "create_edge",
		Description: "ステップ間の接続（エッジ）を作成します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"workflow_id": {
					"type": "string",
					"description": "ワークフローのUUID"
				},
				"source_step_id": {
					"type": "string",
					"description": "接続元ステップのUUID"
				},
				"target_step_id": {
					"type": "string",
					"description": "接続先ステップのUUID"
				},
				"source_port": {
					"type": "string",
					"description": "出力ポート名（default, true, false等）"
				},
				"condition": {
					"type": "string",
					"description": "エッジの条件式（オプション）"
				}
			},
			"required": ["workflow_id", "source_step_id", "target_step_id"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				WorkflowID   string `json:"workflow_id"`
				SourceStepID string `json:"source_step_id"`
				TargetStepID string `json:"target_step_id"`
				SourcePort   string `json:"source_port"`
				Condition    string `json:"condition"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			workflowID, err := uuid.Parse(params.WorkflowID)
			if err != nil {
				return nil, fmt.Errorf("invalid workflow_id: %w", err)
			}
			sourceStepID, err := uuid.Parse(params.SourceStepID)
			if err != nil {
				return nil, fmt.Errorf("invalid source_step_id: %w", err)
			}
			targetStepID, err := uuid.Parse(params.TargetStepID)
			if err != nil {
				return nil, fmt.Errorf("invalid target_step_id: %w", err)
			}

			sourcePort := params.SourcePort
			if sourcePort == "" {
				sourcePort = "default"
			}

			edge := &domain.Edge{
				ID:           uuid.New(),
				TenantID:     tenantID,
				ProjectID:    workflowID,
				SourceStepID: &sourceStepID,
				TargetStepID: &targetStepID,
				SourcePort:   sourcePort,
			}

			// Set condition if provided
			if params.Condition != "" {
				edge.Condition = &params.Condition
			}

			if err := deps.EdgeRepo.Create(ctx, edge); err != nil {
				return nil, fmt.Errorf("create edge: %w", err)
			}

			return json.Marshal(map[string]interface{}{
				"id":      edge.ID.String(),
				"message": "エッジを作成しました",
			})
		},
	}
}

func deleteEdgeTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "delete_edge",
		Description: "エッジを削除します。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"workflow_id": {
					"type": "string",
					"description": "ワークフローのUUID"
				},
				"edge_id": {
					"type": "string",
					"description": "削除するエッジのUUID"
				}
			},
			"required": ["workflow_id", "edge_id"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				WorkflowID string `json:"workflow_id"`
				EdgeID     string `json:"edge_id"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			workflowID, err := uuid.Parse(params.WorkflowID)
			if err != nil {
				return nil, fmt.Errorf("invalid workflow_id: %w", err)
			}

			edgeID, err := uuid.Parse(params.EdgeID)
			if err != nil {
				return nil, fmt.Errorf("invalid edge_id: %w", err)
			}

			if err := deps.EdgeRepo.Delete(ctx, tenantID, workflowID, edgeID); err != nil {
				return nil, fmt.Errorf("delete edge: %w", err)
			}

			return json.Marshal(map[string]interface{}{
				"id":      edgeID.String(),
				"message": "エッジを削除しました",
			})
		},
	}
}

// ============================================================================
// Validation Tools
// ============================================================================

func validateWorkflowTool(deps *Dependencies) *Tool {
	return &Tool{
		Name:        "validate_workflow",
		Description: "ワークフローの構造を検証します。エントリーポイント、エッジの整合性、設定の妥当性をチェックします。",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"workflow_id": {
					"type": "string",
					"description": "検証するワークフローのUUID"
				}
			},
			"required": ["workflow_id"]
		}`),
		Handler: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			tenantID, _ := ctx.Value(TenantIDKey).(uuid.UUID)

			var params struct {
				WorkflowID string `json:"workflow_id"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}

			workflowID, err := uuid.Parse(params.WorkflowID)
			if err != nil {
				return nil, fmt.Errorf("invalid workflow_id: %w", err)
			}

			project, err := deps.ProjectRepo.GetWithStepsAndEdges(ctx, tenantID, workflowID)
			if err != nil {
				return nil, fmt.Errorf("get workflow: %w", err)
			}

			var errors []string
			var warnings []string

			// Check for entry point (start block)
			hasEntry := false
			for _, step := range project.Steps {
				if step.IsStartBlock() {
					hasEntry = true
					break
				}
			}
			if !hasEntry && len(project.Steps) > 0 {
				errors = append(errors, "エントリーポイント（Startブロック）が設定されていません")
			}

			// Check for disconnected steps
			connectedSteps := make(map[uuid.UUID]bool)
			for _, edge := range project.Edges {
				if edge.SourceStepID != nil {
					connectedSteps[*edge.SourceStepID] = true
				}
				if edge.TargetStepID != nil {
					connectedSteps[*edge.TargetStepID] = true
				}
			}
			for _, step := range project.Steps {
				if !step.IsStartBlock() && !connectedSteps[step.ID] && len(project.Steps) > 1 {
					warnings = append(warnings, fmt.Sprintf("ステップ「%s」は他のステップと接続されていません", step.Name))
				}
			}

			// Check for cycles (simple check)
			// TODO: Implement proper cycle detection

			isValid := len(errors) == 0

			return json.Marshal(map[string]interface{}{
				"valid":    isValid,
				"errors":   errors,
				"warnings": warnings,
				"summary": map[string]interface{}{
					"steps_count": len(project.Steps),
					"edges_count": len(project.Edges),
					"has_entry":   hasEntry,
				},
			})
		},
	}
}
