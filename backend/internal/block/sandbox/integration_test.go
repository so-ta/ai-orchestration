// Package sandbox provides integration tests for preset blocks.
// These tests require actual API keys and make real API calls.
//
// To run integration tests:
//
//	INTEGRATION_TEST=1 go test ./internal/block/sandbox/... -v -run Integration
//
// Required environment variables (in .env.test.local):
//   - SLACK_WEBHOOK_URL: Slack Incoming Webhook URL
//   - DISCORD_WEBHOOK_URL: Discord Webhook URL
//   - GITHUB_TOKEN: GitHub Personal Access Token
//   - NOTION_API_KEY: Notion Integration Token
//   - LINEAR_API_KEY: Linear API Key
//   - SENDGRID_API_KEY: SendGrid API Key
//   - TAVILY_API_KEY: Tavily API Key
package sandbox

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestSandbox creates a sandbox with HTTP client for testing
func createTestSandbox() (*Sandbox, *ExecutionContext) {
	sandbox := New(DefaultConfig())
	httpClient := NewHTTPClient(30 * time.Second)

	execCtx := &ExecutionContext{
		HTTP: httpClient,
		Logger: func(args ...interface{}) {
			// No-op for tests
		},
	}

	return sandbox, execCtx
}

// =============================================================================
// Slack Integration Tests
// =============================================================================

func TestSlackBlock_Integration_SendMessage(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	webhookURL := testutil.RequireEnvVar(t, "SLACK_WEBHOOK_URL")

	sandbox, execCtx := createTestSandbox()

	// Slack webhook code (simplified from block definition)
	code := `
const payload = {
    text: "Integration Test: " + (input.message || "Hello from test!")
};
const response = ctx.http.post(input.webhook_url, payload);
if (response.status >= 400) {
    throw new Error('Slack send failed: ' + response.status);
}
return { success: true, status: response.status };
`

	input := map[string]interface{}{
		"webhook_url": webhookURL,
		"message":     "Integration test at " + time.Now().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "Slack message should be sent successfully")
	assert.True(t, result["success"].(bool))
	t.Logf("Slack message sent successfully, status: %v", result["status"])
}

// =============================================================================
// Discord Integration Tests
// =============================================================================

func TestDiscordBlock_Integration_SendMessage(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	webhookURL := testutil.RequireEnvVar(t, "DISCORD_WEBHOOK_URL")

	sandbox, execCtx := createTestSandbox()

	code := `
const payload = {
    content: input.message || "Hello from integration test!"
};
if (input.username) payload.username = input.username;
const response = ctx.http.post(input.webhook_url, payload);
if (response.status === 429) {
    throw new Error('Discord rate limited');
}
if (response.status >= 400) {
    throw new Error('Discord send failed: ' + response.status);
}
return { success: true, status: response.status };
`

	input := map[string]interface{}{
		"webhook_url": webhookURL,
		"message":     "Integration test at " + time.Now().Format(time.RFC3339),
		"username":    "Integration Test Bot",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "Discord message should be sent successfully")
	assert.True(t, result["success"].(bool))
	t.Logf("Discord message sent successfully, status: %v", result["status"])
}

// =============================================================================
// GitHub Integration Tests
// =============================================================================

func TestGitHubBlock_Integration_GetUser(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	token := testutil.RequireEnvVar(t, "GITHUB_TOKEN")

	sandbox, execCtx := createTestSandbox()

	// Test getting authenticated user info
	code := `
const response = ctx.http.get('https://api.github.com/user', {
    headers: {
        'Authorization': 'Bearer ' + input.token,
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28'
    }
});
if (response.status >= 400) {
    throw new Error('GitHub API error: ' + response.status);
}
return {
    login: response.data.login,
    name: response.data.name,
    id: response.data.id
};
`

	input := map[string]interface{}{
		"token": token,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "GitHub API call should succeed")
	assert.NotEmpty(t, result["login"], "Should return user login")
	t.Logf("GitHub user: %s (ID: %v)", result["login"], result["id"])
}

func TestGitHubBlock_Integration_ListRepos(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	token := testutil.RequireEnvVar(t, "GITHUB_TOKEN")

	sandbox, execCtx := createTestSandbox()

	code := `
const response = ctx.http.get('https://api.github.com/user/repos', {
    headers: {
        'Authorization': 'Bearer ' + input.token,
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28'
    },
    params: {
        per_page: 5,
        sort: 'updated'
    }
});
if (response.status >= 400) {
    throw new Error('GitHub API error: ' + response.status);
}
return {
    count: response.data.length,
    repos: response.data.map(r => ({ name: r.name, full_name: r.full_name }))
};
`

	input := map[string]interface{}{
		"token": token,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "GitHub repos list should succeed")
	t.Logf("Found %v repositories", result["count"])
}

