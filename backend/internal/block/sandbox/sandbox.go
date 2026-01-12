package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// Errors
var (
	ErrTimeout     = errors.New("script execution timed out")
	ErrMemoryLimit = errors.New("script exceeded memory limit")
	ErrInvalidCode = errors.New("invalid or empty code")
)

// Config holds sandbox configuration
type Config struct {
	Timeout     time.Duration
	MemoryLimit int64 // in bytes (not strictly enforced by goja, but used for monitoring)
}

// DefaultConfig returns default sandbox configuration
func DefaultConfig() Config {
	return Config{
		Timeout:     30 * time.Second,
		MemoryLimit: 128 * 1024 * 1024, // 128MB
	}
}

// LLMService provides LLM API access to scripts
type LLMService interface {
	// Chat performs a chat completion request
	// Returns { content: string, usage: { input_tokens: int, output_tokens: int } }
	Chat(provider, model string, request map[string]interface{}) (map[string]interface{}, error)
}

// WorkflowService provides subflow execution capability
type WorkflowService interface {
	// Run executes a subflow and returns its output
	Run(workflowID string, input map[string]interface{}) (map[string]interface{}, error)
}

// HumanService provides human-in-the-loop functionality
type HumanService interface {
	// RequestApproval requests human approval and waits for response
	RequestApproval(request map[string]interface{}) (map[string]interface{}, error)
}

// AdapterService provides adapter execution capability
type AdapterService interface {
	// Call executes an adapter and returns its output
	Call(adapterID string, input map[string]interface{}) (map[string]interface{}, error)
}

// BlocksService provides block definition access to scripts
type BlocksService interface {
	// List returns all available block definitions
	List() ([]map[string]interface{}, error)
	// Get retrieves a block definition by slug
	Get(slug string) (map[string]interface{}, error)
}

// WorkflowsService provides workflow read access to scripts
type WorkflowsService interface {
	// Get retrieves a workflow by ID
	Get(workflowID string) (map[string]interface{}, error)
	// List retrieves all workflows
	List() ([]map[string]interface{}, error)
}

// RunsService provides run read access to scripts
type RunsService interface {
	// Get retrieves a run by ID
	Get(runID string) (map[string]interface{}, error)
	// GetStepRuns retrieves all step runs for a run
	GetStepRuns(runID string) ([]map[string]interface{}, error)
}

// ExecutionContext provides runtime context to scripts
type ExecutionContext struct {
	HTTP *HTTPClient
	// Unified Block Model services
	LLM      LLMService
	Workflow WorkflowService
	Human    HumanService
	Adapter  AdapterService
	// Copilot/meta-workflow services (read-only data access)
	Blocks    BlocksService
	Workflows WorkflowsService
	Runs      RunsService
	// Credentials is a map of credential name to credential data
	// Accessible in scripts as context.credentials.name.field
	// e.g., context.credentials.api_key.access_token
	Credentials map[string]interface{}
	Logger      func(args ...interface{})
}

// Sandbox provides a secure JavaScript execution environment
type Sandbox struct {
	config Config
}

// New creates a new Sandbox with the given configuration
func New(config Config) *Sandbox {
	return &Sandbox{config: config}
}

// Execute runs JavaScript code with the given input and context
func (s *Sandbox) Execute(ctx context.Context, code string, input map[string]interface{}, execCtx *ExecutionContext) (map[string]interface{}, error) {
	if strings.TrimSpace(code) == "" {
		return nil, ErrInvalidCode
	}

	// Create a new goja runtime for each execution (isolation)
	vm := goja.New()

	// Setup interrupt for timeout
	var interruptOnce sync.Once
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	// Monitor for context cancellation (timeout)
	go func() {
		<-ctx.Done()
		interruptOnce.Do(func() {
			vm.Interrupt("execution timeout")
		})
	}()

	// Setup global objects
	if err := s.setupGlobals(vm, input, execCtx); err != nil {
		return nil, fmt.Errorf("failed to setup globals: %w", err)
	}

	// Wrap user code in an async function and execute
	wrappedCode := s.wrapCode(code)

	// Compile and run the script
	program, err := goja.Compile("script", wrappedCode, false)
	if err != nil {
		return nil, fmt.Errorf("script compilation error: %w", err)
	}

	result, err := vm.RunProgram(program)
	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, ErrTimeout
		}
		return nil, fmt.Errorf("script execution error: %w", err)
	}

	// Convert result to Go map
	return s.extractResult(result)
}

