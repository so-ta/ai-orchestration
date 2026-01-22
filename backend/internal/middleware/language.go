package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/souta/ai-orchestration/internal/domain"
)

// LanguageKey is the context key for storing the language
const LanguageKey contextKey = "language"

// LanguageMiddleware extracts the language from Accept-Language header
// and stores it in the request context
func LanguageMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := parseAcceptLanguage(r.Header.Get("Accept-Language"))
		ctx := context.WithValue(r.Context(), LanguageKey, lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetLanguage extracts the language from context
// Returns the default language if not set
func GetLanguage(ctx context.Context) string {
	if lang, ok := ctx.Value(LanguageKey).(string); ok && lang != "" {
		return lang
	}
	return domain.DefaultLanguage
}

// parseAcceptLanguage parses the Accept-Language header and returns
// the best matching supported language
// Examples:
//   - "ja" -> "ja"
//   - "ja-JP" -> "ja"
//   - "ja-JP,ja;q=0.9,en;q=0.8" -> "ja"
//   - "en-US,en;q=0.9" -> "en"
//   - "fr-FR,fr;q=0.9" -> "ja" (fallback to default)
func parseAcceptLanguage(header string) string {
	if header == "" {
		return domain.DefaultLanguage
	}

	// Parse comma-separated language preferences
	parts := strings.Split(header, ",")
	for _, part := range parts {
		// Remove quality value (e.g., ";q=0.9")
		lang := strings.TrimSpace(strings.Split(part, ";")[0])
		// Extract primary language tag (e.g., "ja-JP" -> "ja")
		lang = strings.ToLower(strings.Split(lang, "-")[0])

		// Check if this language is supported
		for _, supported := range domain.SupportedLanguages {
			if lang == supported {
				return lang
			}
		}
	}

	return domain.DefaultLanguage
}

// WithLanguage returns a new context with the specified language
func WithLanguage(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, LanguageKey, lang)
}