// =============================================================================
// Notion Integration Tests
// =============================================================================

func TestNotionBlock_Integration_ListUsers(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "NOTION_API_KEY")

	sandbox, execCtx := createTestSandbox()

	code := `
const response = ctx.http.get('https://api.notion.com/v1/users', {
    headers: {
        'Authorization': 'Bearer ' + input.api_key,
        'Notion-Version': '2022-06-28'
    }
});
if (response.status >= 400) {
    throw new Error('Notion API error: ' + response.status + ' - ' + JSON.stringify(response.data));
}
return {
    users: response.data.results.map(u => ({
        id: u.id,
        type: u.type,
        name: u.name
    })),
    count: response.data.results.length
};
`

	input := map[string]interface{}{
		"api_key": apiKey,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "Notion API call should succeed")
	t.Logf("Found %v users in Notion workspace", result["count"])
}

func TestNotionBlock_Integration_Search(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "NOTION_API_KEY")

	sandbox, execCtx := createTestSandbox()

	code := `
const response = ctx.http.post('https://api.notion.com/v1/search', {
    page_size: 5
}, {
    headers: {
        'Authorization': 'Bearer ' + input.api_key,
        'Notion-Version': '2022-06-28'
    }
});
if (response.status >= 400) {
    throw new Error('Notion API error: ' + response.status);
}
return {
    results: response.data.results.map(r => ({
        id: r.id,
        type: r.object,
        title: r.properties?.title?.title?.[0]?.plain_text || r.title?.[0]?.plain_text || 'Untitled'
    })),
    count: response.data.results.length,
    has_more: response.data.has_more
};
`

	input := map[string]interface{}{
		"api_key": apiKey,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "Notion search should succeed")
	t.Logf("Found %v items in Notion", result["count"])
}

// =============================================================================
// Linear Integration Tests
// =============================================================================

func TestLinearBlock_Integration_GetViewer(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "LINEAR_API_KEY")

	sandbox, execCtx := createTestSandbox()

	code := `
const query = '{ viewer { id name email } }';
const response = ctx.http.post('https://api.linear.app/graphql', {
    query: query
}, {
    headers: {
        'Authorization': input.api_key,
        'Content-Type': 'application/json'
    }
});
if (response.status >= 400) {
    throw new Error('Linear API error: ' + response.status);
}
if (response.data.errors) {
    throw new Error('Linear GraphQL error: ' + response.data.errors[0].message);
}
return response.data.data.viewer;
`

	input := map[string]interface{}{
		"api_key": apiKey,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "Linear API call should succeed")
	assert.NotEmpty(t, result["id"], "Should return viewer ID")
	t.Logf("Linear user: %s (%s)", result["name"], result["email"])
}

func TestLinearBlock_Integration_ListTeams(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "LINEAR_API_KEY")

	sandbox, execCtx := createTestSandbox()

	code := `
const query = '{ teams { nodes { id name key } } }';
const response = ctx.http.post('https://api.linear.app/graphql', {
    query: query
}, {
    headers: {
        'Authorization': input.api_key,
        'Content-Type': 'application/json'
    }
});
if (response.status >= 400) {
    throw new Error('Linear API error: ' + response.status);
}
if (response.data.errors) {
    throw new Error('Linear GraphQL error: ' + response.data.errors[0].message);
}
return {
    teams: response.data.data.teams.nodes,
    count: response.data.data.teams.nodes.length
};
`

	input := map[string]interface{}{
		"api_key": apiKey,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "Linear teams list should succeed")
	t.Logf("Found %v teams in Linear", result["count"])
}

// =============================================================================
// SendGrid Integration Tests
// =============================================================================

func TestSendGridBlock_Integration_ValidateAPIKey(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "SENDGRID_API_KEY")

	sandbox, execCtx := createTestSandbox()

	// Test API key validation by getting API key scopes
	code := `
const response = ctx.http.get('https://api.sendgrid.com/v3/scopes', {
    headers: {
        'Authorization': 'Bearer ' + input.api_key
    }
});
if (response.status === 401) {
    throw new Error('Invalid API key');
}
if (response.status >= 400) {
    throw new Error('SendGrid API error: ' + response.status);
}
return {
    valid: true,
    scopes: response.data.scopes ? response.data.scopes.slice(0, 5) : []
};
`

	input := map[string]interface{}{
		"api_key": apiKey,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "SendGrid API key should be valid")
	assert.True(t, result["valid"].(bool))
	t.Logf("SendGrid API key validated, scopes: %v", result["scopes"])
}

