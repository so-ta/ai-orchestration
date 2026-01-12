package domain

// TokenPricing represents pricing per 1000 tokens for a specific model
type TokenPricing struct {
	Provider    string  // LLM provider (openai, anthropic, google)
	Model       string  // Model identifier
	InputPer1K  float64 // USD per 1K input tokens
	OutputPer1K float64 // USD per 1K output tokens
}

// DefaultPricing contains pricing for common LLM models
// Prices are in USD per 1000 tokens
// Last updated: 2024-01
var DefaultPricing = []TokenPricing{
	// OpenAI models
	{"openai", "gpt-4o", 0.0025, 0.01},
	{"openai", "gpt-4o-2024-11-20", 0.0025, 0.01},
	{"openai", "gpt-4o-2024-08-06", 0.0025, 0.01},
	{"openai", "gpt-4o-2024-05-13", 0.005, 0.015},
	{"openai", "gpt-4o-mini", 0.00015, 0.0006},
	{"openai", "gpt-4o-mini-2024-07-18", 0.00015, 0.0006},
	{"openai", "gpt-4-turbo", 0.01, 0.03},
	{"openai", "gpt-4-turbo-2024-04-09", 0.01, 0.03},
	{"openai", "gpt-4-turbo-preview", 0.01, 0.03},
	{"openai", "gpt-4", 0.03, 0.06},
	{"openai", "gpt-4-0613", 0.03, 0.06},
	{"openai", "gpt-3.5-turbo", 0.0005, 0.0015},
	{"openai", "gpt-3.5-turbo-0125", 0.0005, 0.0015},
	{"openai", "gpt-3.5-turbo-1106", 0.001, 0.002},

	// Anthropic models
	{"anthropic", "claude-3-opus-20240229", 0.015, 0.075},
	{"anthropic", "claude-3-opus", 0.015, 0.075},
	{"anthropic", "claude-3-sonnet-20240229", 0.003, 0.015},
	{"anthropic", "claude-3-sonnet", 0.003, 0.015},
	{"anthropic", "claude-3-haiku-20240307", 0.00025, 0.00125},
	{"anthropic", "claude-3-haiku", 0.00025, 0.00125},
	{"anthropic", "claude-3-5-sonnet-20241022", 0.003, 0.015},
	{"anthropic", "claude-3-5-sonnet-20240620", 0.003, 0.015},
	{"anthropic", "claude-3-5-sonnet", 0.003, 0.015},
	{"anthropic", "claude-3-5-haiku-20241022", 0.0008, 0.004},
	{"anthropic", "claude-3-5-haiku", 0.0008, 0.004},

	// Google models
	{"google", "gemini-1.5-pro", 0.00125, 0.005},
	{"google", "gemini-1.5-pro-latest", 0.00125, 0.005},
	{"google", "gemini-1.5-flash", 0.000075, 0.0003},
	{"google", "gemini-1.5-flash-latest", 0.000075, 0.0003},
	{"google", "gemini-1.0-pro", 0.0005, 0.0015},
}

// pricingIndex is a map for O(1) lookup
var pricingIndex map[string]*TokenPricing

func init() {
	pricingIndex = make(map[string]*TokenPricing)
	for i := range DefaultPricing {
		key := DefaultPricing[i].Provider + ":" + DefaultPricing[i].Model
		pricingIndex[key] = &DefaultPricing[i]
	}
}

// GetPricing returns the pricing for a specific provider and model
// Returns nil if no pricing is found
func GetPricing(provider, model string) *TokenPricing {
	key := provider + ":" + model
	if pricing, ok := pricingIndex[key]; ok {
		return pricing
	}
	return nil
}

// CalculateCost calculates the cost in USD for given token counts
// Returns inputCost, outputCost, totalCost
// If no pricing is found, returns 0 for all values
func CalculateCost(provider, model string, inputTokens, outputTokens int) (inputCost, outputCost, totalCost float64) {
	pricing := GetPricing(provider, model)
	if pricing == nil {
		return 0, 0, 0
	}

	inputCost = float64(inputTokens) / 1000.0 * pricing.InputPer1K
	outputCost = float64(outputTokens) / 1000.0 * pricing.OutputPer1K
	totalCost = inputCost + outputCost
	return
}

// GetAllPricing returns all available pricing configurations
func GetAllPricing() []TokenPricing {
	return DefaultPricing
}

// GetProviders returns a list of all supported providers
func GetProviders() []string {
	providers := make(map[string]bool)
	for _, p := range DefaultPricing {
		providers[p.Provider] = true
	}

	result := make([]string, 0, len(providers))
	for provider := range providers {
		result = append(result, provider)
	}
	return result
}

// GetModelsByProvider returns all models for a specific provider
func GetModelsByProvider(provider string) []string {
	var models []string
	for _, p := range DefaultPricing {
		if p.Provider == provider {
			models = append(models, p.Model)
		}
	}
	return models
}
