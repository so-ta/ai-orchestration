package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/souta/ai-orchestration/internal/domain"
)

// Errors
var (
	ErrTimeout     = errors.New("script execution timed out")
	ErrMemoryLimit = errors.New("script exceeded memory limit")
	ErrInvalidCode = errors.New("invalid or empty code")
)

// sanitizeError removes internal system information from error messages
// to prevent leaking implementation details to users.
func sanitizeError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Remove Go stack traces (lines starting with "at github.com/...")
	// These come from goja runtime and expose internal package paths
	if idx := strings.Index(errStr, " at github.com/"); idx != -1 {
		errStr = strings.TrimSpace(errStr[:idx])
	}

	// Remove generic "(native)" suffixes
	errStr = strings.TrimSuffix(errStr, " (native)")

	// Remove any remaining file paths that look like Go imports
	// Pattern: github.com/anything/path
	lines := strings.Split(errStr, "\n")
	var cleanLines []string
	for _, line := range lines {
		// Skip lines that are purely stack traces
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "at ") && strings.Contains(trimmed, "github.com/") {
			continue
		}
		if strings.HasPrefix(trimmed, "github.com/") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	return errors.New(strings.TrimSpace(strings.Join(cleanLines, "\n")))
}

// Config holds sandbox configuration
type Config struct {
	Timeout     time.Duration
	MemoryLimit int64 // in bytes - NOTE: Goja does not support native memory limits.
	// This value is used for documentation and future monitoring integration.
	// Memory safety is achieved through: timeout limits, blocked dangerous APIs,
	// and Go's garbage collector. For strict memory enforcement, consider
	// running sandboxed code in separate processes or containers.
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

// WorkflowService provides subflow execution capability.
//
// Note: Methods do not take context.Context as a parameter because this interface
// is called from JavaScript code via goja, which cannot pass Go contexts.
// The context is captured in closures at service creation time (see WorkflowServiceImpl)
// and is properly used for execution, cancellation, timeout, and tenant isolation.
type WorkflowService interface {
	// Run executes a subflow and returns its output
	Run(workflowID string, input map[string]interface{}) (map[string]interface{}, error)
	// ExecuteStep executes a step within the current workflow by name and returns its output.
	// This enables agent blocks to call other steps as tools.
	// Context for execution is captured when the service is created via NewWorkflowServiceWithExecutor.
	ExecuteStep(stepName string, input map[string]interface{}) (map[string]interface{}, error)
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
	// GetWithSchema retrieves a block with full config schema (for AI agents)
	GetWithSchema(slug string) (map[string]interface{}, error)
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

// BuilderSessionsService provides builder session access to scripts
type BuilderSessionsService interface {
	// Get retrieves a builder session by ID
	Get(sessionID string) (map[string]interface{}, error)
	// Update updates a builder session
	Update(sessionID string, updates map[string]interface{}) error
	// AddMessage adds a message to a builder session
	AddMessage(sessionID string, message map[string]interface{}) error
}

// ProjectsService provides project management for builder workflows
type ProjectsService interface {
	// Get retrieves a project by ID
	Get(projectID string) (map[string]interface{}, error)
	// Create creates a new project
	Create(data map[string]interface{}) (map[string]interface{}, error)
	// Update updates a project
	Update(projectID string, updates map[string]interface{}) error
	// IncrementVersion increments the project version
	IncrementVersion(projectID string) error
}

// StepsService provides step management for builder workflows
type StepsService interface {
	// ListByProject retrieves all steps for a project
	ListByProject(projectID string) ([]map[string]interface{}, error)
	// Create creates a new step
	Create(data map[string]interface{}) (map[string]interface{}, error)
	// Update updates a step
	Update(stepID string, updates map[string]interface{}) error
	// Delete soft-deletes a step
	Delete(stepID string) error
}

// EdgesService provides edge management for builder workflows
type EdgesService interface {
	// ListByProject retrieves all edges for a project
	ListByProject(projectID string) ([]map[string]interface{}, error)
	// Create creates a new edge
	Create(data map[string]interface{}) (map[string]interface{}, error)
	// Delete soft-deletes an edge
	Delete(edgeID string) error
}

// ExecutionContext provides runtime context to scripts
type ExecutionContext struct {
	HTTP *HTTPClient
	// Unified Block Model services
	LLM      LLMService
	Workflow WorkflowService
	Human    HumanService
	Adapter  AdapterService
	// RAG services (with tenant isolation)
	Embedding EmbeddingService
	Vector    VectorService
	// Copilot/meta-workflow services (read-only data access)
	Blocks    BlocksService
	Workflows WorkflowsService
	Runs      RunsService
	// Builder services (for AI workflow builder)
	BuilderSessions BuilderSessionsService
	Projects        ProjectsService
	Steps           StepsService
	Edges           EdgesService
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
		return nil, fmt.Errorf("script compilation error: %v", sanitizeError(err))
	}

	result, err := vm.RunProgram(program)
	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, ErrTimeout
		}
		return nil, fmt.Errorf("script execution error: %v", sanitizeError(err))
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
	// SECURITY: Block Function constructor
	if err := vm.Set("Function", blockedFunc); err != nil {
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
		if err := workflowObj.Set("executeStep", func(call goja.FunctionCall) goja.Value {
			return s.workflowExecuteStep(vm, execCtx.Workflow, call)
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

	// Add Embedding service if available (RAG)
	if execCtx != nil && execCtx.Embedding != nil {
		embeddingObj := vm.NewObject()
		if err := embeddingObj.Set("embed", func(call goja.FunctionCall) goja.Value {
			return s.embeddingEmbed(vm, execCtx.Embedding, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("embedding", embeddingObj); err != nil {
			return err
		}
	}

	// Add Vector service if available (RAG with tenant isolation)
	if execCtx != nil && execCtx.Vector != nil {
		vectorObj := vm.NewObject()
		if err := vectorObj.Set("upsert", func(call goja.FunctionCall) goja.Value {
			return s.vectorUpsert(vm, execCtx.Vector, call)
		}); err != nil {
			return err
		}
		if err := vectorObj.Set("query", func(call goja.FunctionCall) goja.Value {
			return s.vectorQuery(vm, execCtx.Vector, call)
		}); err != nil {
			return err
		}
		if err := vectorObj.Set("delete", func(call goja.FunctionCall) goja.Value {
			return s.vectorDelete(vm, execCtx.Vector, call)
		}); err != nil {
			return err
		}
		if err := vectorObj.Set("listCollections", func(call goja.FunctionCall) goja.Value {
			return s.vectorListCollections(vm, execCtx.Vector, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("vector", vectorObj); err != nil {
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
		if err := blocksObj.Set("getWithSchema", func(call goja.FunctionCall) goja.Value {
			return s.blocksGetWithSchema(vm, execCtx.Blocks, call)
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

	// Add BuilderSessions service if available (for AI workflow builder)
	if execCtx != nil && execCtx.BuilderSessions != nil {
		builderSessionsObj := vm.NewObject()
		if err := builderSessionsObj.Set("get", func(call goja.FunctionCall) goja.Value {
			return s.builderSessionsGet(vm, execCtx.BuilderSessions, call)
		}); err != nil {
			return err
		}
		if err := builderSessionsObj.Set("update", func(call goja.FunctionCall) goja.Value {
			return s.builderSessionsUpdate(vm, execCtx.BuilderSessions, call)
		}); err != nil {
			return err
		}
		if err := builderSessionsObj.Set("addMessage", func(call goja.FunctionCall) goja.Value {
			return s.builderSessionsAddMessage(vm, execCtx.BuilderSessions, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("builderSessions", builderSessionsObj); err != nil {
			return err
		}
	}

	// Add Projects service if available (for AI workflow builder)
	if execCtx != nil && execCtx.Projects != nil {
		projectsObj := vm.NewObject()
		if err := projectsObj.Set("get", func(call goja.FunctionCall) goja.Value {
			return s.projectsGet(vm, execCtx.Projects, call)
		}); err != nil {
			return err
		}
		if err := projectsObj.Set("create", func(call goja.FunctionCall) goja.Value {
			return s.projectsCreate(vm, execCtx.Projects, call)
		}); err != nil {
			return err
		}
		if err := projectsObj.Set("update", func(call goja.FunctionCall) goja.Value {
			return s.projectsUpdate(vm, execCtx.Projects, call)
		}); err != nil {
			return err
		}
		if err := projectsObj.Set("incrementVersion", func(call goja.FunctionCall) goja.Value {
			return s.projectsIncrementVersion(vm, execCtx.Projects, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("projects", projectsObj); err != nil {
			return err
		}
	}

	// Add Steps service if available (for AI workflow builder)
	if execCtx != nil && execCtx.Steps != nil {
		stepsObj := vm.NewObject()
		if err := stepsObj.Set("listByProject", func(call goja.FunctionCall) goja.Value {
			return s.stepsListByProject(vm, execCtx.Steps, call)
		}); err != nil {
			return err
		}
		if err := stepsObj.Set("create", func(call goja.FunctionCall) goja.Value {
			return s.stepsCreate(vm, execCtx.Steps, call)
		}); err != nil {
			return err
		}
		if err := stepsObj.Set("update", func(call goja.FunctionCall) goja.Value {
			return s.stepsUpdate(vm, execCtx.Steps, call)
		}); err != nil {
			return err
		}
		if err := stepsObj.Set("delete", func(call goja.FunctionCall) goja.Value {
			return s.stepsDelete(vm, execCtx.Steps, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("steps", stepsObj); err != nil {
			return err
		}
	}

	// Add Edges service if available (for AI workflow builder)
	if execCtx != nil && execCtx.Edges != nil {
		edgesObj := vm.NewObject()
		if err := edgesObj.Set("listByProject", func(call goja.FunctionCall) goja.Value {
			return s.edgesListByProject(vm, execCtx.Edges, call)
		}); err != nil {
			return err
		}
		if err := edgesObj.Set("create", func(call goja.FunctionCall) goja.Value {
			return s.edgesCreate(vm, execCtx.Edges, call)
		}); err != nil {
			return err
		}
		if err := edgesObj.Set("delete", func(call goja.FunctionCall) goja.Value {
			return s.edgesDelete(vm, execCtx.Edges, call)
		}); err != nil {
			return err
		}
		if err := contextObj.Set("edges", edgesObj); err != nil {
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

	result, err := client.Request(client.Context(), method, url, body, options)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("HTTP request failed: %v", err)))
	}

	return vm.ToValue(result)
}

// HTTPClient provides HTTP request capabilities to scripts
type HTTPClient struct {
	client  *http.Client
	headers map[string]string
	mu      sync.RWMutex
	ctx     context.Context // Context for cancellation/timeout propagation
}

// NewHTTPClient creates a new HTTPClient
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		headers: make(map[string]string),
		ctx:     context.Background(),
	}
}

// NewHTTPClientWithContext creates a new HTTPClient with a context for cancellation support
func NewHTTPClientWithContext(ctx context.Context, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		headers: make(map[string]string),
		ctx:     ctx,
	}
}

// Context returns the context associated with this client
func (c *HTTPClient) Context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// SetHeader sets a default header for all requests
func (c *HTTPClient) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[key] = value
}

// getHeaders returns a copy of the default headers (thread-safe)
func (c *HTTPClient) getHeaders() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	headers := make(map[string]string, len(c.headers))
	for k, v := range c.headers {
		headers[k] = v
	}
	return headers
}

// Request performs an HTTP request with context support for cancellation and timeout
func (c *HTTPClient) Request(ctx context.Context, method, url string, body interface{}, options map[string]interface{}) (map[string]interface{}, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyJSON)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers (thread-safe read via copy)
	for k, v := range c.getHeaders() {
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
		// Sanitize error to prevent leaking internal implementation details
		sanitized := sanitizeError(err)
		panic(vm.ToValue(fmt.Sprintf("LLM chat failed: %v", sanitized)))
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
		// Sanitize error to prevent leaking internal implementation details
		sanitized := sanitizeError(err)
		panic(vm.ToValue(fmt.Sprintf("Workflow run failed: %v", sanitized)))
	}

	return vm.ToValue(result)
}

// workflowExecuteStep handles ctx.workflow.executeStep(stepName, input) calls
// This enables agent blocks to call other steps as tools within the current workflow
func (s *Sandbox) workflowExecuteStep(vm *goja.Runtime, service WorkflowService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(vm.ToValue("ctx.workflow.executeStep requires stepName and input arguments"))
	}

	stepName := call.Arguments[0].String()

	inputArg := call.Arguments[1].Export()
	input, ok := inputArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.workflow.executeStep input must be an object"))
	}

	result, err := service.ExecuteStep(stepName, input)
	if err != nil {
		// Sanitize error to prevent leaking internal implementation details
		sanitized := sanitizeError(err)
		panic(vm.ToValue(fmt.Sprintf("Step execution failed: %v", sanitized)))
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

// blocksGetWithSchema handles ctx.blocks.getWithSchema(slug) calls
// Returns full block information including config_schema for AI agents
func (s *Sandbox) blocksGetWithSchema(vm *goja.Runtime, service BlocksService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.blocks.getWithSchema requires slug argument"))
	}

	slug := call.Arguments[0].String()

	result, err := service.GetWithSchema(slug)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Blocks getWithSchema failed: %v", err)))
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

// ============================================================================
// RAG Service Methods (Embedding & Vector with Tenant Isolation)
// ============================================================================

// embeddingEmbed handles ctx.embedding.embed(provider, model, texts) calls
func (s *Sandbox) embeddingEmbed(vm *goja.Runtime, service EmbeddingService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 3 {
		panic(vm.ToValue("ctx.embedding.embed requires provider, model, and texts arguments"))
	}

	provider := call.Arguments[0].String()
	model := call.Arguments[1].String()

	// Handle both single string and array of strings
	textsArg := call.Arguments[2].Export()
	var texts []string

	switch v := textsArg.(type) {
	case string:
		texts = []string{v}
	case []interface{}:
		texts = make([]string, len(v))
		for i, t := range v {
			texts[i] = fmt.Sprintf("%v", t)
		}
	default:
		panic(vm.ToValue("ctx.embedding.embed texts must be a string or array of strings"))
	}

	result, err := service.Embed(provider, model, texts)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Embedding failed: %v", err)))
	}

	// Convert to JS-compatible format
	vectors := make([]interface{}, len(result.Vectors))
	for i, v := range result.Vectors {
		floats := make([]interface{}, len(v))
		for j, f := range v {
			floats[j] = float64(f)
		}
		vectors[i] = floats
	}

	return vm.ToValue(map[string]interface{}{
		"vectors":   vectors,
		"model":     result.Model,
		"dimension": result.Dimension,
		"usage": map[string]interface{}{
			"total_tokens": result.Usage.TotalTokens,
		},
	})
}

// vectorUpsert handles ctx.vector.upsert(collection, documents, options) calls
func (s *Sandbox) vectorUpsert(vm *goja.Runtime, service VectorService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(vm.ToValue("ctx.vector.upsert requires collection and documents arguments"))
	}

	collection := call.Arguments[0].String()

	docsArg := call.Arguments[1].Export()
	docsArray, ok := docsArg.([]interface{})
	if !ok {
		panic(vm.ToValue("ctx.vector.upsert documents must be an array"))
	}

	documents := make([]VectorDocument, len(docsArray))
	for i, d := range docsArray {
		docMap, ok := d.(map[string]interface{})
		if !ok {
			panic(vm.ToValue(fmt.Sprintf("document at index %d must be an object", i)))
		}

		doc := VectorDocument{}
		if id, ok := docMap["id"].(string); ok {
			doc.ID = id
		}
		if content, ok := docMap["content"].(string); ok {
			doc.Content = content
		}
		if metadata, ok := docMap["metadata"].(map[string]interface{}); ok {
			doc.Metadata = metadata
		}
		if vector, ok := docMap["vector"].([]interface{}); ok {
			doc.Vector = make([]float32, len(vector))
			for j, v := range vector {
				if f, ok := v.(float64); ok {
					doc.Vector[j] = float32(f)
				}
			}
		}

		documents[i] = doc
	}

	// Parse options
	var opts *UpsertOptions
	if len(call.Arguments) > 2 {
		if optsArg, ok := call.Arguments[2].Export().(map[string]interface{}); ok {
			opts = &UpsertOptions{}
			if p, ok := optsArg["embedding_provider"].(string); ok {
				opts.EmbeddingProvider = p
			}
			if m, ok := optsArg["embedding_model"].(string); ok {
				opts.EmbeddingModel = m
			}
		}
	}

	result, err := service.Upsert(collection, documents, opts)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Vector upsert failed: %v", err)))
	}

	return vm.ToValue(map[string]interface{}{
		"upserted_count": result.UpsertedCount,
		"ids":            result.IDs,
	})
}

// vectorQuery handles ctx.vector.query(collection, vector, options) calls
func (s *Sandbox) vectorQuery(vm *goja.Runtime, service VectorService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(vm.ToValue("ctx.vector.query requires collection and vector arguments"))
	}

	collection := call.Arguments[0].String()

	vectorArg := call.Arguments[1].Export()
	vectorArray, ok := vectorArg.([]interface{})
	if !ok {
		panic(vm.ToValue("ctx.vector.query vector must be an array of numbers"))
	}

	vector := make([]float32, len(vectorArray))
	for i, v := range vectorArray {
		if f, ok := v.(float64); ok {
			vector[i] = float32(f)
		}
	}

	// Parse options
	opts := &QueryOptions{
		TopK:           5,
		IncludeContent: true,
	}
	if len(call.Arguments) > 2 {
		if optsArg, ok := call.Arguments[2].Export().(map[string]interface{}); ok {
			if topK, ok := optsArg["top_k"].(float64); ok {
				opts.TopK = int(topK)
			}
			if threshold, ok := optsArg["threshold"].(float64); ok {
				opts.Threshold = threshold
			}
			if filter, ok := optsArg["filter"].(map[string]interface{}); ok {
				opts.Filter = filter
			}
			if includeContent, ok := optsArg["include_content"].(bool); ok {
				opts.IncludeContent = includeContent
			}
		}
	}

	result, err := service.Query(collection, vector, opts)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Vector query failed: %v", err)))
	}

	// Convert to JS-compatible format
	matches := make([]interface{}, len(result.Matches))
	for i, m := range result.Matches {
		match := map[string]interface{}{
			"id":    m.ID,
			"score": m.Score,
		}
		if m.Content != "" {
			match["content"] = m.Content
		}
		if m.Metadata != nil {
			match["metadata"] = m.Metadata
		}
		matches[i] = match
	}

	return vm.ToValue(map[string]interface{}{
		"matches": matches,
	})
}

// vectorDelete handles ctx.vector.delete(collection, ids) calls
func (s *Sandbox) vectorDelete(vm *goja.Runtime, service VectorService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(vm.ToValue("ctx.vector.delete requires collection and ids arguments"))
	}

	collection := call.Arguments[0].String()

	idsArg := call.Arguments[1].Export()
	idsArray, ok := idsArg.([]interface{})
	if !ok {
		panic(vm.ToValue("ctx.vector.delete ids must be an array of strings"))
	}

	ids := make([]string, len(idsArray))
	for i, id := range idsArray {
		ids[i] = fmt.Sprintf("%v", id)
	}

	result, err := service.Delete(collection, ids)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Vector delete failed: %v", err)))
	}

	return vm.ToValue(map[string]interface{}{
		"deleted_count": result.DeletedCount,
	})
}

// vectorListCollections handles ctx.vector.listCollections() calls
func (s *Sandbox) vectorListCollections(vm *goja.Runtime, service VectorService, call goja.FunctionCall) goja.Value {
	result, err := service.ListCollections()
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("List collections failed: %v", err)))
	}

	// Convert to JS-compatible format
	collections := make([]interface{}, len(result))
	for i, c := range result {
		collections[i] = map[string]interface{}{
			"name":           c.Name,
			"document_count": c.DocumentCount,
			"dimension":      c.Dimension,
			"created_at":     c.CreatedAt,
		}
	}

	return vm.ToValue(collections)
}

// ============================================================================
// Builder Service Methods (for AI workflow builder)
// ============================================================================

// builderSessionsGet handles ctx.builderSessions.get(sessionID) calls
func (s *Sandbox) builderSessionsGet(vm *goja.Runtime, service BuilderSessionsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.builderSessions.get requires sessionID argument"))
	}

	sessionID := call.Arguments[0].String()

	result, err := service.Get(sessionID)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("BuilderSessions get failed: %v", err)))
	}

	return vm.ToValue(result)
}

// builderSessionsUpdate handles ctx.builderSessions.update(sessionID, updates) calls
func (s *Sandbox) builderSessionsUpdate(vm *goja.Runtime, service BuilderSessionsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(vm.ToValue("ctx.builderSessions.update requires sessionID and updates arguments"))
	}

	sessionID := call.Arguments[0].String()

	updatesArg := call.Arguments[1].Export()
	updates, ok := updatesArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.builderSessions.update updates must be an object"))
	}

	err := service.Update(sessionID, updates)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("BuilderSessions update failed: %v", err)))
	}

	return vm.ToValue(map[string]interface{}{"success": true})
}

