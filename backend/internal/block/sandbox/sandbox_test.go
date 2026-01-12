package sandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestSandbox_Execute_WithCredential(t *testing.T) {
	sb := New(DefaultConfig())

	input := map[string]interface{}{}

	execCtx := &ExecutionContext{
		Credential: map[string]interface{}{
			"api_key": "sk-test-12345",
		},
	}

	code := `return { key: context.credential.api_key };`

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
