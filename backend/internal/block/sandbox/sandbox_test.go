package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandbox_Execute_SimpleCode(t *testing.T) {
	sb := New(DefaultConfig())

	input := map[string]interface{}{
		"value": 10,
	}

	code := `return { result: input.value * 2 };`

	result, err := sb.Execute(context.Background(), code, input, nil)
	require.NoError(t, err)
	assert.EqualValues(t, 20, result["result"])
}

func TestSandbox_Execute_WithExecuteFunction(t *testing.T) {
	sb := New(DefaultConfig())

	input := map[string]interface{}{
		"name": "World",
	}

	code := `
function execute(input, context) {
	return { greeting: "Hello, " + input.name + "!" };
}
`

	result, err := sb.Execute(context.Background(), code, input, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result["greeting"])
}

func TestSandbox_Execute_WithCredentials(t *testing.T) {
	sb := New(DefaultConfig())

	input := map[string]interface{}{}

	execCtx := &ExecutionContext{
		Credentials: map[string]interface{}{
			"my_api": map[string]interface{}{
				"api_key": "sk-test-12345",
			},
		},
	}

	code := `return { key: context.credentials.my_api.api_key };`

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)
	assert.Equal(t, "sk-test-12345", result["key"])
}

func TestSandbox_Execute_Timeout(t *testing.T) {
	config := Config{
		Timeout:     100 * time.Millisecond,
		MemoryLimit: 128 * 1024 * 1024,
	}
	sb := New(config)

	input := map[string]interface{}{}

	// Infinite loop
	code := `while(true) {}`

	_, err := sb.Execute(context.Background(), code, input, nil)
	assert.ErrorIs(t, err, ErrTimeout)
}

func TestSandbox_Execute_InvalidCode(t *testing.T) {
	sb := New(DefaultConfig())

	_, err := sb.Execute(context.Background(), "", map[string]interface{}{}, nil)
	assert.ErrorIs(t, err, ErrInvalidCode)

	_, err = sb.Execute(context.Background(), "   ", map[string]interface{}{}, nil)
	assert.ErrorIs(t, err, ErrInvalidCode)
}

func TestSandbox_Execute_SyntaxError(t *testing.T) {
	sb := New(DefaultConfig())

	code := `return { invalid syntax here`

	_, err := sb.Execute(context.Background(), code, map[string]interface{}{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compilation error")
}

func TestSandbox_Execute_RuntimeError(t *testing.T) {
	sb := New(DefaultConfig())

	code := `throw new Error("test error");`

	_, err := sb.Execute(context.Background(), code, map[string]interface{}{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test error")
}

func TestSandbox_Execute_WithLogger(t *testing.T) {
	sb := New(DefaultConfig())

	var logged []interface{}
	execCtx := &ExecutionContext{
		Logger: func(args ...interface{}) {
			logged = append(logged, args...)
		},
	}

	code := `
console.log("test message");
context.log("context log");
return { done: true };
`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.Equal(t, true, result["done"])
	assert.Contains(t, logged, "test message")
	assert.Contains(t, logged, "context log")
}

func TestSandbox_Execute_ComplexDataTransformation(t *testing.T) {
	sb := New(DefaultConfig())

	input := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "a", "value": 1},
			map[string]interface{}{"name": "b", "value": 2},
			map[string]interface{}{"name": "c", "value": 3},
		},
	}

	code := `
function execute(input) {
	var total = 0;
	var names = [];
	for (var i = 0; i < input.items.length; i++) {
		total += input.items[i].value;
		names.push(input.items[i].name);
	}
	return {
		total: total,
		names: names,
		count: input.items.length
	};
}
`

	result, err := sb.Execute(context.Background(), code, input, nil)
	require.NoError(t, err)
	assert.EqualValues(t, 6, result["total"])
	assert.EqualValues(t, 3, result["count"])
}

func TestSandbox_Execute_HTTP(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Hello from server",
			"method":  r.Method,
		})
	}))
	defer server.Close()

	sb := New(DefaultConfig())

	execCtx := &ExecutionContext{
		HTTP: NewHTTPClient(10 * time.Second),
	}

	code := `
function execute(input, context) {
	var response = context.http.get(input.url);
	return {
		status: response.status,
		message: response.data.message
	};
}
`

	input := map[string]interface{}{
		"url": server.URL,
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)
	assert.EqualValues(t, 200, result["status"])
	assert.Equal(t, "Hello from server", result["message"])
}

