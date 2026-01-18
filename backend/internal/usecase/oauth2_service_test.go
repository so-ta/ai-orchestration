package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/souta/ai-orchestration/pkg/crypto"
)

// ============================================================================
// Mock Repositories
// ============================================================================

type mockOAuth2ProviderRepo struct {
	providers    map[string]*domain.OAuth2Provider
	providerByID map[uuid.UUID]*domain.OAuth2Provider
	listErr      error
	getErr       error
}

func newMockOAuth2ProviderRepo() *mockOAuth2ProviderRepo {
	return &mockOAuth2ProviderRepo{
		providers:    make(map[string]*domain.OAuth2Provider),
		providerByID: make(map[uuid.UUID]*domain.OAuth2Provider),
	}
}

func (m *mockOAuth2ProviderRepo) addProvider(p *domain.OAuth2Provider) {
	m.providers[p.Slug] = p
	m.providerByID[p.ID] = p
}

func (m *mockOAuth2ProviderRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.OAuth2Provider, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	p, ok := m.providerByID[id]
	if !ok {
		return nil, domain.ErrOAuth2ProviderNotFound
	}
	return p, nil
}

func (m *mockOAuth2ProviderRepo) GetBySlug(ctx context.Context, slug string) (*domain.OAuth2Provider, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	p, ok := m.providers[slug]
	if !ok {
		return nil, domain.ErrOAuth2ProviderNotFound
	}
	return p, nil
}

