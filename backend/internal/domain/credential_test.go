package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCredentialType_IsValid(t *testing.T) {
	validTypes := []CredentialType{
		CredentialTypeOAuth2,
		CredentialTypeAPIKey,
		CredentialTypeBasic,
		CredentialTypeBearer,
		CredentialTypeCustom,
		CredentialTypeQueryAuth,
		CredentialTypeHeaderAuth,
	}

	for _, ct := range validTypes {
		t.Run(string(ct), func(t *testing.T) {
			if !ct.IsValid() {
				t.Errorf("IsValid() = false for valid type %v", ct)
			}
		})
	}

	invalidTypes := []CredentialType{
		CredentialType("invalid"),
		CredentialType(""),
	}

	for _, ct := range invalidTypes {
		t.Run(string(ct), func(t *testing.T) {
			if ct.IsValid() {
				t.Errorf("IsValid() = true for invalid type %v", ct)
			}
		})
	}
}

func TestOwnerScope_IsValid(t *testing.T) {
	validScopes := []OwnerScope{
		OwnerScopeOrganization,
		OwnerScopeProject,
		OwnerScopePersonal,
	}

	for _, scope := range validScopes {
		t.Run(string(scope), func(t *testing.T) {
			if !scope.IsValid() {
				t.Errorf("IsValid() = false for valid scope %v", scope)
			}
		})
	}

	invalidScopes := []OwnerScope{
		OwnerScope("invalid"),
		OwnerScope(""),
	}

	for _, scope := range invalidScopes {
		t.Run(string(scope), func(t *testing.T) {
			if scope.IsValid() {
				t.Errorf("IsValid() = true for invalid scope %v", scope)
			}
		})
	}
}

func TestNewCredential(t *testing.T) {
	tenantID := uuid.New()
	name := "Test Credential"
	credType := CredentialTypeAPIKey

	cred := NewCredential(tenantID, name, credType)

	if cred.ID == uuid.Nil {
		t.Error("NewCredential() should generate a non-nil UUID")
	}
	if cred.TenantID != tenantID {
		t.Errorf("NewCredential() TenantID = %v, want %v", cred.TenantID, tenantID)
	}
	if cred.Name != name {
		t.Errorf("NewCredential() Name = %v, want %v", cred.Name, name)
	}
	if cred.CredentialType != credType {
		t.Errorf("NewCredential() CredentialType = %v, want %v", cred.CredentialType, credType)
	}
	if cred.Scope != OwnerScopeOrganization {
		t.Errorf("NewCredential() Scope = %v, want %v", cred.Scope, OwnerScopeOrganization)
	}
	if cred.Status != CredentialStatusActive {
		t.Errorf("NewCredential() Status = %v, want %v", cred.Status, CredentialStatusActive)
	}
}

func TestNewProjectCredential(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	name := "Project Credential"
	credType := CredentialTypeAPIKey

	cred := NewProjectCredential(tenantID, projectID, name, credType)

	if cred.Scope != OwnerScopeProject {
		t.Errorf("NewProjectCredential() Scope = %v, want %v", cred.Scope, OwnerScopeProject)
	}
	if cred.ProjectID == nil || *cred.ProjectID != projectID {
		t.Error("NewProjectCredential() ProjectID mismatch")
	}
}

func TestNewPersonalCredential(t *testing.T) {
	tenantID := uuid.New()
	ownerUserID := uuid.New()
	name := "Personal Credential"
	credType := CredentialTypeAPIKey

	cred := NewPersonalCredential(tenantID, ownerUserID, name, credType)

	if cred.Scope != OwnerScopePersonal {
		t.Errorf("NewPersonalCredential() Scope = %v, want %v", cred.Scope, OwnerScopePersonal)
	}
	if cred.OwnerUserID == nil || *cred.OwnerUserID != ownerUserID {
		t.Error("NewPersonalCredential() OwnerUserID mismatch")
	}
}