func TestSandbox_Execute_HTTP_POST(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"received": body,
			"method":   r.Method,
		})
	}))
	defer server.Close()

	sb := New(DefaultConfig())

	execCtx := &ExecutionContext{
		HTTP: NewHTTPClient(10 * time.Second),
	}

	code := `
function execute(input, context) {
	var response = context.http.post(input.url, { name: "test", value: 42 });
	return {
		status: response.status,
		received: response.data.received
	};
}
`

	input := map[string]interface{}{
		"url": server.URL,
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)
	assert.EqualValues(t, 200, result["status"])

	received := result["received"].(map[string]interface{})
	assert.Equal(t, "test", received["name"])
	assert.EqualValues(t, 42, received["value"])
}

func TestSandbox_Execute_NilResult(t *testing.T) {
	sb := New(DefaultConfig())

	code := `var x = 1; // no return`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, nil)
	require.NoError(t, err)
	// Should return empty map for undefined result
	assert.Empty(t, result)
}

func TestSandbox_Execute_PrimitiveResult(t *testing.T) {
	sb := New(DefaultConfig())

	code := `return 42;`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, nil)
	require.NoError(t, err)
	// Primitive results should be wrapped
	assert.Equal(t, int64(42), result["result"])
}

// TestHTTPClient_SetHeader tests the SetHeader method
func TestHTTPClient_SetHeader(t *testing.T) {
	client := NewHTTPClient(10 * time.Second)

	client.SetHeader("Authorization", "Bearer token")
	client.SetHeader("X-Custom-Header", "custom-value")

	headers := client.getHeaders()
	assert.Equal(t, "Bearer token", headers["Authorization"])
	assert.Equal(t, "custom-value", headers["X-Custom-Header"])
}

// TestHTTPClient_SetHeader_Concurrent tests concurrent access to SetHeader and getHeaders
func TestHTTPClient_SetHeader_Concurrent(t *testing.T) {
	client := NewHTTPClient(10 * time.Second)

	// Run multiple goroutines writing and reading headers concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				client.SetHeader("Header", "value")
			}
			done <- true
		}()
		go func() {
			for j := 0; j < 100; j++ {
				_ = client.getHeaders()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 20; i++ {
		<-done
	}
}

// TestHTTPClient_Request tests the Request method with a mock server
func TestHTTPClient_Request(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify default header is present
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := NewHTTPClient(10 * time.Second)
	client.SetHeader("Authorization", "Bearer test-token")

	result, err := client.Request("POST", server.URL, map[string]string{"data": "test"}, nil)
	require.NoError(t, err)

	assert.Equal(t, 200, result["status"])
	data, ok := result["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "success", data["message"])
}

// TestHTTPClient_getHeaders_ReturnsCopy tests that getHeaders returns a copy
func TestHTTPClient_getHeaders_ReturnsCopy(t *testing.T) {
	client := NewHTTPClient(10 * time.Second)
	client.SetHeader("Key", "original")

	headers := client.getHeaders()
	headers["Key"] = "modified"

	// Original should be unchanged
	originalHeaders := client.getHeaders()
	assert.Equal(t, "original", originalHeaders["Key"])
}

// TestSanitizeError verifies that internal system information is removed from errors
func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes github path",
			input:    "Embedding failed: error at github.com/souta/ai-orchestration/internal/block/sandbox.(*Sandbox).setupGlobals.func11 (native)",
			expected: "Embedding failed: error",
		},
		{
			name:     "removes native suffix",
			input:    "some error (native)",
			expected: "some error",
		},
		{
			name:     "keeps simple error message",
			input:    "embedding provider (openai) is not configured",
			expected: "embedding provider (openai) is not configured",
		},
		{
			name:     "removes multiline stack traces",
			input:    "Error occurred\n\tat github.com/example/pkg.Function\n\tat github.com/example/pkg.Another",
			expected: "Error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.input)
			result := sanitizeError(err)
			assert.Equal(t, tt.expected, result.Error())
		})
	}

	// Test nil error
	assert.Nil(t, sanitizeError(nil))
}