// builderSessionsAddMessage handles ctx.builderSessions.addMessage(sessionID, message) calls
func (s *Sandbox) builderSessionsAddMessage(vm *goja.Runtime, service BuilderSessionsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(vm.ToValue("ctx.builderSessions.addMessage requires sessionID and message arguments"))
	}

	sessionID := call.Arguments[0].String()

	messageArg := call.Arguments[1].Export()
	message, ok := messageArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.builderSessions.addMessage message must be an object"))
	}

	err := service.AddMessage(sessionID, message)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("BuilderSessions addMessage failed: %v", err)))
	}

	return vm.ToValue(map[string]interface{}{"success": true})
}

// projectsGet handles ctx.projects.get(projectID) calls
func (s *Sandbox) projectsGet(vm *goja.Runtime, service ProjectsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.projects.get requires projectID argument"))
	}

	projectID := call.Arguments[0].String()

	result, err := service.Get(projectID)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Projects get failed: %v", err)))
	}

	return vm.ToValue(result)
}

// projectsCreate handles ctx.projects.create(data) calls
func (s *Sandbox) projectsCreate(vm *goja.Runtime, service ProjectsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.projects.create requires data argument"))
	}

	dataArg := call.Arguments[0].Export()
	data, ok := dataArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.projects.create data must be an object"))
	}

	result, err := service.Create(data)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Projects create failed: %v", err)))
	}

	return vm.ToValue(result)
}