func (m *mockOAuth2ProviderRepo) List(ctx context.Context) ([]*domain.OAuth2Provider, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*domain.OAuth2Provider
	for _, p := range m.providers {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockOAuth2ProviderRepo) ListPresets(ctx context.Context) ([]*domain.OAuth2Provider, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*domain.OAuth2Provider
	for _, p := range m.providers {
		if p.IsPreset {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockOAuth2ProviderRepo) Create(ctx context.Context, provider *domain.OAuth2Provider) error {
	m.providers[provider.Slug] = provider
	m.providerByID[provider.ID] = provider
	return nil
}

func (m *mockOAuth2ProviderRepo) Update(ctx context.Context, provider *domain.OAuth2Provider) error {
	m.providers[provider.Slug] = provider
	m.providerByID[provider.ID] = provider
	return nil
}

func (m *mockOAuth2ProviderRepo) Delete(ctx context.Context, id uuid.UUID) error {
	p, ok := m.providerByID[id]
	if !ok {
		return domain.ErrOAuth2ProviderNotFound
	}
	delete(m.providers, p.Slug)
	delete(m.providerByID, id)
	return nil
}

type mockOAuth2AppRepo struct {
	apps             map[uuid.UUID]*domain.OAuth2App
	tenantProviderID map[string]*domain.OAuth2App
	createErr        error
	getErr           error
}

func newMockOAuth2AppRepo() *mockOAuth2AppRepo {
	return &mockOAuth2AppRepo{
		apps:             make(map[uuid.UUID]*domain.OAuth2App),
		tenantProviderID: make(map[string]*domain.OAuth2App),
	}
}

func (m *mockOAuth2AppRepo) addApp(app *domain.OAuth2App) {
	m.apps[app.ID] = app
	key := app.TenantID.String() + "-" + app.ProviderID.String()
	m.tenantProviderID[key] = app
}

func (m *mockOAuth2AppRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.OAuth2App, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	app, ok := m.apps[id]
	if !ok {
		return nil, domain.ErrOAuth2AppNotFound
	}
	return app, nil
}

func (m *mockOAuth2AppRepo) GetByTenantAndProvider(ctx context.Context, tenantID, providerID uuid.UUID) (*domain.OAuth2App, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	key := tenantID.String() + "-" + providerID.String()
	app, ok := m.tenantProviderID[key]
	if !ok {
		return nil, domain.ErrOAuth2AppNotFound
	}
	return app, nil
}

func (m *mockOAuth2AppRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.OAuth2App, error) {
	var result []*domain.OAuth2App
	for _, app := range m.apps {
		if app.TenantID == tenantID {
			result = append(result, app)
		}
	}
	return result, nil
}

func (m *mockOAuth2AppRepo) Create(ctx context.Context, app *domain.OAuth2App) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.addApp(app)
	return nil
}

func (m *mockOAuth2AppRepo) Update(ctx context.Context, app *domain.OAuth2App) error {
	m.apps[app.ID] = app
	return nil
}

func (m *mockOAuth2AppRepo) Delete(ctx context.Context, id uuid.UUID) error {
	app, ok := m.apps[id]
	if !ok {
		return domain.ErrOAuth2AppNotFound
	}
	delete(m.apps, id)
	key := app.TenantID.String() + "-" + app.ProviderID.String()
	delete(m.tenantProviderID, key)
	return nil
}

type mockOAuth2ConnectionRepo struct {
	connections    map[uuid.UUID]*domain.OAuth2Connection
	byState        map[string]*domain.OAuth2Connection
	byCredentialID map[uuid.UUID]*domain.OAuth2Connection
	createErr      error
	getErr         error
}

func newMockOAuth2ConnectionRepo() *mockOAuth2ConnectionRepo {
	return &mockOAuth2ConnectionRepo{
		connections:    make(map[uuid.UUID]*domain.OAuth2Connection),
		byState:        make(map[string]*domain.OAuth2Connection),
		byCredentialID: make(map[uuid.UUID]*domain.OAuth2Connection),
	}
}

func (m *mockOAuth2ConnectionRepo) addConnection(conn *domain.OAuth2Connection) {
	m.connections[conn.ID] = conn
	if conn.State != "" {
		m.byState[conn.State] = conn
	}
	m.byCredentialID[conn.CredentialID] = conn
}

func (m *mockOAuth2ConnectionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.OAuth2Connection, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	conn, ok := m.connections[id]
	if !ok {
		return nil, domain.ErrOAuth2ConnectionNotFound
	}
	return conn, nil
}

func (m *mockOAuth2ConnectionRepo) GetByState(ctx context.Context, state string) (*domain.OAuth2Connection, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	conn, ok := m.byState[state]
	if !ok {
		return nil, domain.ErrOAuth2ConnectionNotFound
	}
	return conn, nil
}

func (m *mockOAuth2ConnectionRepo) GetByCredentialID(ctx context.Context, credentialID uuid.UUID) (*domain.OAuth2Connection, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	conn, ok := m.byCredentialID[credentialID]
	if !ok {
		return nil, domain.ErrOAuth2ConnectionNotFound
	}
	return conn, nil
}

func (m *mockOAuth2ConnectionRepo) ListByApp(ctx context.Context, appID uuid.UUID) ([]*domain.OAuth2Connection, error) {
	var result []*domain.OAuth2Connection
	for _, conn := range m.connections {
		if conn.OAuth2AppID == appID {
			result = append(result, conn)
		}
	}
	return result, nil
}

func (m *mockOAuth2ConnectionRepo) Create(ctx context.Context, conn *domain.OAuth2Connection) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.addConnection(conn)
	return nil
}

func (m *mockOAuth2ConnectionRepo) Update(ctx context.Context, conn *domain.OAuth2Connection) error {
	m.connections[conn.ID] = conn
	m.byCredentialID[conn.CredentialID] = conn
	// Clear old state if changed
	for state, c := range m.byState {
		if c.ID == conn.ID && state != conn.State {
			delete(m.byState, state)
		}
	}
	if conn.State != "" {
		m.byState[conn.State] = conn
	}
	return nil
}

func (m *mockOAuth2ConnectionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	conn, ok := m.connections[id]
	if !ok {
		return domain.ErrOAuth2ConnectionNotFound
	}
	delete(m.connections, id)
	delete(m.byCredentialID, conn.CredentialID)
	if conn.State != "" {
		delete(m.byState, conn.State)
	}
	return nil
}

func (m *mockOAuth2ConnectionRepo) ListExpiring(ctx context.Context, within time.Duration) ([]*domain.OAuth2Connection, error) {
	var result []*domain.OAuth2Connection
	deadline := time.Now().Add(within)
	for _, conn := range m.connections {
		if conn.AccessTokenExpiresAt != nil && conn.AccessTokenExpiresAt.Before(deadline) {
			result = append(result, conn)
		}
	}
	return result, nil
}

type mockCredentialRepo struct {
	credentials   map[uuid.UUID]*domain.Credential
	credsByName   map[string]*domain.Credential
	createErr     error
	getErr        error
}

func newMockCredentialRepo() *mockCredentialRepo {
	return &mockCredentialRepo{
		credentials:   make(map[uuid.UUID]*domain.Credential),
		credsByName:   make(map[string]*domain.Credential),
	}
}

func (m *mockCredentialRepo) addCredential(cred *domain.Credential) {
	m.credentials[cred.ID] = cred
	key := cred.TenantID.String() + "-" + cred.Name
	m.credsByName[key] = cred
}

func (m *mockCredentialRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Credential, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	cred, ok := m.credentials[id]
	if !ok {
		return nil, domain.ErrCredentialNotFound
	}
	if cred.TenantID != tenantID {
		return nil, domain.ErrCredentialNotFound
	}
	return cred, nil
}

func (m *mockCredentialRepo) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Credential, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	key := tenantID.String() + "-" + name
	cred, ok := m.credsByName[key]
	if !ok {
		return nil, domain.ErrCredentialNotFound
	}
	return cred, nil
}