// TestSandbox_Execute_AllServicesAccessible verifies that all expected services
// are accessible from JavaScript when ExecutionContext is fully initialized.
// This test prevents "undefined" errors when blocks try to access ctx.* properties.
func TestSandbox_Execute_AllServicesAccessible(t *testing.T) {
	sb := New(DefaultConfig())
	ctx := context.Background()

	// Create a fully initialized ExecutionContext (mirrors createSandboxContext in executor.go)
	execCtx := &ExecutionContext{
		HTTP:       NewHTTPClient(30 * time.Second),
		LLM:        NewLLMService(ctx),
		Embedding:  NewEmbeddingService(ctx),
		Workflow:   NewWorkflowService(),
		Human:      NewHumanService(),
		Adapter:    NewAdapterService(),
		Logger:     func(args ...interface{}) {},
	}

	// JavaScript code that checks if all expected services are defined
	code := `
function execute(input, context) {
	var services = {
		http: typeof context.http !== 'undefined' && typeof context.http.get === 'function',
		llm: typeof context.llm !== 'undefined' && typeof context.llm.chat === 'function',
		embedding: typeof context.embedding !== 'undefined' && typeof context.embedding.embed === 'function',
		workflow: typeof context.workflow !== 'undefined' && typeof context.workflow.run === 'function',
		human: typeof context.human !== 'undefined' && typeof context.human.requestApproval === 'function',
		adapter: typeof context.adapter !== 'undefined' && typeof context.adapter.call === 'function',
		log: typeof context.log === 'function'
	};

	// Find any missing services
	var missing = [];
	for (var name in services) {
		if (!services[name]) {
			missing.push(name);
		}
	}

	return {
		services: services,
		missing: missing,
		allAccessible: missing.length === 0
	};
}
`

	result, err := sb.Execute(ctx, code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)

	// Verify all services are accessible
	assert.True(t, result["allAccessible"].(bool), "Not all services are accessible: %v", result["missing"])

	// If test fails, show which services are missing
	if !result["allAccessible"].(bool) {
		missing := result["missing"].([]interface{})
		t.Errorf("Missing services: %v", missing)
	}
}

// TestSandbox_Execute_StubServicesReturnErrors verifies that stub services
// return appropriate errors when called, rather than silently failing.
func TestSandbox_Execute_StubServicesReturnErrors(t *testing.T) {
	sb := New(DefaultConfig())
	ctx := context.Background()

	// Test WorkflowService stub
	workflowSvc := NewWorkflowService()
	_, err := workflowSvc.Run("test-workflow", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")

	// Test HumanService stub
	humanSvc := NewHumanService()
	_, err = humanSvc.RequestApproval(map[string]interface{}{"instructions": "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")

	// Test AdapterService stub
	adapterSvc := NewAdapterService()
	_, err = adapterSvc.Call("test-adapter", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")

	// Test that these errors are surfaced to JavaScript
	execCtx := &ExecutionContext{
		Workflow: workflowSvc,
		Human:    humanSvc,
		Adapter:  adapterSvc,
	}

	// Test workflow.run from JavaScript - the error should be thrown and caught
	code := `
try {
	context.workflow.run("test", {});
	return { error: null, caught: false };
} catch (e) {
	return { error: String(e), caught: true };
}
`
	result, err := sb.Execute(ctx, code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	// Verify the error was caught
	assert.True(t, result["caught"].(bool), "Error should have been caught")
	if errStr, ok := result["error"].(string); ok && errStr != "" {
		assert.Contains(t, errStr, "not yet implemented")
	}
}

// ============================================================================
// Declarative Request/Response Tests
// ============================================================================

func TestExpandTemplate_Simple(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"owner": "octocat",
			"repo":  "hello-world",
		},
		Input: map[string]interface{}{
			"title": "Test Issue",
		},
		Credentials: map[string]interface{}{
			"github_token": "ghp_xxxx",
		},
	}

	// Config variable
	result := ExpandTemplate("https://api.github.com/repos/{{owner}}/{{repo}}/issues", ctx)
	assert.Equal(t, "https://api.github.com/repos/octocat/hello-world/issues", result)

	// Input variable
	result = ExpandTemplate("Issue: {{input.title}}", ctx)
	assert.Equal(t, "Issue: Test Issue", result)

	// Secret variable
	result = ExpandTemplate("Bearer {{secret.github_token}}", ctx)
	assert.Equal(t, "Bearer ghp_xxxx", result)
}

func TestExpandTemplate_NestedValues(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"api": map[string]interface{}{
				"baseUrl": "https://api.example.com",
				"version": "v1",
			},
		},
		Input: map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "John",
				"email": "john@example.com",
			},
		},
		Credentials: nil,
	}

	result := ExpandTemplate("{{api.baseUrl}}/{{api.version}}/users", ctx)
	assert.Equal(t, "https://api.example.com/v1/users", result)

	result = ExpandTemplate("Name: {{input.user.name}}, Email: {{input.user.email}}", ctx)
	assert.Equal(t, "Name: John, Email: john@example.com", result)
}