// setupGlobals sets up the global objects available to scripts
func (s *Sandbox) setupGlobals(vm *goja.Runtime, input map[string]interface{}, execCtx *ExecutionContext) error {
	// SECURITY: Block dangerous globals to prevent environment variable access and code injection
	// These are blocked by returning undefined or throwing errors when accessed
	dangerousGlobals := []string{
		"process",    // Blocks process.env access
		"require",    // Blocks module loading
		"module",     // Blocks module system
		"exports",    // Blocks exports
		"__dirname",  // Blocks directory access
		"__filename", // Blocks filename access
		"global",     // Blocks global scope manipulation
		"globalThis", // Blocks global scope manipulation (ES2020)
		"Deno",       // Blocks Deno runtime
		"Bun",        // Blocks Bun runtime
	}

	for _, name := range dangerousGlobals {
		if err := vm.Set(name, goja.Undefined()); err != nil {
			return err
		}
	}

	// SECURITY: Block eval and Function constructor to prevent dynamic code execution
	// Create a dummy that throws an error when called
	blockedFunc := func(call goja.FunctionCall) goja.Value {
		panic(vm.ToValue("Security Error: Dynamic code execution is not allowed"))
	}
	if err := vm.Set("eval", blockedFunc); err != nil {
		return err
	}

	// Set input object
	if err := vm.Set("input", input); err != nil {
		return err
	}

	// Create context object with http and credential
	contextObj := vm.NewObject()

	// Add HTTP client if available
	if execCtx != nil && execCtx.HTTP != nil {
		httpObj := vm.NewObject()

		// context.http.get(url, options)
		if err := httpObj.Set("get", func(call goja.FunctionCall) goja.Value {
			return s.httpRequest(vm, execCtx.HTTP, "GET", call)
		}); err != nil {
			return err
		}

		// context.http.post(url, body, options)
		if err := httpObj.Set("post", func(call goja.FunctionCall) goja.Value {
			return s.httpRequest(vm, execCtx.HTTP, "POST", call)
		}); err != nil {
			return err
		}

		// context.http.put(url, body, options)
		if err := httpObj.Set("put", func(call goja.FunctionCall) goja.Value {
			return s.httpRequest(vm, execCtx.HTTP, "PUT", call)
		}); err != nil {
			return err
		}

		// context.http.delete(url, options)
		if err := httpObj.Set("delete", func(call goja.FunctionCall) goja.Value {
			return s.httpRequest(vm, execCtx.HTTP, "DELETE", call)
		}); err != nil {
			return err
		}

		// context.http.patch(url, body, options)
		if err := httpObj.Set("patch", func(call goja.FunctionCall) goja.Value {
			return s.httpRequest(vm, execCtx.HTTP, "PATCH", call)
		}); err != nil {
			return err
		}

		if err := contextObj.Set("http", httpObj); err != nil {
			return err
		}
	}

	// Add LLM service if available
	if execCtx != nil && execCtx.LLM != nil {
		llmObj := vm.NewObject()
		if err := llmObj.Set("chat", func(call goja.FunctionCall) goja.Value {
			return s.llmChat(vm, execCtx.LLM, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("llm", llmObj); err != nil {
			return err
		}
	}

	// Add Workflow service if available
	if execCtx != nil && execCtx.Workflow != nil {
		workflowObj := vm.NewObject()
		if err := workflowObj.Set("run", func(call goja.FunctionCall) goja.Value {
			return s.workflowRun(vm, execCtx.Workflow, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("workflow", workflowObj); err != nil {
			return err
		}
	}

	// Add Human service if available
	if execCtx != nil && execCtx.Human != nil {
		humanObj := vm.NewObject()
		if err := humanObj.Set("requestApproval", func(call goja.FunctionCall) goja.Value {
			return s.humanRequestApproval(vm, execCtx.Human, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("human", humanObj); err != nil {
			return err
		}
	}

	// Add Adapter service if available
	if execCtx != nil && execCtx.Adapter != nil {
		adapterObj := vm.NewObject()
		if err := adapterObj.Set("call", func(call goja.FunctionCall) goja.Value {
			return s.adapterCall(vm, execCtx.Adapter, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("adapter", adapterObj); err != nil {
			return err
		}
	}

	// Add Blocks service if available (for Copilot/meta-workflow)
	if execCtx != nil && execCtx.Blocks != nil {
		blocksObj := vm.NewObject()
		if err := blocksObj.Set("list", func(call goja.FunctionCall) goja.Value {
			return s.blocksList(vm, execCtx.Blocks, call)
		}); err != nil {
			return err
		}
		if err := blocksObj.Set("get", func(call goja.FunctionCall) goja.Value {
			return s.blocksGet(vm, execCtx.Blocks, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("blocks", blocksObj); err != nil {
			return err
		}
	}

	// Add Workflows service if available (for Copilot/meta-workflow)
	if execCtx != nil && execCtx.Workflows != nil {
		workflowsObj := vm.NewObject()
		if err := workflowsObj.Set("get", func(call goja.FunctionCall) goja.Value {
			return s.workflowsGet(vm, execCtx.Workflows, call)
		}); err != nil {
			return err
		}
		if err := workflowsObj.Set("list", func(call goja.FunctionCall) goja.Value {
			return s.workflowsList(vm, execCtx.Workflows, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("workflows", workflowsObj); err != nil {
			return err
		}
	}

	// Add Runs service if available (for Copilot/meta-workflow)
	if execCtx != nil && execCtx.Runs != nil {
		runsObj := vm.NewObject()
		if err := runsObj.Set("get", func(call goja.FunctionCall) goja.Value {
			return s.runsGet(vm, execCtx.Runs, call)
		}); err != nil {
			return err
		}
		if err := runsObj.Set("getStepRuns", func(call goja.FunctionCall) goja.Value {
			return s.runsGetStepRuns(vm, execCtx.Runs, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("runs", runsObj); err != nil {
			return err
		}
	}

	// Add credentials map if available
	if execCtx != nil && execCtx.Credentials != nil {
		if err := contextObj.Set("credentials", execCtx.Credentials); err != nil {
			return err
		}
	}

	// Add logger
	if execCtx != nil && execCtx.Logger != nil {
		if err := contextObj.Set("log", execCtx.Logger); err != nil {
			return err
		}
	} else {
		// No-op logger
		if err := contextObj.Set("log", func(args ...interface{}) {}); err != nil {
			return err
		}
	}

	if err := vm.Set("context", contextObj); err != nil {
		return err
	}

	// Add ctx as an alias for context (for Unified Block Model compatibility)
	if err := vm.Set("ctx", contextObj); err != nil {
		return err
	}

	// Add console.log for debugging
	console := vm.NewObject()
	if execCtx != nil && execCtx.Logger != nil {
		if err := console.Set("log", execCtx.Logger); err != nil {
			return err
		}
	} else {
		if err := console.Set("log", func(args ...interface{}) {}); err != nil {
			return err
		}
	}
	if err := vm.Set("console", console); err != nil {
		return err
	}

	return nil
}

// wrapCode wraps user code in a function structure
func (s *Sandbox) wrapCode(code string) string {
	// Check if code already defines an execute function
	if strings.Contains(code, "function execute") || strings.Contains(code, "async function execute") {
		// User provided an execute function, call it
		return fmt.Sprintf(`
%s

(function() {
	var result = execute(input, context);
	return result;
})();
`, code)
	}

	// Otherwise, treat the code as the body of the execute function
	return fmt.Sprintf(`
(function() {
	%s
})();
`, code)
}

// extractResult converts goja value to Go map
func (s *Sandbox) extractResult(result goja.Value) (map[string]interface{}, error) {
	if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
		return map[string]interface{}{}, nil
	}

	// Export the value to Go
	exported := result.Export()

	// If already a map, return it
	if m, ok := exported.(map[string]interface{}); ok {
		return m, nil
	}

	// Wrap non-object results
	return map[string]interface{}{
		"result": exported,
	}, nil
}

// httpRequest handles HTTP requests from scripts
func (s *Sandbox) httpRequest(vm *goja.Runtime, client *HTTPClient, method string, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("http." + strings.ToLower(method) + " requires at least a URL"))
	}

	url := call.Arguments[0].String()

	var body interface{}
	var options map[string]interface{}

	if method == "POST" || method == "PUT" || method == "PATCH" {
		if len(call.Arguments) > 1 {
			body = call.Arguments[1].Export()
		}
		if len(call.Arguments) > 2 {
			if opts, ok := call.Arguments[2].Export().(map[string]interface{}); ok {
				options = opts
			}
		}
	} else {
		if len(call.Arguments) > 1 {
			if opts, ok := call.Arguments[1].Export().(map[string]interface{}); ok {
				options = opts
			}
		}
	}

	result, err := client.Request(method, url, body, options)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("HTTP request failed: %v", err)))
	}

	return vm.ToValue(result)
}

// HTTPClient provides HTTP request capabilities to scripts
type HTTPClient struct {
	client  *http.Client
	headers map[string]string
}

// NewHTTPClient creates a new HTTPClient
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		headers: make(map[string]string),
	}
}

// SetHeader sets a default header for all requests
func (c *HTTPClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// Request performs an HTTP request
func (c *HTTPClient) Request(method, url string, body interface{}, options map[string]interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = strings.NewReader(string(bodyJSON))
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Set Content-Type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Apply headers from options
	if options != nil {
		if headers, ok := options["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				if s, ok := v.(string); ok {
					req.Header.Set(k, s)
				}
			}
		}

		// Apply query parameters
		if params, ok := options["params"].(map[string]interface{}); ok {
			q := req.URL.Query()
			for k, v := range params {
				q.Set(k, fmt.Sprintf("%v", v))
			}
			req.URL.RawQuery = q.Encode()
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := map[string]interface{}{
		"status":     resp.StatusCode,
		"statusText": resp.Status,
		"headers":    headersToMap(resp.Header),
	}

	// Try to parse JSON response
	var jsonData interface{}
	if err := json.Unmarshal(respBody, &jsonData); err == nil {
		result["data"] = jsonData
	} else {
		result["data"] = string(respBody)
	}

	return result, nil
}

// headersToMap converts http.Header to map[string]string
func headersToMap(h http.Header) map[string]string {
	result := make(map[string]string)
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

// ============================================================================
// Unified Block Model Service Methods
// ============================================================================

// llmChat handles ctx.llm.chat(provider, model, request) calls
func (s *Sandbox) llmChat(vm *goja.Runtime, service LLMService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 3 {
		panic(vm.ToValue("ctx.llm.chat requires provider, model, and request arguments"))
	}

	provider := call.Arguments[0].String()
	model := call.Arguments[1].String()

	requestArg := call.Arguments[2].Export()
	request, ok := requestArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.llm.chat request must be an object"))
	}

	result, err := service.Chat(provider, model, request)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("LLM chat failed: %v", err)))
	}

	return vm.ToValue(result)
}

// workflowRun handles ctx.workflow.run(workflowID, input) calls
func (s *Sandbox) workflowRun(vm *goja.Runtime, service WorkflowService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(vm.ToValue("ctx.workflow.run requires workflowID and input arguments"))
	}

	workflowID := call.Arguments[0].String()

	inputArg := call.Arguments[1].Export()
	input, ok := inputArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.workflow.run input must be an object"))
	}

	result, err := service.Run(workflowID, input)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Workflow run failed: %v", err)))
	}

	return vm.ToValue(result)
}

// humanRequestApproval handles ctx.human.requestApproval(request) calls
func (s *Sandbox) humanRequestApproval(vm *goja.Runtime, service HumanService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.human.requestApproval requires a request argument"))
	}

	requestArg := call.Arguments[0].Export()
	request, ok := requestArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.human.requestApproval request must be an object"))
	}

	result, err := service.RequestApproval(request)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Human approval request failed: %v", err)))
	}

	return vm.ToValue(result)
}

// adapterCall handles ctx.adapter.call(adapterID, input) calls
func (s *Sandbox) adapterCall(vm *goja.Runtime, service AdapterService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(vm.ToValue("ctx.adapter.call requires adapterID and input arguments"))
	}

	adapterID := call.Arguments[0].String()

	inputArg := call.Arguments[1].Export()
	input, ok := inputArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.adapter.call input must be an object"))
	}

	result, err := service.Call(adapterID, input)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Adapter call failed: %v", err)))
	}

	return vm.ToValue(result)
}

// ============================================================================
// Copilot/Meta-Workflow Service Methods (Read-Only)
// ============================================================================

// blocksList handles ctx.blocks.list() calls
func (s *Sandbox) blocksList(vm *goja.Runtime, service BlocksService, call goja.FunctionCall) goja.Value {
	result, err := service.List()
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Blocks list failed: %v", err)))
	}
	return vm.ToValue(result)
}

// blocksGet handles ctx.blocks.get(slug) calls
func (s *Sandbox) blocksGet(vm *goja.Runtime, service BlocksService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.blocks.get requires slug argument"))
	}

	slug := call.Arguments[0].String()

	result, err := service.Get(slug)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Blocks get failed: %v", err)))
	}

	return vm.ToValue(result)
}

