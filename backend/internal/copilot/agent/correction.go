package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// CorrectionStrategy defines how to handle specific error types
type CorrectionStrategy int

const (
	CorrectionNone CorrectionStrategy = iota
	CorrectionRetry
	CorrectionReformat
	CorrectionAlternative
	CorrectionAsk
)

// CorrectionConfig holds configuration for self-correction
type CorrectionConfig struct {
	MaxRetries        int
	RetryOnParseError bool
	RetryOnValidation bool
	RetryOnToolError  bool
}

// DefaultCorrectionConfig returns the default correction configuration
func DefaultCorrectionConfig() CorrectionConfig {
	return CorrectionConfig{
		MaxRetries:        3,
		RetryOnParseError: true,
		RetryOnValidation: true,
		RetryOnToolError:  true,
	}
}

// ErrorType categorizes errors for correction strategy selection
type ErrorType int

const (
	ErrorUnknown ErrorType = iota
	ErrorJSONParse
	ErrorValidation
	ErrorToolNotFound
	ErrorToolExecution
	ErrorRateLimit
	ErrorTimeout
	ErrorContext
)

// CorrectionResult represents the result of a correction attempt
type CorrectionResult struct {
	Strategy      CorrectionStrategy
	RetryMessage  string
	ShouldRetry   bool
	ErrorType     ErrorType
}

// SelfCorrector handles error recovery and retry logic
type SelfCorrector struct {
	config CorrectionConfig
}

// NewSelfCorrector creates a new self-corrector
func NewSelfCorrector(config CorrectionConfig) *SelfCorrector {
	return &SelfCorrector{config: config}
}

// AnalyzeError determines the error type and appropriate correction strategy
func (sc *SelfCorrector) AnalyzeError(err error, context string) *CorrectionResult {
	errStr := err.Error()

	// JSON parse error
	if strings.Contains(errStr, "json") || strings.Contains(errStr, "unmarshal") ||
		strings.Contains(errStr, "parse") || strings.Contains(errStr, "syntax") {
		if sc.config.RetryOnParseError {
			return &CorrectionResult{
				Strategy:     CorrectionReformat,
				RetryMessage: "前回のレスポンスがJSONとして解析できませんでした。マークダウンのコードブロックなしで、有効なJSONのみを返してください。",
				ShouldRetry:  true,
				ErrorType:    ErrorJSONParse,
			}
		}
		return &CorrectionResult{
			Strategy:    CorrectionNone,
			ShouldRetry: false,
			ErrorType:   ErrorJSONParse,
		}
	}

	// Validation error
	if strings.Contains(errStr, "validation") || strings.Contains(errStr, "invalid") ||
		strings.Contains(errStr, "required") || strings.Contains(errStr, "missing") {
		if sc.config.RetryOnValidation {
			return &CorrectionResult{
				Strategy:     CorrectionRetry,
				RetryMessage: fmt.Sprintf("バリデーションエラーが発生しました: %s\n\nこのエラーを修正して再度試してください。", errStr),
				ShouldRetry:  true,
				ErrorType:    ErrorValidation,
			}
		}
		return &CorrectionResult{
			Strategy:    CorrectionNone,
			ShouldRetry: false,
			ErrorType:   ErrorValidation,
		}
	}

	// Tool not found
	if strings.Contains(errStr, "tool") && strings.Contains(errStr, "not found") {
		return &CorrectionResult{
			Strategy:     CorrectionAlternative,
			RetryMessage: fmt.Sprintf("指定されたツールが見つかりませんでした: %s\n\n利用可能なツールのみを使用してください。", errStr),
			ShouldRetry:  true,
			ErrorType:    ErrorToolNotFound,
		}
	}

	// Tool execution error
	if strings.Contains(errStr, "execute") || strings.Contains(errStr, "execution") {
		if sc.config.RetryOnToolError {
			return &CorrectionResult{
				Strategy:     CorrectionRetry,
				RetryMessage: fmt.Sprintf("ツール実行でエラーが発生しました: %s\n\n異なるアプローチを試してください。", errStr),
				ShouldRetry:  true,
				ErrorType:    ErrorToolExecution,
			}
		}
		return &CorrectionResult{
			Strategy:    CorrectionNone,
			ShouldRetry: false,
			ErrorType:   ErrorToolExecution,
		}
	}

	// Rate limit
	if strings.Contains(errStr, "rate") || strings.Contains(errStr, "limit") ||
		strings.Contains(errStr, "429") {
		return &CorrectionResult{
			Strategy:    CorrectionNone,
			ShouldRetry: false,
			ErrorType:   ErrorRateLimit,
		}
	}

	// Timeout
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return &CorrectionResult{
			Strategy:    CorrectionNone,
			ShouldRetry: false,
			ErrorType:   ErrorTimeout,
		}
	}

	// Unknown error
	return &CorrectionResult{
		Strategy:    CorrectionAsk,
		ShouldRetry: false,
		ErrorType:   ErrorUnknown,
	}
}