func (m *mockCredentialRepo) List(ctx context.Context, tenantID uuid.UUID, filter repository.CredentialFilter) ([]*domain.Credential, int, error) {
	var result []*domain.Credential
	for _, cred := range m.credentials {
		if cred.TenantID == tenantID {
			result = append(result, cred)
		}
	}
	return result, len(result), nil
}

func (m *mockCredentialRepo) Create(ctx context.Context, cred *domain.Credential) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.addCredential(cred)
	return nil
}

func (m *mockCredentialRepo) Update(ctx context.Context, cred *domain.Credential) error {
	m.credentials[cred.ID] = cred
	key := cred.TenantID.String() + "-" + cred.Name
	m.credsByName[key] = cred
	return nil
}

func (m *mockCredentialRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	cred, ok := m.credentials[id]
	if !ok {
		return domain.ErrCredentialNotFound
	}
	if cred.TenantID != tenantID {
		return domain.ErrCredentialNotFound
	}
	key := cred.TenantID.String() + "-" + cred.Name
	delete(m.credsByName, key)
	delete(m.credentials, id)
	return nil
}

func (m *mockCredentialRepo) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.CredentialStatus) error {
	cred, ok := m.credentials[id]
	if !ok {
		return domain.ErrCredentialNotFound
	}
	if cred.TenantID != tenantID {
		return domain.ErrCredentialNotFound
	}
	cred.Status = status
	return nil
}

// ============================================================================
// Test Helpers
// ============================================================================

