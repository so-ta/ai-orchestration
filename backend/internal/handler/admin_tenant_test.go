package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTenantRepository is a mock implementation of repository.TenantRepository
type mockTenantRepository struct {
	getByIDFunc    func(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	getBySlugFunc  func(ctx context.Context, slug string) (*domain.Tenant, error)
	createFunc     func(ctx context.Context, tenant *domain.Tenant) error
	updateFunc     func(ctx context.Context, tenant *domain.Tenant) error
	deleteFunc     func(ctx context.Context, id uuid.UUID) error
	listFunc       func(ctx context.Context, filter repository.TenantFilter) ([]*domain.Tenant, int, error)
	updateStatusFunc func(ctx context.Context, id uuid.UUID, status domain.TenantStatus, reason string) error
	getStatsFunc   func(ctx context.Context, id uuid.UUID) (*domain.TenantStats, error)
	getAllStatsFunc func(ctx context.Context) (map[uuid.UUID]*domain.TenantStats, error)
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockTenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	if m.getBySlugFunc != nil {
		return m.getBySlugFunc(ctx, slug)
	}
	return nil, nil
}

func (m *mockTenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, tenant)
	}
	return nil
}

func (m *mockTenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, tenant)
	}
	return nil
}

func (m *mockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockTenantRepository) List(ctx context.Context, filter repository.TenantFilter) ([]*domain.Tenant, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockTenantRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TenantStatus, reason string) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, id, status, reason)
	}
	return nil
}

func (m *mockTenantRepository) GetStats(ctx context.Context, id uuid.UUID) (*domain.TenantStats, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx, id)
	}
	return &domain.TenantStats{}, nil
}

func (m *mockTenantRepository) GetAllStats(ctx context.Context) (map[uuid.UUID]*domain.TenantStats, error) {
	if m.getAllStatsFunc != nil {
		return m.getAllStatsFunc(ctx)
	}
	return map[uuid.UUID]*domain.TenantStats{}, nil
}