// projectsUpdate handles ctx.projects.update(projectID, updates) calls
func (s *Sandbox) projectsUpdate(vm *goja.Runtime, service ProjectsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(vm.ToValue("ctx.projects.update requires projectID and updates arguments"))
	}

	projectID := call.Arguments[0].String()

	updatesArg := call.Arguments[1].Export()
	updates, ok := updatesArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.projects.update updates must be an object"))
	}

	err := service.Update(projectID, updates)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Projects update failed: %v", err)))
	}

	return vm.ToValue(map[string]interface{}{"success": true})
}

// projectsIncrementVersion handles ctx.projects.incrementVersion(projectID) calls
func (s *Sandbox) projectsIncrementVersion(vm *goja.Runtime, service ProjectsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.projects.incrementVersion requires projectID argument"))
	}

	projectID := call.Arguments[0].String()

	err := service.IncrementVersion(projectID)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Projects incrementVersion failed: %v", err)))
	}

	return vm.ToValue(map[string]interface{}{"success": true})
}

// stepsListByProject handles ctx.steps.listByProject(projectID) calls
func (s *Sandbox) stepsListByProject(vm *goja.Runtime, service StepsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.steps.listByProject requires projectID argument"))
	}

	projectID := call.Arguments[0].String()

	result, err := service.ListByProject(projectID)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Steps listByProject failed: %v", err)))
	}

	return vm.ToValue(result)
}