func createTestEncryptor(t *testing.T) *crypto.Encryptor {
	// Use test key (32 bytes)
	key := []byte("01234567890123456789012345678901")
	enc, err := crypto.NewEncryptorWithKey(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	return enc
}

func createTestProvider() *domain.OAuth2Provider {
	return &domain.OAuth2Provider{
		ID:               uuid.New(),
		Slug:             "google",
		Name:             "Google",
		AuthorizationURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:         "https://oauth2.googleapis.com/token",
		UserinfoURL:      "https://www.googleapis.com/oauth2/v3/userinfo",
		PKCERequired:     true,
		DefaultScopes:    []string{"openid", "email", "profile"},
		IsPreset:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func createTestApp(tenantID, providerID uuid.UUID, enc *crypto.Encryptor) *domain.OAuth2App {
	clientIDEnc, clientIDNonce, _ := enc.EncryptDEK([]byte("test-client-id"))
	clientSecretEnc, clientSecretNonce, _ := enc.EncryptDEK([]byte("test-client-secret"))

	return &domain.OAuth2App{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		ProviderID:            providerID,
		EncryptedClientID:     clientIDEnc,
		ClientIDNonce:         clientIDNonce,
		EncryptedClientSecret: clientSecretEnc,
		ClientSecretNonce:     clientSecretNonce,
		Status:                domain.OAuth2AppStatusActive,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

// ============================================================================
// OAuth2Service Tests
// ============================================================================

func TestOAuth2Service_ListProviders(t *testing.T) {
	providerRepo := newMockOAuth2ProviderRepo()
	appRepo := newMockOAuth2AppRepo()
	connRepo := newMockOAuth2ConnectionRepo()
	credRepo := newMockCredentialRepo()
	enc := createTestEncryptor(t)

	service := NewOAuth2Service(providerRepo, appRepo, connRepo, credRepo, enc, "http://localhost:8090")

	// Add test providers
	google := createTestProvider()
	github := &domain.OAuth2Provider{
		ID:       uuid.New(),
		Slug:     "github",
		Name:     "GitHub",
		IsPreset: true,
	}
	providerRepo.addProvider(google)
	providerRepo.addProvider(github)

	t.Run("success", func(t *testing.T) {
		providers, err := service.ListProviders(context.Background())
		if err != nil {
			t.Fatalf("ListProviders() error = %v", err)
		}
		if len(providers) != 2 {
			t.Errorf("ListProviders() got %d providers, want 2", len(providers))
		}
	})

	t.Run("with error", func(t *testing.T) {
		providerRepo.listErr = domain.ErrOAuth2ProviderNotFound
		defer func() { providerRepo.listErr = nil }()

		_, err := service.ListProviders(context.Background())
		if err == nil {
			t.Error("ListProviders() expected error, got nil")
		}
	})
}

func TestOAuth2Service_ListPresetProviders(t *testing.T) {
	providerRepo := newMockOAuth2ProviderRepo()
	appRepo := newMockOAuth2AppRepo()
	connRepo := newMockOAuth2ConnectionRepo()
	credRepo := newMockCredentialRepo()
	enc := createTestEncryptor(t)

	service := NewOAuth2Service(providerRepo, appRepo, connRepo, credRepo, enc, "http://localhost:8090")

	// Add test providers
	google := createTestProvider()
	custom := &domain.OAuth2Provider{
		ID:       uuid.New(),
		Slug:     "custom",
		Name:     "Custom Provider",
		IsPreset: false,
	}
	providerRepo.addProvider(google)
	providerRepo.addProvider(custom)

	t.Run("returns only presets", func(t *testing.T) {
		providers, err := service.ListPresetProviders(context.Background())
		if err != nil {
			t.Fatalf("ListPresetProviders() error = %v", err)
		}
		if len(providers) != 1 {
			t.Errorf("ListPresetProviders() got %d providers, want 1", len(providers))
		}
		if providers[0].Slug != "google" {
			t.Errorf("ListPresetProviders() got slug %s, want google", providers[0].Slug)
		}
	})
}

func TestOAuth2Service_GetProvider(t *testing.T) {
	providerRepo := newMockOAuth2ProviderRepo()
	appRepo := newMockOAuth2AppRepo()
	connRepo := newMockOAuth2ConnectionRepo()
	credRepo := newMockCredentialRepo()
	enc := createTestEncryptor(t)

	service := NewOAuth2Service(providerRepo, appRepo, connRepo, credRepo, enc, "http://localhost:8090")

	google := createTestProvider()
	providerRepo.addProvider(google)

	t.Run("success", func(t *testing.T) {
		provider, err := service.GetProvider(context.Background(), "google")
		if err != nil {
			t.Fatalf("GetProvider() error = %v", err)
		}
		if provider.Slug != "google" {
			t.Errorf("GetProvider() slug = %s, want google", provider.Slug)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := service.GetProvider(context.Background(), "nonexistent")
		if err != domain.ErrOAuth2ProviderNotFound {
			t.Errorf("GetProvider() error = %v, want ErrOAuth2ProviderNotFound", err)
		}
	})
}

func TestOAuth2Service_CreateApp(t *testing.T) {
	providerRepo := newMockOAuth2ProviderRepo()
	appRepo := newMockOAuth2AppRepo()
	connRepo := newMockOAuth2ConnectionRepo()
	credRepo := newMockCredentialRepo()
	enc := createTestEncryptor(t)

	service := NewOAuth2Service(providerRepo, appRepo, connRepo, credRepo, enc, "http://localhost:8090")

	google := createTestProvider()
	providerRepo.addProvider(google)

	tenantID := uuid.New()

	t.Run("success", func(t *testing.T) {
		app, err := service.CreateApp(context.Background(), CreateAppInput{
			TenantID:     tenantID,
			ProviderSlug: "google",
			ClientID:     "my-client-id",
			ClientSecret: "my-client-secret",
			CustomScopes: []string{"calendar.readonly"},
		})
		if err != nil {
			t.Fatalf("CreateApp() error = %v", err)
		}
		if app.TenantID != tenantID {
			t.Errorf("CreateApp() tenantID = %s, want %s", app.TenantID, tenantID)
		}
		if len(app.CustomScopes) != 1 || app.CustomScopes[0] != "calendar.readonly" {
			t.Errorf("CreateApp() customScopes = %v, want [calendar.readonly]", app.CustomScopes)
		}
	})

	t.Run("provider not found", func(t *testing.T) {
		_, err := service.CreateApp(context.Background(), CreateAppInput{
			TenantID:     tenantID,
			ProviderSlug: "nonexistent",
			ClientID:     "test",
			ClientSecret: "test",
		})
		if err == nil {
			t.Error("CreateApp() expected error for nonexistent provider")
		}
	})

	t.Run("duplicate app", func(t *testing.T) {
		// First app already created in success test
		_, err := service.CreateApp(context.Background(), CreateAppInput{
			TenantID:     tenantID,
			ProviderSlug: "google",
			ClientID:     "another-id",
			ClientSecret: "another-secret",
		})
		if err != domain.ErrOAuth2AppAlreadyExists {
			t.Errorf("CreateApp() error = %v, want ErrOAuth2AppAlreadyExists", err)
		}
	})
}

func TestOAuth2Service_ListApps(t *testing.T) {
	providerRepo := newMockOAuth2ProviderRepo()
	appRepo := newMockOAuth2AppRepo()
	connRepo := newMockOAuth2ConnectionRepo()
	credRepo := newMockCredentialRepo()
	enc := createTestEncryptor(t)

	service := NewOAuth2Service(providerRepo, appRepo, connRepo, credRepo, enc, "http://localhost:8090")

	tenantID := uuid.New()
	otherTenantID := uuid.New()

	google := createTestProvider()
	providerRepo.addProvider(google)

	app1 := createTestApp(tenantID, google.ID, enc)
	app2 := createTestApp(otherTenantID, google.ID, enc)
	appRepo.addApp(app1)
	appRepo.addApp(app2)

	t.Run("returns only tenant apps", func(t *testing.T) {
		apps, err := service.ListApps(context.Background(), tenantID)
		if err != nil {
			t.Fatalf("ListApps() error = %v", err)
		}
		if len(apps) != 1 {
			t.Errorf("ListApps() got %d apps, want 1", len(apps))
		}
		if apps[0].TenantID != tenantID {
			t.Errorf("ListApps() wrong tenant returned")
		}
	})
}

func TestOAuth2Service_DeleteApp(t *testing.T) {
	providerRepo := newMockOAuth2ProviderRepo()
	appRepo := newMockOAuth2AppRepo()
	connRepo := newMockOAuth2ConnectionRepo()
	credRepo := newMockCredentialRepo()
	enc := createTestEncryptor(t)

	service := NewOAuth2Service(providerRepo, appRepo, connRepo, credRepo, enc, "http://localhost:8090")

	google := createTestProvider()
	providerRepo.addProvider(google)

	tenantID := uuid.New()
	app := createTestApp(tenantID, google.ID, enc)
	appRepo.addApp(app)

	t.Run("success", func(t *testing.T) {
		err := service.DeleteApp(context.Background(), app.ID)
		if err != nil {
			t.Fatalf("DeleteApp() error = %v", err)
		}

		// Verify deleted
		_, err = service.GetApp(context.Background(), app.ID)
		if err != domain.ErrOAuth2AppNotFound {
			t.Errorf("GetApp() after delete should return not found")
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := service.DeleteApp(context.Background(), uuid.New())
		if err != domain.ErrOAuth2AppNotFound {
			t.Errorf("DeleteApp() error = %v, want ErrOAuth2AppNotFound", err)
		}
	})
}

func TestOAuth2Service_GetConnection(t *testing.T) {
	providerRepo := newMockOAuth2ProviderRepo()
	appRepo := newMockOAuth2AppRepo()
	connRepo := newMockOAuth2ConnectionRepo()
	credRepo := newMockCredentialRepo()
	enc := createTestEncryptor(t)

	service := NewOAuth2Service(providerRepo, appRepo, connRepo, credRepo, enc, "http://localhost:8090")

	credentialID := uuid.New()
	appID := uuid.New()
	conn := domain.NewOAuth2Connection(credentialID, appID)
	conn.Status = domain.OAuth2ConnectionStatusConnected
	connRepo.addConnection(conn)

	t.Run("success", func(t *testing.T) {
		result, err := service.GetConnection(context.Background(), conn.ID)
		if err != nil {
			t.Fatalf("GetConnection() error = %v", err)
		}
		if result.ID != conn.ID {
			t.Errorf("GetConnection() ID = %s, want %s", result.ID, conn.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := service.GetConnection(context.Background(), uuid.New())
		if err != domain.ErrOAuth2ConnectionNotFound {
			t.Errorf("GetConnection() error = %v, want ErrOAuth2ConnectionNotFound", err)
		}
	})
}

func TestOAuth2Service_GetConnectionByCredential(t *testing.T) {
	providerRepo := newMockOAuth2ProviderRepo()
	appRepo := newMockOAuth2AppRepo()
	connRepo := newMockOAuth2ConnectionRepo()
	credRepo := newMockCredentialRepo()
	enc := createTestEncryptor(t)

	service := NewOAuth2Service(providerRepo, appRepo, connRepo, credRepo, enc, "http://localhost:8090")

	credentialID := uuid.New()
	appID := uuid.New()
	conn := domain.NewOAuth2Connection(credentialID, appID)
	connRepo.addConnection(conn)

	t.Run("success", func(t *testing.T) {
		result, err := service.GetConnectionByCredential(context.Background(), credentialID)
		if err != nil {
			t.Fatalf("GetConnectionByCredential() error = %v", err)
		}
		if result.CredentialID != credentialID {
			t.Errorf("GetConnectionByCredential() credentialID = %s, want %s", result.CredentialID, credentialID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := service.GetConnectionByCredential(context.Background(), uuid.New())
		if err != domain.ErrOAuth2ConnectionNotFound {
			t.Errorf("GetConnectionByCredential() error = %v, want ErrOAuth2ConnectionNotFound", err)
		}
	})
}

func TestOAuth2Service_ListConnectionsByApp(t *testing.T) {
	providerRepo := newMockOAuth2ProviderRepo()
	appRepo := newMockOAuth2AppRepo()
	connRepo := newMockOAuth2ConnectionRepo()
	credRepo := newMockCredentialRepo()
	enc := createTestEncryptor(t)

	service := NewOAuth2Service(providerRepo, appRepo, connRepo, credRepo, enc, "http://localhost:8090")

	appID := uuid.New()
	otherAppID := uuid.New()

	conn1 := domain.NewOAuth2Connection(uuid.New(), appID)
	conn2 := domain.NewOAuth2Connection(uuid.New(), appID)
	conn3 := domain.NewOAuth2Connection(uuid.New(), otherAppID)
	connRepo.addConnection(conn1)
	connRepo.addConnection(conn2)
	connRepo.addConnection(conn3)

	t.Run("returns only app connections", func(t *testing.T) {
		conns, err := service.ListConnectionsByApp(context.Background(), appID)
		if err != nil {
			t.Fatalf("ListConnectionsByApp() error = %v", err)
		}
		if len(conns) != 2 {
			t.Errorf("ListConnectionsByApp() got %d connections, want 2", len(conns))
		}
	})
}

// ============================================================================
// Response Conversion Tests
// ============================================================================

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

	if challenge == "" {
		t.Error("generateCodeChallenge returned empty string")
	}

	// Same verifier should produce same challenge (deterministic)
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
		ID:               uuid.New(),
		Slug:             "google",
		Name:             "Google",
		PKCERequired:     true,
		DefaultScopes:    []string{"openid", "email"},
		IconURL:          "https://example.com/icon.png",
		DocumentationURL: "https://docs.example.com",
		IsPreset:         true,
	}

	resp := ToProviderResponse(provider)

	if resp.ID != provider.ID {
		t.Errorf("ID = %s, want %s", resp.ID, provider.ID)
	}
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
	if resp.IconURL != provider.IconURL {
		t.Errorf("IconURL = %s, want %s", resp.IconURL, provider.IconURL)
	}
	if resp.DocumentationURL != provider.DocumentationURL {
		t.Errorf("DocumentationURL = %s, want %s", resp.DocumentationURL, provider.DocumentationURL)
	}
	if resp.IsPreset != provider.IsPreset {
		t.Errorf("IsPreset = %v, want %v", resp.IsPreset, provider.IsPreset)
	}
}

func TestToProviderResponses(t *testing.T) {
	providers := []*domain.OAuth2Provider{
		{ID: uuid.New(), Slug: "google", Name: "Google"},
		{ID: uuid.New(), Slug: "github", Name: "GitHub"},
		{ID: uuid.New(), Slug: "slack", Name: "Slack"},
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

func TestToAppResponse(t *testing.T) {
	provider := createTestProvider()
	app := &domain.OAuth2App{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		ProviderID:   provider.ID,
		CustomScopes: []string{"custom.scope"},
		Status:       domain.OAuth2AppStatusActive,
		CreatedAt:    time.Now(),
		Provider:     provider,
	}

	resp := ToAppResponse(app)

	if resp.ID != app.ID {
		t.Errorf("ID = %s, want %s", resp.ID, app.ID)
	}
	if resp.TenantID != app.TenantID {
		t.Errorf("TenantID = %s, want %s", resp.TenantID, app.TenantID)
	}
	if resp.ProviderSlug != provider.Slug {
		t.Errorf("ProviderSlug = %s, want %s", resp.ProviderSlug, provider.Slug)
	}
	if resp.ProviderName != provider.Name {
		t.Errorf("ProviderName = %s, want %s", resp.ProviderName, provider.Name)
	}
	if resp.Status != app.Status {
		t.Errorf("Status = %s, want %s", resp.Status, app.Status)
	}
	if resp.Provider == nil {
		t.Error("Provider should not be nil")
	}
}

func TestToConnectionResponse(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(time.Hour)
	lastRefresh := now.Add(-5 * time.Minute)
	lastUsed := now.Add(-1 * time.Minute)

	conn := &domain.OAuth2Connection{
		ID:                   uuid.New(),
		CredentialID:         uuid.New(),
		OAuth2AppID:          uuid.New(),
		Status:               domain.OAuth2ConnectionStatusConnected,
		AccountID:            "12345",
		AccountEmail:         "test@example.com",
		AccountName:          "Test User",
		AccessTokenExpiresAt: &expiresAt,
		LastRefreshAt:        &lastRefresh,
		LastUsedAt:           &lastUsed,
		CreatedAt:            now,
	}

	resp := ToConnectionResponse(conn)

	if resp.ID != conn.ID {
		t.Errorf("ID = %s, want %s", resp.ID, conn.ID)
	}
	if resp.CredentialID != conn.CredentialID {
		t.Errorf("CredentialID = %s, want %s", resp.CredentialID, conn.CredentialID)
	}
	if resp.Status != conn.Status {
		t.Errorf("Status = %s, want %s", resp.Status, conn.Status)
	}
	if resp.AccountID != conn.AccountID {
		t.Errorf("AccountID = %s, want %s", resp.AccountID, conn.AccountID)
	}
	if resp.AccountEmail != conn.AccountEmail {
		t.Errorf("AccountEmail = %s, want %s", resp.AccountEmail, conn.AccountEmail)
	}
	if resp.AccountName != conn.AccountName {
		t.Errorf("AccountName = %s, want %s", resp.AccountName, conn.AccountName)
	}
}

func TestToConnectionResponses(t *testing.T) {
	conns := []*domain.OAuth2Connection{
		{ID: uuid.New(), Status: domain.OAuth2ConnectionStatusPending},
		{ID: uuid.New(), Status: domain.OAuth2ConnectionStatusConnected},
		{ID: uuid.New(), Status: domain.OAuth2ConnectionStatusExpired},
	}

	responses := ToConnectionResponses(conns)

	if len(responses) != len(conns) {
		t.Errorf("Response length = %d, want %d", len(responses), len(conns))
	}

	for i, resp := range responses {
		if resp.ID != conns[i].ID {
			t.Errorf("Response[%d].ID = %s, want %s", i, resp.ID, conns[i].ID)
		}
		if resp.Status != conns[i].Status {
			t.Errorf("Response[%d].Status = %s, want %s", i, resp.Status, conns[i].Status)
		}
	}
}