// CreateRetryMessages creates messages to guide the LLM in recovering from errors
func (sc *SelfCorrector) CreateRetryMessages(result *CorrectionResult, originalResponse string) []Message {
	switch result.Strategy {
	case CorrectionReformat:
		return []Message{
			CreateTextMessage("user", result.RetryMessage),
		}
	case CorrectionRetry:
		return []Message{
			CreateTextMessage("user", result.RetryMessage),
		}
	case CorrectionAlternative:
		return []Message{
			CreateTextMessage("user", result.RetryMessage),
		}
	default:
		return nil
	}
}

// RetryableError wraps an error with retry information
type RetryableError struct {
	Original   error
	Correction *CorrectionResult
	Attempt    int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (attempt %d): %v", e.Attempt, e.Original)
}

// WithSelfCorrection wraps a function with self-correction logic
func WithSelfCorrection[T any](ctx context.Context, sc *SelfCorrector, fn func() (T, error)) (T, error) {
	var lastErr error
	var zero T

	for attempt := 0; attempt <= sc.config.MaxRetries; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err
		correction := sc.AnalyzeError(err, "")

		if !correction.ShouldRetry || attempt >= sc.config.MaxRetries {
			return zero, &RetryableError{
				Original:   err,
				Correction: correction,
				Attempt:    attempt + 1,
			}
		}

		// Log retry attempt
		// Continue to next attempt
	}

	return zero, lastErr
}

// ParseJSONWithCorrection attempts to parse JSON with format correction
func ParseJSONWithCorrection[T any](data []byte) (T, error) {
	var result T

	// First, try direct parse
	if err := json.Unmarshal(data, &result); err == nil {
		return result, nil
	}

	// Try to strip markdown code blocks
	cleaned := stripMarkdownCodeBlocks(string(data))
	if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
		return result, nil
	}

	// Try to find JSON object in the response
	extracted := extractJSON(string(data))
	if extracted != "" {
		if err := json.Unmarshal([]byte(extracted), &result); err == nil {
			return result, nil
		}
	}

	var zero T
	return zero, fmt.Errorf("failed to parse JSON from response")
}

// stripMarkdownCodeBlocks removes markdown code block wrappers
func stripMarkdownCodeBlocks(s string) string {
	s = strings.TrimSpace(s)

	// Remove ```json ... ``` wrapper
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}

	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}

	return strings.TrimSpace(s)
}

// extractJSON attempts to extract a JSON object or array from text
func extractJSON(s string) string {
	// Find the first { or [
	startObj := strings.Index(s, "{")
	startArr := strings.Index(s, "[")

	var start int
	var endChar byte

	if startObj == -1 && startArr == -1 {
		return ""
	} else if startObj == -1 {
		start = startArr
		endChar = ']'
	} else if startArr == -1 {
		start = startObj
		endChar = '}'
	} else if startObj < startArr {
		start = startObj
		endChar = '}'
	} else {
		start = startArr
		endChar = ']'
	}

	// Find matching end bracket
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 && s[i] == endChar {
				return s[start : i+1]
			}
		}
	}

	return ""
}