// stepsCreate handles ctx.steps.create(data) calls
func (s *Sandbox) stepsCreate(vm *goja.Runtime, service StepsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.steps.create requires data argument"))
	}

	dataArg := call.Arguments[0].Export()
	data, ok := dataArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.steps.create data must be an object"))
	}

	result, err := service.Create(data)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Steps create failed: %v", err)))
	}

	return vm.ToValue(result)
}

// stepsUpdate handles ctx.steps.update(stepID, updates) calls
func (s *Sandbox) stepsUpdate(vm *goja.Runtime, service StepsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(vm.ToValue("ctx.steps.update requires stepID and updates arguments"))
	}

	stepID := call.Arguments[0].String()

	updatesArg := call.Arguments[1].Export()
	updates, ok := updatesArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.steps.update updates must be an object"))
	}

	err := service.Update(stepID, updates)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Steps update failed: %v", err)))
	}

	return vm.ToValue(map[string]interface{}{"success": true})
}

// stepsDelete handles ctx.steps.delete(stepID) calls
func (s *Sandbox) stepsDelete(vm *goja.Runtime, service StepsService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.steps.delete requires stepID argument"))
	}

	stepID := call.Arguments[0].String()

	err := service.Delete(stepID)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Steps delete failed: %v", err)))
	}

	return vm.ToValue(map[string]interface{}{"success": true})
}

