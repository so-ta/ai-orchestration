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