func TestCredential_ValidateScope(t *testing.T) {
	projectID := uuid.New()
	ownerUserID := uuid.New()

	tests := []struct {
		name        string
		scope       OwnerScope
		projectID   *uuid.UUID
		ownerUserID *uuid.UUID
		wantErr     bool
	}{
		{"organization valid", OwnerScopeOrganization, nil, nil, false},
		{"organization with project", OwnerScopeOrganization, &projectID, nil, true},
		{"organization with owner", OwnerScopeOrganization, nil, &ownerUserID, true},
		{"project valid", OwnerScopeProject, &projectID, nil, false},
		{"project without projectID", OwnerScopeProject, nil, nil, true},
		{"project with owner", OwnerScopeProject, &projectID, &ownerUserID, true},
		{"personal valid", OwnerScopePersonal, nil, &ownerUserID, false},
		{"personal without owner", OwnerScopePersonal, nil, nil, true},
		{"personal with project", OwnerScopePersonal, &projectID, &ownerUserID, true},
		{"invalid scope", OwnerScope("invalid"), nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &Credential{
				Scope:       tt.scope,
				ProjectID:   tt.projectID,
				OwnerUserID: tt.ownerUserID,
			}
			err := cred.ValidateScope()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScope() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCredential_IsExpired(t *testing.T) {
	pastTime := time.Now().Add(-time.Hour)
	futureTime := time.Now().Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"no expiry", nil, false},
		{"expired", &pastTime, true},
		{"not expired", &futureTime, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &Credential{ExpiresAt: tt.expiresAt}
			if got := cred.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCredential_IsActive(t *testing.T) {
	pastTime := time.Now().Add(-time.Hour)
	futureTime := time.Now().Add(time.Hour)

	tests := []struct {
		name      string
		status    CredentialStatus
		expiresAt *time.Time
		want      bool
	}{
		{"active not expired", CredentialStatusActive, nil, true},
		{"active with future expiry", CredentialStatusActive, &futureTime, true},
		{"active but expired", CredentialStatusActive, &pastTime, false},
		{"revoked", CredentialStatusRevoked, nil, false},
		{"error status", CredentialStatusError, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &Credential{Status: tt.status, ExpiresAt: tt.expiresAt}
			if got := cred.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCredentialData_GetSecretValue(t *testing.T) {
	tests := []struct {
		name string
		data CredentialData
		want string
	}{
		{
			"api key",
			CredentialData{Type: string(CredentialTypeAPIKey), APIKey: "sk-123"},
			"sk-123",
		},
		{
			"bearer",
			CredentialData{Type: string(CredentialTypeBearer), AccessToken: "token123"},
			"token123",
		},
		{
			"oauth2",
			CredentialData{Type: string(CredentialTypeOAuth2), AccessToken: "oauth-token"},
			"oauth-token",
		},
		{
			"basic",
			CredentialData{Type: string(CredentialTypeBasic), Username: "user", Password: "pass"},
			"user:pass",
		},
		{
			"custom",
			CredentialData{Type: string(CredentialTypeCustom)},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.GetSecretValue(); got != tt.want {
				t.Errorf("GetSecretValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCredentialData_ToJSON(t *testing.T) {
	data := &CredentialData{
		Type:   string(CredentialTypeAPIKey),
		APIKey: "sk-123",
	}

	jsonData, err := data.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var result CredentialData
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if result.APIKey != "sk-123" {
		t.Errorf("ToJSON() APIKey = %v, want sk-123", result.APIKey)
	}
}

func TestCredentialDataFromJSON(t *testing.T) {
	jsonData := []byte(`{"type": "api_key", "api_key": "sk-123"}`)

	data, err := CredentialDataFromJSON(jsonData)
	if err != nil {
		t.Fatalf("CredentialDataFromJSON() error = %v", err)
	}

	if data.Type != string(CredentialTypeAPIKey) {
		t.Errorf("Type = %v, want api_key", data.Type)
	}
	if data.APIKey != "sk-123" {
		t.Errorf("APIKey = %v, want sk-123", data.APIKey)
	}
}

func TestDecryptedCredential_GetAuthHeader(t *testing.T) {
	tests := []struct {
		name        string
		credType    CredentialType
		data        *CredentialData
		wantName    string
		wantPrefix  string
	}{
		{
			"api key default header",
			CredentialTypeAPIKey,
			&CredentialData{APIKey: "sk-123"},
			"Authorization",
			"sk-123",
		},
		{
			"api key custom header",
			CredentialTypeAPIKey,
			&CredentialData{APIKey: "sk-123", HeaderName: "X-API-Key", HeaderPrefix: ""},
			"X-API-Key",
			"sk-123",
		},
		{
			"bearer",
			CredentialTypeBearer,
			&CredentialData{AccessToken: "token123"},
			"Authorization",
			"Bearer token123",
		},
		{
			"oauth2",
			CredentialTypeOAuth2,
			&CredentialData{AccessToken: "token123", TokenType: "Bearer"},
			"Authorization",
			"Bearer token123",
		},
		{
			"basic",
			CredentialTypeBasic,
			&CredentialData{Username: "user", Password: "pass"},
			"Authorization",
			"Basic dXNlcjpwYXNz",
		},
		{
			"nil data",
			CredentialTypeAPIKey,
			nil,
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &DecryptedCredential{
				Credential: &Credential{CredentialType: tt.credType},
				Data:       tt.data,
			}
			gotName, gotValue := dc.GetAuthHeader()
			if gotName != tt.wantName {
				t.Errorf("GetAuthHeader() name = %v, want %v", gotName, tt.wantName)
			}
			if gotValue != tt.wantPrefix {
				t.Errorf("GetAuthHeader() value = %v, want %v", gotValue, tt.wantPrefix)
			}
		})
	}
}

func TestNewSystemCredential(t *testing.T) {
	name := "System API Key"
	credType := CredentialTypeAPIKey

	cred := NewSystemCredential(name, credType)

	if cred.ID == uuid.Nil {
		t.Error("NewSystemCredential() should generate a non-nil UUID")
	}
	if cred.Name != name {
		t.Errorf("NewSystemCredential() Name = %v, want %v", cred.Name, name)
	}
	if cred.CredentialType != credType {
		t.Errorf("NewSystemCredential() CredentialType = %v, want %v", cred.CredentialType, credType)
	}
	if cred.Status != CredentialStatusActive {
		t.Errorf("NewSystemCredential() Status = %v, want %v", cred.Status, CredentialStatusActive)
	}
}

func TestSystemCredential_IsExpired(t *testing.T) {
	pastTime := time.Now().Add(-time.Hour)
	futureTime := time.Now().Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"no expiry", nil, false},
		{"expired", &pastTime, true},
		{"not expired", &futureTime, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &SystemCredential{ExpiresAt: tt.expiresAt}
			if got := cred.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSystemCredential_IsActive(t *testing.T) {
	pastTime := time.Now().Add(-time.Hour)

	tests := []struct {
		name      string
		status    CredentialStatus
		expiresAt *time.Time
		want      bool
	}{
		{"active", CredentialStatusActive, nil, true},
		{"active expired", CredentialStatusActive, &pastTime, false},
		{"revoked", CredentialStatusRevoked, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &SystemCredential{Status: tt.status, ExpiresAt: tt.expiresAt}
			if got := cred.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRequiredCredentials(t *testing.T) {
	tests := []struct {
		name    string
		data    json.RawMessage
		want    int
		wantErr bool
	}{
		{"nil data", nil, 0, false},
		{"empty data", json.RawMessage(""), 0, false},
		{"null data", json.RawMessage("null"), 0, false},
		{
			"valid data",
			json.RawMessage(`[{"name": "api_key", "type": "api_key", "scope": "system", "required": true}]`),
			1,
			false,
		},
		{"invalid json", json.RawMessage("invalid"), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := ParseRequiredCredentials(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequiredCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(creds) != tt.want {
				t.Errorf("ParseRequiredCredentials() len = %v, want %v", len(creds), tt.want)
			}
		})
	}
}

func TestParseCredentialBindings(t *testing.T) {
	credID := uuid.New()

	tests := []struct {
		name    string
		data    json.RawMessage
		want    int
		wantErr bool
	}{
		{"nil data", nil, 0, false},
		{"empty data", json.RawMessage(""), 0, false},
		{"null data", json.RawMessage("null"), 0, false},
		{
			"valid data",
			json.RawMessage(`{"api_key": "` + credID.String() + `"}`),
			1,
			false,
		},
		{"invalid json", json.RawMessage("invalid"), 0, true},
		{"invalid uuid", json.RawMessage(`{"api_key": "not-a-uuid"}`), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bindings, err := ParseCredentialBindings(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCredentialBindings() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(bindings) != tt.want {
				t.Errorf("ParseCredentialBindings() len = %v, want %v", len(bindings), tt.want)
			}
		})
	}
}