// edgesListByProject handles ctx.edges.listByProject(projectID) calls
func (s *Sandbox) edgesListByProject(vm *goja.Runtime, service EdgesService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.edges.listByProject requires projectID argument"))
	}

	projectID := call.Arguments[0].String()

	result, err := service.ListByProject(projectID)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Edges listByProject failed: %v", err)))
	}

	return vm.ToValue(result)
}

// edgesCreate handles ctx.edges.create(data) calls
func (s *Sandbox) edgesCreate(vm *goja.Runtime, service EdgesService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.edges.create requires data argument"))
	}

	dataArg := call.Arguments[0].Export()
	data, ok := dataArg.(map[string]interface{})
	if !ok {
		panic(vm.ToValue("ctx.edges.create data must be an object"))
	}

	result, err := service.Create(data)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Edges create failed: %v", err)))
	}

	return vm.ToValue(result)
}

// edgesDelete handles ctx.edges.delete(edgeID) calls
func (s *Sandbox) edgesDelete(vm *goja.Runtime, service EdgesService, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.ToValue("ctx.edges.delete requires edgeID argument"))
	}

	edgeID := call.Arguments[0].String()

	err := service.Delete(edgeID)
	if err != nil {
		panic(vm.ToValue(fmt.Sprintf("Edges delete failed: %v", err)))
	}

	return vm.ToValue(map[string]interface{}{"success": true})
}