// Note: Actual email sending test is commented out to avoid sending real emails
// func TestSendGridBlock_Integration_SendEmail(t *testing.T) { ... }

// =============================================================================
// Tavily Web Search Integration Tests
// =============================================================================

func TestTavilyBlock_Integration_Search(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "TAVILY_API_KEY")

	sandbox, execCtx := createTestSandbox()

	code := `
const response = ctx.http.post('https://api.tavily.com/search', {
    api_key: input.api_key,
    query: input.query,
    search_depth: 'basic',
    max_results: 3,
    include_answer: true
});
if (response.status >= 400) {
    throw new Error('Tavily API error: ' + response.status + ' - ' + JSON.stringify(response.data));
}
return {
    answer: response.data.answer,
    results: response.data.results.map(r => ({
        title: r.title,
        url: r.url,
        score: r.score
    })),
    count: response.data.results.length
};
`

	input := map[string]interface{}{
		"api_key": apiKey,
		"query":   "What is the capital of Japan?",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "Tavily search should succeed")
	assert.NotEmpty(t, result["answer"], "Should return an AI answer")
	t.Logf("Tavily answer: %s", result["answer"])
	t.Logf("Found %v search results", result["count"])
}

// =============================================================================
// Google Sheets Integration Tests (requires service account)
// =============================================================================

func TestGoogleSheetsBlock_Integration_GetSpreadsheet(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "GOOGLE_API_KEY")
	spreadsheetID := os.Getenv("GOOGLE_TEST_SPREADSHEET_ID")
	if spreadsheetID == "" {
		t.Skip("Skipping: GOOGLE_TEST_SPREADSHEET_ID not set")
	}

	sandbox, execCtx := createTestSandbox()

	code := `
const url = 'https://sheets.googleapis.com/v4/spreadsheets/' + input.spreadsheet_id + '?key=' + input.api_key;
const response = ctx.http.get(url);
if (response.status >= 400) {
    throw new Error('Google Sheets API error: ' + response.status + ' - ' + JSON.stringify(response.data));
}
return {
    title: response.data.properties.title,
    sheets: response.data.sheets.map(s => s.properties.title)
};
`

	input := map[string]interface{}{
		"api_key":        apiKey,
		"spreadsheet_id": spreadsheetID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, code, input, execCtx)

	require.NoError(t, err, "Google Sheets API call should succeed")
	t.Logf("Spreadsheet: %s, Sheets: %v", result["title"], result["sheets"])
}

// =============================================================================
// Declarative Block Integration Tests
// =============================================================================

