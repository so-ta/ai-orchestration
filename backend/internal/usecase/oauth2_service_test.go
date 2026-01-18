package usecase

import (
	"testing"

	"github.com/souta/ai-orchestration/internal/domain"
)

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"32 chars", 32},
		{"64 chars", 64},
		{"16 chars", 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateRandomString(tt.length)
			if err != nil {
				t.Fatalf("generateRandomString(%d) returned error: %v", tt.length, err)
			}
			if len(result) != tt.length {
				t.Errorf("generateRandomString(%d) returned string of length %d, want %d", tt.length, len(result), tt.length)
			}
		})
	}
}

func TestGenerateRandomStringUniqueness(t *testing.T) {
	// Generate multiple strings and verify they're unique
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s, err := generateRandomString(32)
		if err != nil {
			t.Fatalf("generateRandomString returned error: %v", err)
		}
		if seen[s] {
			t.Errorf("generateRandomString returned duplicate string: %s", s)
		}
		seen[s] = true
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test-code-verifier-12345"
	challenge := generateCodeChallenge(verifier)

	// Verify it's base64url encoded
	if challenge == "" {
		t.Error("generateCodeChallenge returned empty string")
	}

	// Same verifier should produce same challenge
	challenge2 := generateCodeChallenge(verifier)
	if challenge != challenge2 {
		t.Error("generateCodeChallenge not deterministic")
	}

	// Different verifier should produce different challenge
	challenge3 := generateCodeChallenge("different-verifier")
	if challenge == challenge3 {
		t.Error("generateCodeChallenge produced same result for different inputs")
	}
}

func TestToProviderResponse(t *testing.T) {
	provider := &domain.OAuth2Provider{
		Slug:          "google",
		Name:          "Google",
		PKCERequired:  true,
		DefaultScopes: []string{"openid", "email"},
		IsPreset:      true,
	}

	resp := ToProviderResponse(provider)

	if resp.Slug != provider.Slug {
		t.Errorf("Slug = %s, want %s", resp.Slug, provider.Slug)
	}
	if resp.Name != provider.Name {
		t.Errorf("Name = %s, want %s", resp.Name, provider.Name)
	}
	if resp.PKCERequired != provider.PKCERequired {
		t.Errorf("PKCERequired = %v, want %v", resp.PKCERequired, provider.PKCERequired)
	}
	if len(resp.DefaultScopes) != len(provider.DefaultScopes) {
		t.Errorf("DefaultScopes length = %d, want %d", len(resp.DefaultScopes), len(provider.DefaultScopes))
	}
}

func TestToProviderResponses(t *testing.T) {
	providers := []*domain.OAuth2Provider{
		{Slug: "google", Name: "Google"},
		{Slug: "github", Name: "GitHub"},
	}

	responses := ToProviderResponses(providers)

	if len(responses) != len(providers) {
		t.Errorf("Response length = %d, want %d", len(responses), len(providers))
	}

	for i, resp := range responses {
		if resp.Slug != providers[i].Slug {
			t.Errorf("Response[%d].Slug = %s, want %s", i, resp.Slug, providers[i].Slug)
		}
	}
}

func TestToConnectionResponse(t *testing.T) {
	conn := &domain.OAuth2Connection{
		Status:       domain.OAuth2ConnectionStatusConnected,
		AccountEmail: "test@example.com",
		AccountName:  "Test User",
	}

	resp := ToConnectionResponse(conn)

	if resp.Status != conn.Status {
		t.Errorf("Status = %s, want %s", resp.Status, conn.Status)
	}
	if resp.AccountEmail != conn.AccountEmail {
		t.Errorf("AccountEmail = %s, want %s", resp.AccountEmail, conn.AccountEmail)
	}
	if resp.AccountName != conn.AccountName {
		t.Errorf("AccountName = %s, want %s", resp.AccountName, conn.AccountName)
	}
}