// ============================================================================
// Declarative Request/Response Processing
// ============================================================================

// templateVarRegex matches template variables like {{field}}, {{input.field}}, {{secret.KEY}}
var templateVarRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// DeclarativeContext holds context for declarative request/response processing
type DeclarativeContext struct {
	Config      map[string]interface{} // Block configuration values
	Input       map[string]interface{} // Input data from previous step
	Credentials map[string]interface{} // Resolved credentials
}

// ExpandTemplate expands template variables in a string
// Supports: {{field}} (config), {{input.field}} (input data), {{secret.KEY}} (credentials)
func ExpandTemplate(template string, ctx *DeclarativeContext) string {
	return templateVarRegex.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable path without {{ }}
		path := strings.TrimPrefix(strings.TrimSuffix(match, "}}"), "{{")
		path = strings.TrimSpace(path)

		// Handle different prefixes
		if strings.HasPrefix(path, "input.") {
			fieldPath := strings.TrimPrefix(path, "input.")
			return getNestedValue(ctx.Input, fieldPath)
		}
		if strings.HasPrefix(path, "secret.") {
			credName := strings.TrimPrefix(path, "secret.")
			return getNestedValue(ctx.Credentials, credName)
		}

		// Default: config value
		return getNestedValue(ctx.Config, path)
	})
}

// ExpandTemplateForURLPath expands template variables and URL-encodes values for use in URL paths
// Automatically detects already-encoded values to prevent double-encoding
func ExpandTemplateForURLPath(template string, ctx *DeclarativeContext) string {
	return templateVarRegex.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable path without {{ }}
		path := strings.TrimPrefix(strings.TrimSuffix(match, "}}"), "{{")
		path = strings.TrimSpace(path)

		var value string
		// Handle different prefixes
		if strings.HasPrefix(path, "input.") {
			fieldPath := strings.TrimPrefix(path, "input.")
			value = getNestedValue(ctx.Input, fieldPath)
		} else if strings.HasPrefix(path, "secret.") {
			credName := strings.TrimPrefix(path, "secret.")
			value = getNestedValue(ctx.Credentials, credName)
		} else {
			// Default: config value
			value = getNestedValue(ctx.Config, path)
		}

		// URL encode the value, but detect already-encoded values to prevent double-encoding
		return urlEncodePathSegment(value)
	})
}

// urlEncodePathSegment URL-encodes a value for use in a URL path segment
// Detects already-encoded values to prevent double-encoding
func urlEncodePathSegment(value string) string {
	if value == "" {
		return value
	}

	// Check if value appears to be already URL-encoded
	// Look for percent-encoded sequences like %20, %2F, etc.
	if isAlreadyURLEncoded(value) {
		return value
	}

	// Use PathEscape for URL path segments (encodes spaces as %20, not +)
	return url.PathEscape(value)
}

// isAlreadyURLEncoded checks if a string appears to be already URL-encoded
func isAlreadyURLEncoded(s string) bool {
	// If string contains % followed by two hex digits, it's likely already encoded
	for i := 0; i < len(s)-2; i++ {
		if s[i] == '%' {
			// Check if next two characters are hex digits
			if isHexDigit(s[i+1]) && isHexDigit(s[i+2]) {
				return true
			}
		}
	}
	return false
}

// isHexDigit checks if a byte is a valid hexadecimal digit
func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'F') || (b >= 'a' && b <= 'f')
}