func TestExpandTemplate_MissingValue(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"existing": "value",
		},
		Input:       nil,
		Credentials: nil,
	}

	// Missing value should return empty string
	result := ExpandTemplate("{{missing}}", ctx)
	assert.Equal(t, "", result)

	// Mix of existing and missing
	result = ExpandTemplate("{{existing}}-{{missing}}", ctx)
	assert.Equal(t, "value-", result)
}

func TestExpandTemplateValue_Map(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"title": "My Issue",
			"body":  "Issue description",
		},
		Input:       nil,
		Credentials: nil,
	}

	body := map[string]interface{}{
		"title": "{{title}}",
		"body":  "{{body}}",
		"labels": []interface{}{
			"bug",
			"{{title}}",
		},
	}

	result := ExpandTemplateValue(body, ctx)
	resultMap := result.(map[string]interface{})

	assert.Equal(t, "My Issue", resultMap["title"])
	assert.Equal(t, "Issue description", resultMap["body"])

	labels := resultMap["labels"].([]interface{})
	assert.Equal(t, "bug", labels[0])
	assert.Equal(t, "My Issue", labels[1])
}

func TestSandbox_BuildDeclarativeRequest(t *testing.T) {
	sb := New(DefaultConfig())

	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"owner": "octocat",
			"repo":  "hello-world",
		},
		Input: map[string]interface{}{
			"title": "Test Issue",
			"body":  "This is a test",
		},
		Credentials: map[string]interface{}{
			"token": "ghp_xxxx",
		},
	}

	reqConfig := &domain.RequestConfig{
		URL:    "https://api.github.com/repos/{{owner}}/{{repo}}/issues",
		Method: "POST",
		Body: map[string]interface{}{
			"title": "{{input.title}}",
			"body":  "{{input.body}}",
		},
		Headers: map[string]string{
			"Authorization": "Bearer {{secret.token}}",
			"Accept":        "application/vnd.github+json",
		},
	}

	req, err := sb.BuildDeclarativeRequest(reqConfig, ctx)
	require.NoError(t, err)

	assert.Equal(t, "https://api.github.com/repos/octocat/hello-world/issues", req.URL.String())
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "Bearer ghp_xxxx", req.Header.Get("Authorization"))
	assert.Equal(t, "application/vnd.github+json", req.Header.Get("Accept"))
}

