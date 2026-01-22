package domain

// ErrorMessages contains localized error messages for API error codes
var ErrorMessages = map[string]LocalizedText{
	// HTTP standard errors
	"NOT_FOUND":         L("Resource not found", "リソースが見つかりません"),
	"UNAUTHORIZED":      L("Authentication required", "認証が必要です"),
	"FORBIDDEN":         L("Permission denied", "権限がありません"),
	"CONFLICT":          L("Resource conflict", "リソースが競合しています"),
	"VALIDATION_ERROR":  L("Validation failed", "検証に失敗しました"),
	"INTERNAL_ERROR":    L("Internal server error", "内部エラーが発生しました"),
	"BAD_REQUEST":       L("Invalid request", "リクエストが不正です"),
	"RATE_LIMIT":        L("Rate limit exceeded", "レート制限を超過しました"),

	// Project errors
	"PROJECT_NOT_FOUND":          L("Project not found", "プロジェクトが見つかりません"),
	"PROJECT_ALREADY_PUBLISHED":  L("Project is already published", "プロジェクトは既に公開されています"),
	"PROJECT_NOT_PUBLISHED":      L("Project is not published", "プロジェクトは公開されていません"),
	"PROJECT_NOT_EDITABLE":       L("Published project cannot be edited", "公開済みのプロジェクトは編集できません"),
	"PROJECT_HAS_CYCLE":          L("Project contains a cycle", "プロジェクトに循環参照があります"),
	"PROJECT_HAS_UNCONNECTED":    L("Project has unconnected steps", "プロジェクトに未接続のステップがあります"),
	"PROJECT_HAS_UNREACHABLE":    L("Project has unreachable steps", "プロジェクトに到達不能なステップがあります"),
	"PROJECT_BRANCH_OUTSIDE_GROUP": L("Branching blocks must be inside a Block Group", "分岐ブロックはグループ内に配置する必要があります"),
	"PROJECT_VERSION_NOT_FOUND":  L("Project version not found", "プロジェクトのバージョンが見つかりません"),

	// Step errors
	"STEP_NOT_FOUND":      L("Step not found", "ステップが見つかりません"),
	"INVALID_STEP_TYPE":   L("Invalid step type", "無効なステップタイプです"),
	"STEP_CONFIG_INVALID": L("Step configuration is invalid", "ステップの設定が無効です"),

	// Edge errors
	"EDGE_NOT_FOUND":       L("Edge not found", "エッジが見つかりません"),
	"EDGE_DUPLICATE":       L("Edge already exists", "エッジは既に存在します"),
	"EDGE_SELF_LOOP":       L("Edge cannot connect step to itself", "ステップを自身に接続することはできません"),
	"EDGE_CREATES_CYCLE":   L("Edge would create a cycle", "エッジを追加すると循環参照が発生します"),
	"EDGE_INVALID_PORT":    L("Invalid port specified", "無効なポートが指定されました"),
	"SOURCE_PORT_NOT_FOUND": L("Source port not found in block definition", "ブロック定義にソースポートが見つかりません"),

	// Run errors
	"RUN_NOT_FOUND":      L("Run not found", "実行が見つかりません"),
	"RUN_NOT_CANCELLABLE": L("Run cannot be cancelled", "実行をキャンセルできません"),
	"RUN_NOT_RESUMABLE":  L("Run cannot be resumed", "実行を再開できません"),
	"STEP_RUN_NOT_FOUND": L("Step run not found", "ステップ実行が見つかりません"),

	// Block Group errors
	"BLOCK_GROUP_NOT_FOUND":    L("Block group not found", "ブロックグループが見つかりません"),
	"BLOCK_GROUP_INVALID_TYPE": L("Invalid block group type", "無効なブロックグループタイプです"),
	"STEP_CANNOT_BE_IN_GROUP":  L("This step type cannot be added to a block group", "このステップタイプはグループに追加できません"),
	"BLOCK_GROUP_INVALID_ROLE": L("Invalid group role for this block group type", "無効なグループロールです"),

	// Schedule errors
	"SCHEDULE_NOT_FOUND":    L("Schedule not found", "スケジュールが見つかりません"),
	"SCHEDULE_INVALID_CRON": L("Invalid cron expression", "無効なcron式です"),
	"SCHEDULE_DISABLED":     L("Schedule is disabled", "スケジュールは無効です"),

	// Webhook errors
	"WEBHOOK_NOT_FOUND":      L("Webhook not found", "Webhookが見つかりません"),
	"WEBHOOK_DISABLED":       L("Webhook is disabled", "Webhookは無効です"),
	"WEBHOOK_INVALID_SECRET": L("Invalid webhook secret", "Webhookシークレットが無効です"),

	// Credential errors
	"CREDENTIAL_NOT_FOUND":       L("Credential not found", "認証情報が見つかりません"),
	"CREDENTIAL_EXPIRED":         L("Credential has expired", "認証情報の有効期限が切れています"),
	"CREDENTIAL_REVOKED":         L("Credential has been revoked", "認証情報が取り消されています"),
	"CREDENTIAL_INVALID_SCOPE":   L("Credential scope is invalid", "認証情報のスコープが無効です"),
	"CREDENTIAL_ACCESS_DENIED":   L("Access to credential denied", "認証情報へのアクセスが拒否されました"),
	"CREDENTIAL_BINDING_MISSING": L("Required credential binding not found", "必要な認証情報バインディングが見つかりません"),
	"CREDENTIAL_UNAVAILABLE":     L("Credential is unavailable", "認証情報が利用できません"),

	// OAuth2 errors
	"OAUTH2_PROVIDER_NOT_FOUND":   L("OAuth2 provider not found", "OAuth2プロバイダーが見つかりません"),
	"OAUTH2_APP_NOT_FOUND":        L("OAuth2 app not found", "OAuth2アプリが見つかりません"),
	"OAUTH2_APP_ALREADY_EXISTS":   L("OAuth2 app already exists", "OAuth2アプリは既に存在します"),
	"OAUTH2_CONNECTION_NOT_FOUND": L("OAuth2 connection not found", "OAuth2接続が見つかりません"),
	"OAUTH2_INVALID_STATE":        L("Invalid OAuth2 state parameter", "OAuth2 stateパラメータが無効です"),
	"OAUTH2_TOKEN_EXPIRED":        L("OAuth2 access token expired", "OAuth2アクセストークンの有効期限が切れています"),
	"OAUTH2_REFRESH_FAILED":       L("OAuth2 token refresh failed", "OAuth2トークンの更新に失敗しました"),

	// Block Definition errors
	"BLOCK_NOT_FOUND":       L("Block definition not found", "ブロック定義が見つかりません"),
	"BLOCK_SLUG_EXISTS":     L("Block definition slug already exists", "ブロックのスラッグは既に存在します"),
	"BLOCK_CODE_HIDDEN":     L("Block code is hidden for system blocks", "システムブロックのコードは非表示です"),
	"CIRCULAR_INHERITANCE":  L("Circular inheritance detected", "循環継承が検出されました"),
	"BLOCK_NOT_INHERITABLE": L("Block cannot be inherited", "このブロックは継承できません"),
	"INHERITANCE_DEPTH_EXCEEDED": L("Inheritance depth exceeded maximum limit", "継承の深さが最大制限を超えました"),
	"PARENT_BLOCK_NOT_FOUND":     L("Parent block not found", "親ブロックが見つかりません"),
	"INTERNAL_STEP_NOT_FOUND":    L("Internal step block not found", "内部ステップブロックが見つかりません"),

	// Copilot errors
	"COPILOT_SESSION_NOT_FOUND": L("Copilot session not found", "Copilotセッションが見つかりません"),

	// Template errors
	"TEMPLATE_NOT_FOUND": L("Template not found", "テンプレートが見つかりません"),

	// Git Sync errors
	"GIT_SYNC_NOT_FOUND": L("Git sync configuration not found", "Git同期設定が見つかりません"),

	// Block Package errors
	"BLOCK_PACKAGE_NOT_FOUND": L("Block package not found", "ブロックパッケージが見つかりません"),

	// Schema validation errors
	"SCHEMA_VALIDATION_ERROR": L("Input validation failed", "入力値の検証に失敗しました"),

	// Access errors
	"ACCESS_DENIED": L("Access denied", "アクセスが拒否されました"),
	"ALREADY_EXISTS": L("Resource already exists", "リソースは既に存在します"),
	"INVALID_STATE":  L("Invalid resource state", "リソースの状態が無効です"),
}