// ExpandTemplateValue recursively expands templates in any value type
// Supports object format with omit_empty option:
//
//	field:
//	  value: "{{template}}"
//	  omit_empty: true
func ExpandTemplateValue(value interface{}, ctx *DeclarativeContext) interface{} {
	switch v := value.(type) {
	case string:
		// Check if this is a pure template variable reference (e.g., "{{input.filter}}")
		// If so, return the actual value instead of a string representation
		trimmed := strings.TrimSpace(v)
		if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") {
			// Check if there's only one template variable
			if strings.Count(trimmed, "{{") == 1 {
				path := strings.TrimPrefix(strings.TrimSuffix(trimmed, "}}"), "{{")
				path = strings.TrimSpace(path)

				// Get the actual value (not string representation)
				actualValue := getNestedValueAny(ctx.Config, path)
				if strings.HasPrefix(path, "input.") {
					fieldPath := strings.TrimPrefix(path, "input.")
					actualValue = getNestedValueAny(ctx.Input, fieldPath)
				} else if strings.HasPrefix(path, "secret.") {
					credName := strings.TrimPrefix(path, "secret.")
					actualValue = getNestedValueAny(ctx.Credentials, credName)
				}

				// Return actual value (preserves arrays, maps, etc.)
				if actualValue != nil {
					return actualValue
				}
				// If nil, return empty string (for template compatibility)
				return ""
			}
		}
		// For mixed strings with templates, use string expansion
		return ExpandTemplate(v, ctx)
	case map[string]interface{}:
		// Check if this is a field with omit_empty option
		if valueField, hasValue := v["value"]; hasValue {
			// This is an object format field
			expanded := ExpandTemplateValue(valueField, ctx)

			// Check omit_empty option
			if omitEmpty, ok := v["omit_empty"].(bool); ok && omitEmpty {
				if isEmptyValue(expanded) {
					// Return special marker to indicate this field should be omitted
					return omitEmptyMarker{}
				}
			}
			return expanded
		}

		// Regular map - recursively expand and filter out omit_empty markers
		result := make(map[string]interface{})
		for key, val := range v {
			expanded := ExpandTemplateValue(val, ctx)
			// Skip fields marked for omission
			if _, isMarker := expanded.(omitEmptyMarker); !isMarker {
				result[key] = expanded
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, 0, len(v))
		for _, val := range v {
			expanded := ExpandTemplateValue(val, ctx)
			// Skip items marked for omission
			if _, isMarker := expanded.(omitEmptyMarker); !isMarker {
				result = append(result, expanded)
			}
		}
		return result
	default:
		return v
	}
}

// omitEmptyMarker is used to mark fields that should be omitted from output
type omitEmptyMarker struct{}

// isEmptyValue checks if a value should be considered "empty" for omit_empty
func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	case bool:
		return false // booleans are never "empty"
	case float64:
		return false // numbers are never "empty"
	case int:
		return false // numbers are never "empty"
	default:
		return false
	}
}

// getNestedValue retrieves a nested value from a map using dot notation
func getNestedValue(data map[string]interface{}, path string) string {
	if data == nil {
		return ""
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			if val, ok := v[part]; ok {
				current = val
			} else {
				return ""
			}
		default:
			return ""
		}
	}

	// Convert final value to string
	switch v := current.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// BuildDeclarativeRequest builds an HTTP request from declarative RequestConfig
// The goCtx parameter is used for request cancellation and timeout propagation
func (s *Sandbox) BuildDeclarativeRequest(goCtx context.Context, reqConfig *domain.RequestConfig, declCtx *DeclarativeContext) (*http.Request, error) {
	if reqConfig == nil {
		return nil, errors.New("request config is nil")
	}
	if goCtx == nil {
		goCtx = context.Background()
	}

	// Expand URL template with automatic URL-encoding for path variables
	expandedURL := ExpandTemplateForURLPath(reqConfig.URL, declCtx)
	if expandedURL == "" {
		return nil, errors.New("URL is required in request config")
	}

	// Determine method
	method := reqConfig.Method
	if method == "" {
		method = "GET"
	}

	// Build request body
	var bodyReader io.Reader
	if reqConfig.Body != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		expandedBody := ExpandTemplateValue(reqConfig.Body, declCtx)
		bodyJSON, err := json.Marshal(expandedBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyJSON)
	}

	// Create request with context for cancellation/timeout support
	req, err := http.NewRequestWithContext(goCtx, method, expandedURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Content-Type for requests with body
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Apply headers from config
	for key, value := range reqConfig.Headers {
		req.Header.Set(key, ExpandTemplate(value, declCtx))
	}

	// Apply query parameters
	if len(reqConfig.QueryParams) > 0 {
		q := req.URL.Query()
		for key, value := range reqConfig.QueryParams {
			q.Set(key, ExpandTemplate(value, declCtx))
		}
		req.URL.RawQuery = q.Encode()
	}

	return req, nil
}

// ProcessDeclarativeResponse processes an HTTP response using declarative ResponseConfig
func (s *Sandbox) ProcessDeclarativeResponse(respConfig *domain.ResponseConfig, resp *http.Response, respBody []byte) (map[string]interface{}, error) {
	// Parse response body as JSON
	var bodyData interface{}
	if err := json.Unmarshal(respBody, &bodyData); err != nil {
		// If not JSON, use as string
		bodyData = string(respBody)
	}

	// Check status code
	statusOK := false
	if respConfig != nil && len(respConfig.SuccessStatus) > 0 {
		for _, code := range respConfig.SuccessStatus {
			if resp.StatusCode == code {
				statusOK = true
				break
			}
		}
	} else {
		// Default: 200-299 is success
		statusOK = resp.StatusCode >= 200 && resp.StatusCode < 300
	}

	if !statusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// If no output mapping, return raw response
	if respConfig == nil || len(respConfig.OutputMapping) == 0 {
		return map[string]interface{}{
			"status":  resp.StatusCode,
			"headers": headersToMap(resp.Header),
			"body":    bodyData,
		}, nil
	}

	// Apply output mapping
	result := make(map[string]interface{})
	responseData := map[string]interface{}{
		"status":  resp.StatusCode,
		"headers": headersToMap(resp.Header),
		"body":    bodyData,
	}

	for outputKey, sourcePath := range respConfig.OutputMapping {
		// Check for literal values (e.g., "true", "false", numbers in quotes)
		if sourcePath == "true" {
			result[outputKey] = true
			continue
		}
		if sourcePath == "false" {
			result[outputKey] = false
			continue
		}

		// Navigate the path
		value := getNestedValueAny(responseData, sourcePath)
		if value != nil {
			result[outputKey] = value
		}
	}

	return result, nil
}