func TestSandbox_ProcessDeclarativeResponse_OutputMapping(t *testing.T) {
	sb := New(DefaultConfig())

	respConfig := &domain.ResponseConfig{
		OutputMapping: map[string]string{
			"issue_id":  "body.id",
			"issue_url": "body.html_url",
			"number":    "body.number",
			"success":   "true",
		},
		SuccessStatus: []int{200, 201},
	}

	// Create mock response
	resp := &http.Response{
		StatusCode: 201,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	respBody := []byte(`{"id": 12345, "html_url": "https://github.com/octocat/hello-world/issues/1", "number": 1}`)

	result, err := sb.ProcessDeclarativeResponse(respConfig, resp, respBody)
	require.NoError(t, err)

	assert.EqualValues(t, float64(12345), result["issue_id"])
	assert.Equal(t, "https://github.com/octocat/hello-world/issues/1", result["issue_url"])
	assert.EqualValues(t, float64(1), result["number"])
	assert.Equal(t, true, result["success"])
}

func TestSandbox_ProcessDeclarativeResponse_FailedStatus(t *testing.T) {
	sb := New(DefaultConfig())

	respConfig := &domain.ResponseConfig{
		SuccessStatus: []int{200, 201},
	}

	resp := &http.Response{
		StatusCode: 404,
		Header:     http.Header{},
	}
	respBody := []byte(`{"message": "Not Found"}`)

	_, err := sb.ProcessDeclarativeResponse(respConfig, resp, respBody)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestSandbox_ProcessDeclarativeResponse_DefaultSuccessRange(t *testing.T) {
	sb := New(DefaultConfig())

	// No SuccessStatus defined - should use default 200-299 range
	respConfig := &domain.ResponseConfig{
		OutputMapping: map[string]string{
			"data": "body",
		},
	}

	// Test 200
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
	}
	respBody := []byte(`{"test": "value"}`)

	result, err := sb.ProcessDeclarativeResponse(respConfig, resp, respBody)
	require.NoError(t, err)
	assert.NotNil(t, result["data"])

	// Test 204
	resp.StatusCode = 204
	_, err = sb.ProcessDeclarativeResponse(respConfig, resp, []byte{})
	require.NoError(t, err)

	// Test 300 should fail
	resp.StatusCode = 300
	_, err = sb.ProcessDeclarativeResponse(respConfig, resp, []byte{})
	require.Error(t, err)
}

func TestSandbox_DeclarativeHTTP_Integration(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repos/octocat/hello-world/issues", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Test Issue", body["title"])
		assert.Equal(t, "Issue body", body["body"])

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       12345,
			"html_url": "https://github.com/octocat/hello-world/issues/1",
			"number":   1,
		})
	}))
	defer server.Close()

	sb := New(DefaultConfig())

	reqConfig := &domain.RequestConfig{
		URL:    server.URL + "/repos/{{owner}}/{{repo}}/issues",
		Method: "POST",
		Body: map[string]interface{}{
			"title": "{{input.title}}",
			"body":  "{{input.body}}",
		},
		Headers: map[string]string{
			"Authorization": "Bearer {{secret.token}}",
		},
	}

	respConfig := &domain.ResponseConfig{
		OutputMapping: map[string]string{
			"id":  "body.id",
			"url": "body.html_url",
		},
		SuccessStatus: []int{200, 201},
	}

	declCtx := &DeclarativeContext{
		Config: map[string]interface{}{
			"owner": "octocat",
			"repo":  "hello-world",
		},
		Input: map[string]interface{}{
			"title": "Test Issue",
			"body":  "Issue body",
		},
		Credentials: map[string]interface{}{
			"token": "test-token",
		},
	}

	execCtx := &ExecutionContext{
		HTTP: NewHTTPClient(10 * time.Second),
	}

	result, err := sb.executeDeclarativeHTTP(reqConfig, respConfig, declCtx, execCtx)
	require.NoError(t, err)

	assert.EqualValues(t, float64(12345), result["id"])
	assert.Equal(t, "https://github.com/octocat/hello-world/issues/1", result["url"])
}

// ============================================================================
// URL Path Encoding Tests
// ============================================================================

func TestExpandTemplateForURLPath_BasicEncoding(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"spreadsheet_id": "abc123",
			"range":          "Sheet1!A1:B10",
		},
		Input:       map[string]interface{}{},
		Credentials: map[string]interface{}{},
	}

	// Range should be URL-encoded (! is encoded, : is allowed in path per RFC 3986)
	result := ExpandTemplateForURLPath("/{{spreadsheet_id}}/values/{{range}}", ctx)
	assert.Equal(t, "/abc123/values/Sheet1%21A1:B10", result)
}

func TestExpandTemplateForURLPath_SpacesAndSpecialChars(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"folder": "My Documents",
			"file":   "test file.txt",
		},
		Input:       map[string]interface{}{},
		Credentials: map[string]interface{}{},
	}

	result := ExpandTemplateForURLPath("/files/{{folder}}/{{file}}", ctx)
	assert.Equal(t, "/files/My%20Documents/test%20file.txt", result)
}

func TestExpandTemplateForURLPath_JapaneseChars(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"name": "テスト",
		},
		Input:       map[string]interface{}{},
		Credentials: map[string]interface{}{},
	}

	result := ExpandTemplateForURLPath("/users/{{name}}", ctx)
	// Japanese characters should be percent-encoded
	assert.Contains(t, result, "/users/")
	assert.NotContains(t, result, "テスト")
}