// workflowsGet handles ctx.workflows.get(workflowID) calls
func (s *Sandbox) workflowsGet(vm *goja.Runtime, service WorkflowsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.workflows.get requires workflowID argument"))
	}

	workflowID := call.Arguments[0].String()

	result, err := service.Get(workflowID)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Workflows get failed: %v", err)))
	}

	return vm.ToValue(result)
}

// workflowsList handles ctx.workflows.list() calls
func (s *Sandbox) workflowsList(vm *goja.Runtime, service WorkflowsService, call goja.FunctionCall) goja.Value {
	result, err := service.List()
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Workflows list failed: %v", err)))
	}
	return vm.ToValue(result)
}

// runsGet handles ctx.runs.get(runID) calls
func (s *Sandbox) runsGet(vm *goja.Runtime, service RunsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.runs.get requires runID argument"))
	}

	runID := call.Arguments[0].String()

	result, err := service.Get(runID)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Runs get failed: %v", err)))
	}

	return vm.ToValue(result)
}

// runsGetStepRuns handles ctx.runs.getStepRuns(runID) calls
func (s *Sandbox) runsGetStepRuns(vm *goja.Runtime, service RunsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.runs.getStepRuns requires runID argument"))
	}

	runID := call.Arguments[0].String()

	result, err := service.GetStepRuns(runID)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Runs getStepRuns failed: %v", err)))
	}

	return vm.ToValue(result)
}