// getNestedValueAny retrieves a nested value from a map, returning the actual value (not string)
func getNestedValueAny(data map[string]interface{}, path string) interface{} {
	if data == nil {
		return nil
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			if val, ok := v[part]; ok {
				current = val
			} else {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

// ExecuteWithDeclarative runs a block with declarative request/response configuration
// This combines declarative config processing with optional PreProcess/PostProcess code
func (s *Sandbox) ExecuteWithDeclarative(
	ctx context.Context,
	block *domain.BlockDefinition,
	config map[string]interface{},
	input map[string]interface{},
	execCtx *ExecutionContext,
) (map[string]interface{}, error) {
	// Build declarative context
	declCtx := &DeclarativeContext{
		Config:      config,
		Input:       input,
		Credentials: execCtx.Credentials,
	}

	// Execute PreProcess chain (if any)
	processedInput := input
	if len(block.PreProcessChain) > 0 {
		var err error
		processedInput, err = s.executePreProcessChain(ctx, block.PreProcessChain, config, input, execCtx)
		if err != nil {
			return nil, fmt.Errorf("preProcess failed: %w", err)
		}
		// Update declarative context with processed input
		declCtx.Input = processedInput
	}

	var result map[string]interface{}

	// Check if we have declarative request config
	if block.Request != nil && block.Request.URL != "" {
		// Build and execute HTTP request declaratively with context for cancellation
		httpResult, err := s.executeDeclarativeHTTP(ctx, block.Request, block.Response, declCtx, execCtx)
		if err != nil {
			return nil, err
		}
		result = httpResult
	} else if block.Code != "" {
		// Execute code-based block
		var err error
		result, err = s.Execute(ctx, block.Code, processedInput, execCtx)
		if err != nil {
			return nil, err
		}
	} else {
		// No execution logic - return processed input
		result = processedInput
	}

	// Execute PostProcess chain (if any)
	if len(block.PostProcessChain) > 0 {
		processedResult, err := s.executePostProcessChain(ctx, block.PostProcessChain, config, result, execCtx)
		if err != nil {
			return nil, fmt.Errorf("postProcess failed: %w", err)
		}
		result = processedResult
	}

	return result, nil
}

// executeDeclarativeHTTP executes an HTTP request using declarative configuration
func (s *Sandbox) executeDeclarativeHTTP(
	goCtx context.Context,
	reqConfig *domain.RequestConfig,
	respConfig *domain.ResponseConfig,
	declCtx *DeclarativeContext,
	execCtx *ExecutionContext,
) (map[string]interface{}, error) {
	if goCtx == nil {
		goCtx = context.Background()
	}

	// Build HTTP request with context for cancellation/timeout
	req, err := s.BuildDeclarativeRequest(goCtx, reqConfig, declCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Use HTTP client from execution context
	if execCtx == nil || execCtx.HTTP == nil {
		return nil, errors.New("HTTP client not available in execution context")
	}

	// Apply default headers from HTTP client
	for k, v := range execCtx.HTTP.getHeaders() {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}

	// Execute request
	resp, err := execCtx.HTTP.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Process response
	return s.ProcessDeclarativeResponse(respConfig, resp, respBody)
}

// executePreProcessChain executes the chain of preProcess code (child -> ... -> root)
func (s *Sandbox) executePreProcessChain(
	ctx context.Context,
	chain []string,
	config map[string]interface{},
	input map[string]interface{},
	execCtx *ExecutionContext,
) (map[string]interface{}, error) {
	current := input

	// Execute chain in order (child -> parent -> ... -> root)
	for _, code := range chain {
		if strings.TrimSpace(code) == "" {
			continue
		}

		// Create combined input for preProcess
		// preProcess receives: input (current data) and config
		preInput := map[string]interface{}{
			"data":   current,
			"config": config,
		}

		result, err := s.Execute(ctx, code, preInput, execCtx)
		if err != nil {
			return nil, err
		}
		current = result
	}

	return current, nil
}

// executePostProcessChain executes the chain of postProcess code (root -> ... -> child)
func (s *Sandbox) executePostProcessChain(
	ctx context.Context,
	chain []string,
	config map[string]interface{},
	output map[string]interface{},
	execCtx *ExecutionContext,
) (map[string]interface{}, error) {
	current := output

	// Execute chain in order (root -> ... -> parent -> child)
	for _, code := range chain {
		if strings.TrimSpace(code) == "" {
			continue
		}

		// Create combined input for postProcess
		// postProcess receives: input (current data) and config
		postInput := map[string]interface{}{
			"data":   current,
			"config": config,
		}

		result, err := s.Execute(ctx, code, postInput, execCtx)
		if err != nil {
			return nil, err
		}
		current = result
	}

	return current, nil
}