func TestDeclarativeBlock_Integration_GitHub_GetUser(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	token := testutil.RequireEnvVar(t, "GITHUB_TOKEN")

	// Use declarative configuration instead of JS code
	block := &domain.BlockDefinition{
		Slug: "github_get_user",
		Request: &domain.RequestConfig{
			URL:    "https://api.github.com/user",
			Method: "GET",
			Headers: map[string]string{
				"Authorization":       "Bearer {{secret.token}}",
				"Accept":              "application/vnd.github+json",
				"X-GitHub-Api-Version": "2022-11-28",
				"User-Agent":          "AI-Orchestration-Test",
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200},
			OutputMapping: map[string]string{
				"login": "body.login",
				"id":    "body.id",
				"name":  "body.name",
			},
		},
	}

	sandbox, execCtx := createTestSandbox()
	execCtx.Credentials = map[string]interface{}{
		"token": token,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.ExecuteWithDeclarative(ctx, block, map[string]interface{}{}, map[string]interface{}{}, execCtx)

	require.NoError(t, err, "Declarative GitHub API call should succeed")
	assert.NotEmpty(t, result["login"], "Should return user login")
	t.Logf("Declarative GitHub user: %s (ID: %v)", result["login"], result["id"])
}

func TestDeclarativeBlock_Integration_GitHub_ListRepos(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	token := testutil.RequireEnvVar(t, "GITHUB_TOKEN")

	block := &domain.BlockDefinition{
		Slug: "github_list_repos",
		Request: &domain.RequestConfig{
			URL:    "https://api.github.com/user/repos",
			Method: "GET",
			Headers: map[string]string{
				"Authorization":       "Bearer {{secret.token}}",
				"Accept":              "application/vnd.github+json",
				"X-GitHub-Api-Version": "2022-11-28",
				"User-Agent":          "AI-Orchestration-Test",
			},
			QueryParams: map[string]string{
				"per_page": "{{per_page}}",
				"sort":     "updated",
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200},
			OutputMapping: map[string]string{
				"repos":  "body",
				"status": "status",
			},
		},
	}

	sandbox, execCtx := createTestSandbox()
	execCtx.Credentials = map[string]interface{}{
		"token": token,
	}

	config := map[string]interface{}{
		"per_page": "5",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.ExecuteWithDeclarative(ctx, block, config, map[string]interface{}{}, execCtx)

	require.NoError(t, err, "Declarative GitHub repos list should succeed")
	repos, ok := result["repos"].([]interface{})
	require.True(t, ok, "repos should be an array")
	t.Logf("Found %d repositories (declarative)", len(repos))
}

func TestDeclarativeBlock_Integration_Slack_SendMessage(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	webhookURL := testutil.RequireEnvVar(t, "SLACK_WEBHOOK_URL")

	block := &domain.BlockDefinition{
		Slug: "slack_declarative",
		Request: &domain.RequestConfig{
			URL:    webhookURL,
			Method: "POST",
			Body: map[string]interface{}{
				"text": "{{input.message}}",
				"blocks": []interface{}{
					map[string]interface{}{
						"type": "section",
						"text": map[string]interface{}{
							"type": "mrkdwn",
							"text": "*Declarative Integration Test*\n{{input.message}}",
						},
					},
				},
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200},
			OutputMapping: map[string]string{
				"success": "true",
				"status":  "status",
			},
		},
	}

	sandbox, execCtx := createTestSandbox()

	input := map[string]interface{}{
		"message": "Declarative test at " + time.Now().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.ExecuteWithDeclarative(ctx, block, map[string]interface{}{}, input, execCtx)

	require.NoError(t, err, "Declarative Slack message should be sent successfully")
	assert.Equal(t, true, result["success"])
	t.Logf("Declarative Slack message sent successfully")
}

func TestDeclarativeBlock_Integration_Discord_SendMessage(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	webhookURL := testutil.RequireEnvVar(t, "DISCORD_WEBHOOK_URL")

	block := &domain.BlockDefinition{
		Slug: "discord_declarative",
		Request: &domain.RequestConfig{
			URL:    webhookURL,
			Method: "POST",
			Body: map[string]interface{}{
				"content":  "{{input.message}}",
				"username": "Declarative Test Bot",
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200, 204},
			OutputMapping: map[string]string{
				"success": "true",
			},
		},
	}

	sandbox, execCtx := createTestSandbox()

	input := map[string]interface{}{
		"message": "Declarative test at " + time.Now().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.ExecuteWithDeclarative(ctx, block, map[string]interface{}{}, input, execCtx)

	require.NoError(t, err, "Declarative Discord message should be sent successfully")
	assert.Equal(t, true, result["success"])
	t.Logf("Declarative Discord message sent successfully")
}

func TestDeclarativeBlock_Integration_PublicAPI(t *testing.T) {
	testutil.SkipIfNotIntegration(t)

	// Test against JSONPlaceholder (public API - always available)
	block := &domain.BlockDefinition{
		Slug: "jsonplaceholder_posts",
		Request: &domain.RequestConfig{
			URL:    "https://jsonplaceholder.typicode.com/posts",
			Method: "GET",
			QueryParams: map[string]string{
				"_limit": "{{limit}}",
			},
		},
		Response: &domain.ResponseConfig{
			SuccessStatus: []int{200},
			OutputMapping: map[string]string{
				"posts":  "body",
				"status": "status",
			},
		},
	}

	sandbox, execCtx := createTestSandbox()

	config := map[string]interface{}{
		"limit": "5",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sandbox.ExecuteWithDeclarative(ctx, block, config, map[string]interface{}{}, execCtx)

	require.NoError(t, err, "Declarative public API call should succeed")
	posts, ok := result["posts"].([]interface{})
	require.True(t, ok, "posts should be an array")
	assert.LessOrEqual(t, len(posts), 5)
	t.Logf("Declarative public API: Found %d posts", len(posts))
}

// =============================================================================
// Environment Status Test
// =============================================================================

func TestBlocks_Integration_AllAvailable(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)

	services := []struct {
		name   string
		envKey string
	}{
		{"Slack", "SLACK_WEBHOOK_URL"},
		{"Discord", "DISCORD_WEBHOOK_URL"},
		{"GitHub", "GITHUB_TOKEN"},
		{"Notion", "NOTION_API_KEY"},
		{"Linear", "LINEAR_API_KEY"},
		{"SendGrid", "SENDGRID_API_KEY"},
		{"Tavily", "TAVILY_API_KEY"},
		{"Google API", "GOOGLE_API_KEY"},
	}

	t.Log("=== Block Integration Test Environment ===")
	for _, svc := range services {
		status := "NOT CONFIGURED"
		if os.Getenv(svc.envKey) != "" {
			status = "Available"
		}
		t.Logf("  %s: %s", svc.name, status)
	}
}