func TestExpandTemplateForURLPath_AlreadyEncodedValue(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"path": "already%20encoded",
		},
		Input:       map[string]interface{}{},
		Credentials: map[string]interface{}{},
	}

	// Already encoded value should not be double-encoded
	result := ExpandTemplateForURLPath("/files/{{path}}", ctx)
	assert.Equal(t, "/files/already%20encoded", result)
}

func TestExpandTemplateForURLPath_DoesNotDoubleEncode(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"encoded_slash": "%2F",
			"encoded_space": "hello%20world",
		},
		Input:       map[string]interface{}{},
		Credentials: map[string]interface{}{},
	}

	// Should not double-encode
	result := ExpandTemplateForURLPath("/path/{{encoded_slash}}/{{encoded_space}}", ctx)
	assert.Equal(t, "/path/%2F/hello%20world", result)
}

func TestExpandTemplateForURLPath_MixedEncodedAndUnencoded(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"id":    "123",                 // No encoding needed
			"range": "Sheet1!A:Z",          // Needs encoding (! is encoded, : is allowed)
			"name":  "already%20encoded",   // Already encoded
		},
		Input:       map[string]interface{}{},
		Credentials: map[string]interface{}{},
	}

	result := ExpandTemplateForURLPath("/spreadsheets/{{id}}/values/{{range}}/{{name}}", ctx)
	// id: 123 (no encoding needed)
	// range: Sheet1!A:Z -> Sheet1%21A:Z (: is allowed in path per RFC 3986)
	// name: already%20encoded (already encoded, should not double-encode)
	assert.Equal(t, "/spreadsheets/123/values/Sheet1%21A:Z/already%20encoded", result)
}

func TestIsAlreadyURLEncoded(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"hello", false},
		{"hello%20world", true},
		{"%2F", true},
		{"%20", true},
		{"%", false},
		{"%2", false},
		{"%GG", false},  // Invalid hex
		{"100%", false}, // Just a percent sign
		{"Sheet1!A:Z", false},
		{"already%2Fencoded", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isAlreadyURLEncoded(tt.input)
			assert.Equal(t, tt.expected, result, "isAlreadyURLEncoded(%q) = %v, want %v", tt.input, result, tt.expected)
		})
	}
}

func TestUrlEncodePathSegment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"hello world", "hello%20world"},
		{"Sheet1!A:Z", "Sheet1%21A:Z"},               // : is allowed in path per RFC 3986
		{"already%20encoded", "already%20encoded"},   // Should not double-encode
		{"path/to/file", "path%2Fto%2Ffile"},
		{"テスト", "%E3%83%86%E3%82%B9%E3%83%88"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := urlEncodePathSegment(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSandbox_BuildDeclarativeRequest_URLEncoding(t *testing.T) {
	sb := New(DefaultConfig())

	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"spreadsheet_id": "1abc-def-123",
			"range":          "Sheet1!A1:B10",
		},
		Input:       map[string]interface{}{},
		Credentials: map[string]interface{}{},
	}

	reqConfig := &domain.RequestConfig{
		URL:    "https://sheets.googleapis.com/v4/spreadsheets/{{spreadsheet_id}}/values/{{range}}",
		Method: "GET",
	}

	req, err := sb.BuildDeclarativeRequest(reqConfig, ctx)
	require.NoError(t, err)

	// The range should be URL-encoded in the path (! is encoded, : is allowed per RFC 3986)
	expectedURL := "https://sheets.googleapis.com/v4/spreadsheets/1abc-def-123/values/Sheet1%21A1:B10"
	assert.Equal(t, expectedURL, req.URL.String())
}