// TestAdminTenantHandler_Create_GetBySlugDBError tests Create handler when GetBySlug returns a DB error
func TestAdminTenantHandler_Create_GetBySlugDBError(t *testing.T) {
	dbError := errors.New("database connection failed")

	mockRepo := &mockTenantRepository{
		getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
			return nil, dbError
		},
	}

	handler := NewAdminTenantHandler(mockRepo)

	body := `{"name": "Test Tenant", "slug": "test-tenant"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tenants", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
}

// TestAdminTenantHandler_Create_GetBySlugNotFound tests Create handler when tenant doesn't exist (success case)
func TestAdminTenantHandler_Create_GetBySlugNotFound(t *testing.T) {
	mockRepo := &mockTenantRepository{
		getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
			return nil, domain.ErrTenantNotFound
		},
		createFunc: func(ctx context.Context, tenant *domain.Tenant) error {
			return nil
		},
	}

	handler := NewAdminTenantHandler(mockRepo)

	body := `{"name": "Test Tenant", "slug": "test-tenant"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tenants", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

// TestAdminTenantHandler_Update_GetBySlugDBError tests Update handler when GetBySlug returns a DB error
func TestAdminTenantHandler_Update_GetBySlugDBError(t *testing.T) {
	tenantID := uuid.New()
	existingTenant := &domain.Tenant{
		ID:   tenantID,
		Name: "Existing Tenant",
		Slug: "existing-tenant",
	}
	dbError := errors.New("database connection failed")

	mockRepo := &mockTenantRepository{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
			return existingTenant, nil
		},
		getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
			return nil, dbError
		},
	}

	handler := NewAdminTenantHandler(mockRepo)

	body := `{"slug": "new-slug"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/tenants/"+tenantID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	// Set chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", tenantID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
}

// TestAdminTenantHandler_Get_GetStatsError tests Get handler when GetStats returns an error
func TestAdminTenantHandler_Get_GetStatsError(t *testing.T) {
	tenantID := uuid.New()
	existingTenant := &domain.Tenant{
		ID:     tenantID,
		Name:   "Test Tenant",
		Slug:   "test-tenant",
		Status: domain.TenantStatusActive,
		Plan:   domain.TenantPlanFree,
	}

	mockRepo := &mockTenantRepository{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
			return existingTenant, nil
		},
		getStatsFunc: func(ctx context.Context, id uuid.UUID) (*domain.TenantStats, error) {
			return nil, errors.New("stats query failed")
		},
	}

	handler := NewAdminTenantHandler(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants/"+tenantID.String(), nil)

	// Set chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", tenantID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.Get(rec, req)

	// Should still return 200 OK because GetStats error is logged but doesn't fail the request
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, tenantID.String(), resp["id"])
}

// TestAdminTenantHandler_Create tests Create handler with various scenarios
func TestAdminTenantHandler_Create(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		mockRepo         *mockTenantRepository
		expectedStatus   int
		expectedCode     string
		validateResponse func(t *testing.T, body []byte)
	}{
		{
			name: "success - creates tenant with minimal fields",
			body: `{"name": "Test Tenant", "slug": "test-tenant"}`,
			mockRepo: &mockTenantRepository{
				getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
					return nil, domain.ErrTenantNotFound
				},
				createFunc: func(ctx context.Context, tenant *domain.Tenant) error {
					return nil
				},
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "Test Tenant", resp["name"])
				assert.Equal(t, "test-tenant", resp["slug"])
				assert.Equal(t, "free", resp["plan"]) // Default plan
				assert.Equal(t, "active", resp["status"])
			},
		},
		{
			name: "success - creates tenant with all fields",
			body: `{
				"name": "Full Tenant",
				"slug": "full-tenant",
				"plan": "professional",
				"owner_email": "owner@example.com",
				"owner_name": "Test Owner",
				"billing_email": "billing@example.com"
			}`,
			mockRepo: &mockTenantRepository{
				getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
					return nil, domain.ErrTenantNotFound
				},
				createFunc: func(ctx context.Context, tenant *domain.Tenant) error {
					return nil
				},
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "Full Tenant", resp["name"])
				assert.Equal(t, "full-tenant", resp["slug"])
				assert.Equal(t, "professional", resp["plan"])
				assert.Equal(t, "owner@example.com", resp["owner_email"])
				assert.Equal(t, "Test Owner", resp["owner_name"])
				assert.Equal(t, "billing@example.com", resp["billing_email"])
			},
		},
		{
			name: "success - creates tenant with custom feature_flags",
			body: `{
				"name": "Custom Tenant",
				"slug": "custom-tenant",
				"feature_flags": {
					"api_access": true,
					"custom_blocks": false,
					"copilot_enabled": true
				}
			}`,
			mockRepo: &mockTenantRepository{
				getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
					return nil, domain.ErrTenantNotFound
				},
				createFunc: func(ctx context.Context, tenant *domain.Tenant) error {
					return nil
				},
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "Custom Tenant", resp["name"])
				// feature_flags should be present in response
				assert.NotNil(t, resp["feature_flags"])
			},
		},
		{
			name: "success - creates tenant with custom limits",
			body: `{
				"name": "Limited Tenant",
				"slug": "limited-tenant",
				"limits": {
					"max_workflows": 100,
					"max_runs_per_day": 1000,
					"max_users": 50,
					"max_storage_mb": 10240
				}
			}`,
			mockRepo: &mockTenantRepository{
				getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
					return nil, domain.ErrTenantNotFound
				},
				createFunc: func(ctx context.Context, tenant *domain.Tenant) error {
					return nil
				},
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "Limited Tenant", resp["name"])
				assert.NotNil(t, resp["limits"])
			},
		},
		{
			name: "success - creates tenant with custom metadata",
			body: `{
				"name": "Metadata Tenant",
				"slug": "metadata-tenant",
				"metadata": {
					"industry": "technology",
					"company_size": "50-200",
					"website": "https://example.com",
					"country": "Japan",
					"notes": "Test tenant for API"
				}
			}`,
			mockRepo: &mockTenantRepository{
				getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
					return nil, domain.ErrTenantNotFound
				},
				createFunc: func(ctx context.Context, tenant *domain.Tenant) error {
					return nil
				},
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "Metadata Tenant", resp["name"])
				assert.NotNil(t, resp["metadata"])
			},
		},
		{
			name: "success - default settings is empty JSON object",
			body: `{"name": "Settings Tenant", "slug": "settings-tenant"}`,
			mockRepo: &mockTenantRepository{
				getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
					return nil, domain.ErrTenantNotFound
				},
				createFunc: func(ctx context.Context, tenant *domain.Tenant) error {
					return nil
				},
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "Settings Tenant", resp["name"])
				// settings should be present with default empty JSON
				assert.NotNil(t, resp["settings"])
			},
		},
		{
			name:           "error - invalid JSON body",
			body:           `{invalid json`,
			mockRepo:       &mockTenantRepository{},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_JSON",
		},
		{
			name:           "error - missing name",
			body:           `{"slug": "test-tenant"}`,
			mockRepo:       &mockTenantRepository{},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "MISSING_NAME",
		},
		{
			name:           "error - missing slug",
			body:           `{"name": "Test Tenant"}`,
			mockRepo:       &mockTenantRepository{},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "MISSING_SLUG",
		},
		{
			name:           "error - invalid plan",
			body:           `{"name": "Test Tenant", "slug": "test-tenant", "plan": "invalid-plan"}`,
			mockRepo:       &mockTenantRepository{},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_PLAN",
		},
		{
			name: "error - slug already exists",
			body: `{"name": "Test Tenant", "slug": "existing-slug"}`,
			mockRepo: &mockTenantRepository{
				getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
					return &domain.Tenant{
						ID:   uuid.New(),
						Name: "Existing Tenant",
						Slug: "existing-slug",
					}, nil
				},
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "SLUG_EXISTS",
		},
		{
			name: "error - Create repository error",
			body: `{"name": "Test Tenant", "slug": "test-tenant"}`,
			mockRepo: &mockTenantRepository{
				getBySlugFunc: func(ctx context.Context, slug string) (*domain.Tenant, error) {
					return nil, domain.ErrTenantNotFound
				},
				createFunc: func(ctx context.Context, tenant *domain.Tenant) error {
					return errors.New("insert failed")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAdminTenantHandler(tt.mockRepo)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tenants", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Create(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedCode != "" {
				var resp ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp.Error.Code)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, rec.Body.Bytes())
			}
		})
	}
}

// TestAdminTenantHandler_Update_GetStatsError tests Update handler when GetStats returns an error
func TestAdminTenantHandler_Update_GetStatsError(t *testing.T) {
	tenantID := uuid.New()
	existingTenant := &domain.Tenant{
		ID:     tenantID,
		Name:   "Test Tenant",
		Slug:   "test-tenant",
		Status: domain.TenantStatusActive,
		Plan:   domain.TenantPlanFree,
	}

	mockRepo := &mockTenantRepository{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
			return existingTenant, nil
		},
		updateFunc: func(ctx context.Context, tenant *domain.Tenant) error {
			return nil
		},
		getStatsFunc: func(ctx context.Context, id uuid.UUID) (*domain.TenantStats, error) {
			return nil, errors.New("stats query failed")
		},
	}

	handler := NewAdminTenantHandler(mockRepo)

	body := `{"name": "Updated Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/tenants/"+tenantID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	// Set chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", tenantID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	// Should still return 200 OK because GetStats error is logged but doesn't fail the request
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", resp["name"])
}
