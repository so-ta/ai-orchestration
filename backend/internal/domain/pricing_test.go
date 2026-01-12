package domain

import (
	"testing"
)

func TestGetPricing(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		want     bool // whether pricing should be found
	}{
		{
			name:     "OpenAI GPT-4o",
			provider: "openai",
			model:    "gpt-4o",
			want:     true,
		},
		{
			name:     "Anthropic Claude 3 Opus",
			provider: "anthropic",
			model:    "claude-3-opus",
			want:     true,
		},
		{
			name:     "Unknown model",
			provider: "unknown",
			model:    "unknown-model",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPricing(tt.provider, tt.model)
			if (got != nil) != tt.want {
				t.Errorf("GetPricing() = %v, want found = %v", got, tt.want)
			}
		})
	}
}

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		model        string
		inputTokens  int
		outputTokens int
		wantTotal    float64
	}{
		{
			name:         "GPT-4o with 1000 input and 500 output tokens",
			provider:     "openai",
			model:        "gpt-4o",
			inputTokens:  1000,
			outputTokens: 500,
			// 1000/1000 * 0.0025 + 500/1000 * 0.01 = 0.0025 + 0.005 = 0.0075
			wantTotal: 0.0075,
		},
		{
			name:         "Claude 3 Opus with 2000 input and 1000 output tokens",
			provider:     "anthropic",
			model:        "claude-3-opus",
			inputTokens:  2000,
			outputTokens: 1000,
			// 2000/1000 * 0.015 + 1000/1000 * 0.075 = 0.03 + 0.075 = 0.105
			wantTotal: 0.105,
		},
		{
			name:         "Unknown model returns zero",
			provider:     "unknown",
			model:        "unknown",
			inputTokens:  1000,
			outputTokens: 1000,
			wantTotal:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputCost, outputCost, totalCost := CalculateCost(tt.provider, tt.model, tt.inputTokens, tt.outputTokens)

			if totalCost != tt.wantTotal {
				t.Errorf("CalculateCost() totalCost = %v, want %v", totalCost, tt.wantTotal)
			}

			// Verify input + output = total
			if inputCost+outputCost != totalCost {
				t.Errorf("CalculateCost() inputCost + outputCost = %v, totalCost = %v", inputCost+outputCost, totalCost)
			}
		})
	}
}

func TestGetAllPricing(t *testing.T) {
	pricing := GetAllPricing()
	if len(pricing) == 0 {
		t.Error("GetAllPricing() returned empty slice")
	}

	// Verify all pricing entries have required fields
	for _, p := range pricing {
		if p.Provider == "" {
			t.Error("Pricing entry has empty Provider")
		}
		if p.Model == "" {
			t.Error("Pricing entry has empty Model")
		}
		if p.InputPer1K <= 0 {
			t.Errorf("Pricing entry %s/%s has invalid InputPer1K: %v", p.Provider, p.Model, p.InputPer1K)
		}
		if p.OutputPer1K <= 0 {
			t.Errorf("Pricing entry %s/%s has invalid OutputPer1K: %v", p.Provider, p.Model, p.OutputPer1K)
		}
	}
}

func TestGetProviders(t *testing.T) {
	providers := GetProviders()
	if len(providers) == 0 {
		t.Error("GetProviders() returned empty slice")
	}

	// Verify expected providers exist
	expectedProviders := []string{"openai", "anthropic", "google"}
	for _, expected := range expectedProviders {
		found := false
		for _, p := range providers {
			if p == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetProviders() missing expected provider: %s", expected)
		}
	}
}

func TestGetModelsByProvider(t *testing.T) {
	tests := []struct {
		provider string
		minCount int
	}{
		{"openai", 5},
		{"anthropic", 5},
		{"google", 2},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			models := GetModelsByProvider(tt.provider)
			if len(models) < tt.minCount {
				t.Errorf("GetModelsByProvider(%s) returned %d models, want at least %d", tt.provider, len(models), tt.minCount)
			}
		})
	}
}