func TestExpandTemplateValue_OmitEmpty(t *testing.T) {
	ctx := &DeclarativeContext{
		Config: map[string]interface{}{
			"database_id": "db-123",
			"page_size":   100,
		},
		Input: map[string]interface{}{
			"filter": map[string]interface{}{
				"property": "status",
				"equals":   "done",
			},
			"sorts": []interface{}{},
		},
		Credentials: map[string]interface{}{},
	}

	t.Run("omit_empty removes empty string field", func(t *testing.T) {
		body := map[string]interface{}{
			"database_id": "{{database_id}}",
			"filter": map[string]interface{}{
				"value":      "{{input.missing}}",
				"omit_empty": true,
			},
			"page_size": "{{page_size}}",
		}

		result := ExpandTemplateValue(body, ctx).(map[string]interface{})

		assert.Equal(t, "db-123", result["database_id"])
		assert.Equal(t, 100, result["page_size"]) // Numeric values are preserved as-is
		assert.NotContains(t, result, "filter", "empty filter field should be omitted")
	})

	t.Run("omit_empty removes empty array field", func(t *testing.T) {
		body := map[string]interface{}{
			"database_id": "{{database_id}}",
			"sorts": map[string]interface{}{
				"value":      "{{input.sorts}}",
				"omit_empty": true,
			},
		}

		result := ExpandTemplateValue(body, ctx).(map[string]interface{})

		assert.Equal(t, "db-123", result["database_id"])
		assert.NotContains(t, result, "sorts", "empty sorts array should be omitted")
	})

	t.Run("omit_empty keeps non-empty object field", func(t *testing.T) {
		body := map[string]interface{}{
			"database_id": "{{database_id}}",
			"filter": map[string]interface{}{
				"value":      "{{input.filter}}",
				"omit_empty": true,
			},
		}

		result := ExpandTemplateValue(body, ctx).(map[string]interface{})

		assert.Equal(t, "db-123", result["database_id"])
		assert.Contains(t, result, "filter", "non-empty filter should be kept")
		filterMap := result["filter"].(map[string]interface{})
		assert.Equal(t, "status", filterMap["property"])
	})

	t.Run("without omit_empty keeps empty field", func(t *testing.T) {
		body := map[string]interface{}{
			"database_id": "{{database_id}}",
			"filter":      "{{input.missing}}", // no omit_empty
		}

		result := ExpandTemplateValue(body, ctx).(map[string]interface{})

		assert.Equal(t, "db-123", result["database_id"])
		assert.Contains(t, result, "filter", "empty field without omit_empty should be kept")
		assert.Equal(t, "", result["filter"])
	})

	t.Run("omit_empty=false keeps empty field", func(t *testing.T) {
		body := map[string]interface{}{
			"database_id": "{{database_id}}",
			"filter": map[string]interface{}{
				"value":      "{{input.missing}}",
				"omit_empty": false,
			},
		}

		result := ExpandTemplateValue(body, ctx).(map[string]interface{})

		assert.Equal(t, "db-123", result["database_id"])
		assert.Contains(t, result, "filter", "field with omit_empty=false should be kept")
		assert.Equal(t, "", result["filter"])
	})

	t.Run("complex nested structure with omit_empty", func(t *testing.T) {
		body := map[string]interface{}{
			"data": map[string]interface{}{
				"required": "{{database_id}}",
				"optional_filter": map[string]interface{}{
					"value":      "{{input.missing}}",
					"omit_empty": true,
				},
				"optional_sorts": map[string]interface{}{
					"value":      "{{input.sorts}}",
					"omit_empty": true,
				},
				"existing_filter": map[string]interface{}{
					"value":      "{{input.filter}}",
					"omit_empty": true,
				},
			},
		}

		result := ExpandTemplateValue(body, ctx).(map[string]interface{})
		dataMap := result["data"].(map[string]interface{})

		assert.Equal(t, "db-123", dataMap["required"])
		assert.NotContains(t, dataMap, "optional_filter")
		assert.NotContains(t, dataMap, "optional_sorts")
		assert.Contains(t, dataMap, "existing_filter")
	})
}

func TestIsEmptyValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"empty array", []interface{}{}, true},
		{"non-empty array", []interface{}{"a"}, false},
		{"empty map", map[string]interface{}{}, true},
		{"non-empty map", map[string]interface{}{"a": 1}, false},
		{"boolean true", true, false},
		{"boolean false", false, false}, // booleans are never empty
		{"number zero", float64(0), false},
		{"number non-zero", float64(42), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmptyValue(tt.value)
			assert.Equal(t, tt.expected, result, "isEmptyValue(%v) = %v, want %v", tt.value, result, tt.expected)
		})
	}
}
