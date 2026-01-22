package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExecutionEventType represents the type of execution event
type ExecutionEventType string

const (
	// Step lifecycle events
	EventStepStarted   ExecutionEventType = "step:started"
	EventStepCompleted ExecutionEventType = "step:completed"
	EventStepFailed    ExecutionEventType = "step:failed"

	// Agent group events
	EventThinking    ExecutionEventType = "thinking"
	EventToolCall    ExecutionEventType = "tool:call"
	EventToolResult  ExecutionEventType = "tool:result"
	EventPartialText ExecutionEventType = "partial_text"

	// Run lifecycle events
	EventRunStarted   ExecutionEventType = "run:started"
	EventRunCompleted ExecutionEventType = "run:completed"
	EventRunFailed    ExecutionEventType = "run:failed"

	// Generic events
	EventProgress ExecutionEventType = "progress"
	EventComplete ExecutionEventType = "complete"
	EventError    ExecutionEventType = "error"
)

// ExecutionEvent represents an event during workflow execution
type ExecutionEvent struct {
	RunID     uuid.UUID          `json:"run_id"`
	Type      ExecutionEventType `json:"type"`
	Timestamp time.Time          `json:"timestamp"`
	Data      json.RawMessage    `json:"data"`
}

// NewExecutionEvent creates a new execution event
func NewExecutionEvent(runID uuid.UUID, eventType ExecutionEventType, data interface{}) ExecutionEvent {
	dataBytes, _ := json.Marshal(data)
	return ExecutionEvent{
		RunID:     runID,
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      dataBytes,
	}
}

// StepStartedData represents data for step:started event
type StepStartedData struct {
	StepID   string          `json:"step_id"`
	StepName string          `json:"step_name"`
	StepType string          `json:"step_type"`
	Input    json.RawMessage `json:"input,omitempty"`
}

// StepCompletedData represents data for step:completed event
type StepCompletedData struct {
	StepID   string          `json:"step_id"`
	StepName string          `json:"step_name"`
	Output   json.RawMessage `json:"output,omitempty"`
	Duration int64           `json:"duration_ms"`
}

// StepFailedData represents data for step:failed event
type StepFailedData struct {
	StepID   string `json:"step_id"`
	StepName string `json:"step_name"`
	Error    string `json:"error"`
}

// ThinkingData represents data for thinking event
type ThinkingData struct {
	Iteration int    `json:"iteration"`
	Content   string `json:"content,omitempty"`
}

// ToolCallData represents data for tool:call event
type ToolCallData struct {
	ToolName   string          `json:"tool_name"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
}

// ToolResultData represents data for tool:result event
type ToolResultData struct {
	ToolName   string          `json:"tool_name"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`
	IsError    bool            `json:"is_error"`
}

// PartialTextData represents data for partial_text event
type PartialTextData struct {
	Content string `json:"content"`
}

// RunStartedData represents data for run:started event
type RunStartedData struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name,omitempty"`
}

// RunCompletedData represents data for run:completed event
type RunCompletedData struct {
	Duration int64 `json:"duration_ms"`
}

// RunFailedData represents data for run:failed event
type RunFailedData struct {
	Error string `json:"error"`
}

// CompleteData represents data for complete event
type CompleteData struct {
	Response    string   `json:"response,omitempty"`
	ToolsUsed   []string `json:"tools_used,omitempty"`
	Iterations  int      `json:"iterations,omitempty"`
	TotalTokens int      `json:"total_tokens,omitempty"`
}

// EventEmitter is an interface for emitting execution events
type EventEmitter interface {
	Emit(event ExecutionEvent)
	Close()
}

// ChannelEventEmitter sends events to a Go channel
type ChannelEventEmitter struct {
	events chan<- ExecutionEvent
	closed bool
}

// NewChannelEventEmitter creates a new channel-based event emitter
func NewChannelEventEmitter(events chan<- ExecutionEvent) *ChannelEventEmitter {
	return &ChannelEventEmitter{
		events: events,
		closed: false,
	}
}

// Emit sends an event to the channel
func (e *ChannelEventEmitter) Emit(event ExecutionEvent) {
	if e == nil || e.closed || e.events == nil {
		return
	}
	select {
	case e.events <- event:
	default:
		// Channel full, skip event to prevent blocking
	}
}

// Close marks the emitter as closed
func (e *ChannelEventEmitter) Close() {
	if e != nil {
		e.closed = true
	}
}

// NoopEventEmitter is an emitter that does nothing (useful for testing)
type NoopEventEmitter struct{}

// Emit does nothing
func (e *NoopEventEmitter) Emit(event ExecutionEvent) {}

// Close does nothing
func (e *NoopEventEmitter) Close() {}

// EventBroadcaster broadcasts events to multiple subscribers
// This can be extended to use Redis Pub/Sub for distributed deployments
type EventBroadcaster struct {
	subscribers map[uuid.UUID][]chan ExecutionEvent
}

// NewEventBroadcaster creates a new event broadcaster
func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		subscribers: make(map[uuid.UUID][]chan ExecutionEvent),
	}
}

// Subscribe creates a new subscription for events of a specific run
func (b *EventBroadcaster) Subscribe(ctx context.Context, runID uuid.UUID, bufferSize int) <-chan ExecutionEvent {
	ch := make(chan ExecutionEvent, bufferSize)
	b.subscribers[runID] = append(b.subscribers[runID], ch)

	// Clean up on context cancellation
	go func() {
		<-ctx.Done()
		b.Unsubscribe(runID, ch)
	}()

	return ch
}

// Unsubscribe removes a subscription
func (b *EventBroadcaster) Unsubscribe(runID uuid.UUID, ch chan ExecutionEvent) {
	subs := b.subscribers[runID]
	for i, sub := range subs {
		if sub == ch {
			b.subscribers[runID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
	if len(b.subscribers[runID]) == 0 {
		delete(b.subscribers, runID)
	}
}

// Publish sends an event to all subscribers of the run
func (b *EventBroadcaster) Publish(event ExecutionEvent) {
	subs := b.subscribers[event.RunID]
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Channel full, skip event
		}
	}
}

// BroadcastingEventEmitter emits events to both a channel and a broadcaster
type BroadcastingEventEmitter struct {
	channel     *ChannelEventEmitter
	broadcaster *EventBroadcaster
}

// NewBroadcastingEventEmitter creates an emitter that sends to both channel and broadcaster
func NewBroadcastingEventEmitter(channel chan<- ExecutionEvent, broadcaster *EventBroadcaster) *BroadcastingEventEmitter {
	return &BroadcastingEventEmitter{
		channel:     NewChannelEventEmitter(channel),
		broadcaster: broadcaster,
	}
}

// Emit sends event to both channel and broadcaster
func (e *BroadcastingEventEmitter) Emit(event ExecutionEvent) {
	if e.channel != nil {
		e.channel.Emit(event)
	}
	if e.broadcaster != nil {
		e.broadcaster.Publish(event)
	}
}

// Close closes the emitter
func (e *BroadcastingEventEmitter) Close() {
	if e.channel != nil {
		e.channel.Close()
	}
}