// GetErrorMessage returns the localized error message for the given code and language
func GetErrorMessage(lang, code string) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg.Get(lang)
	}
	return code
}

// CategoryNames contains localized names for block categories
var CategoryNames = map[BlockCategory]LocalizedText{
	BlockCategoryAI:     L("AI", "AI"),
	BlockCategoryFlow:   L("Flow", "フロー"),
	BlockCategoryApps:   L("Apps", "アプリ連携"),
	BlockCategoryCustom: L("Custom", "カスタム"),
}

// GetCategoryName returns the localized name for a category
func GetCategoryName(lang string, category BlockCategory) string {
	if name, ok := CategoryNames[category]; ok {
		return name.Get(lang)
	}
	return string(category)
}

// SubcategoryNames contains localized names for block subcategories
var SubcategoryNames = map[BlockSubcategory]LocalizedText{
	// AI subcategories
	BlockSubcategoryChat:    L("Chat", "チャット"),
	BlockSubcategoryRAG:     L("RAG", "RAG"),
	BlockSubcategoryRouting: L("Routing", "ルーティング"),
	BlockSubcategoryAgent:   L("Agent", "エージェント"),

	// Flow subcategories
	BlockSubcategoryBranching: L("Branching", "分岐"),
	BlockSubcategoryData:      L("Data", "データ"),
	BlockSubcategoryControl:   L("Control", "制御"),
	BlockSubcategoryUtility:   L("Utility", "ユーティリティ"),

	// Apps subcategories
	BlockSubcategorySlack:   L("Slack", "Slack"),
	BlockSubcategoryDiscord: L("Discord", "Discord"),
	BlockSubcategoryNotion:  L("Notion", "Notion"),
	BlockSubcategoryGitHub:  L("GitHub", "GitHub"),
	BlockSubcategoryGoogle:  L("Google", "Google"),
	BlockSubcategoryLinear:  L("Linear", "Linear"),
	BlockSubcategoryEmail:   L("Email", "メール"),
	BlockSubcategoryWeb:     L("Web", "Web"),
}

// GetSubcategoryName returns the localized name for a subcategory
func GetSubcategoryName(lang string, subcategory BlockSubcategory) string {
	if name, ok := SubcategoryNames[subcategory]; ok {
		return name.Get(lang)
	}
	return string(subcategory)
}

// GroupKindNames contains localized names for block group kinds
var GroupKindNames = map[BlockGroupKind]LocalizedText{
	BlockGroupKindParallel: L("Parallel", "並列実行"),
	BlockGroupKindTryCatch: L("Try-Catch", "エラーハンドリング"),
	BlockGroupKindForeach:  L("For Each", "繰り返し"),
	BlockGroupKindWhile:    L("While", "条件ループ"),
	BlockGroupKindAgent:    L("Agent", "エージェント"),
}

// GetGroupKindName returns the localized name for a group kind
func GetGroupKindName(lang string, kind BlockGroupKind) string {
	if name, ok := GroupKindNames[kind]; ok {
		return name.Get(lang)
	}
	return string(kind)
}
