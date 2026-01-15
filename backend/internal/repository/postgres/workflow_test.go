package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowRepository_Create(t *testing.T) {
	tests := []struct {
		name      string
		workflow  *domain.Workflow
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "successful creation",
			workflow: &domain.Workflow{
				ID:          uuid.New(),
				TenantID:    uuid.New(),
				Name:        "Test Workflow",
				Description: "Test Description",
				Status:      domain.WorkflowStatusDraft,
				Version:     1,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO workflows").
					WithArgs(
						pgxmock.AnyArg(), pgxmock.AnyArg(), "Test Workflow", "Test Description",
						domain.WorkflowStatusDraft, 1, pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			workflow: &domain.Workflow{
				ID:       uuid.New(),
				TenantID: uuid.New(),
				Name:     "Test",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO workflows").
					WithArgs(
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(),
					).
					WillReturnError(errors.New("connection error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewWorkflowRepositoryWithDB(mock)
			err = repo.Create(context.Background(), tt.workflow)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWorkflowRepository_GetByID(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	createdBy := uuid.New()
	now := time.Now()

	tests := []struct {
		name      string
		tenantID  uuid.UUID
		id        uuid.UUID
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
		want      *domain.Workflow
	}{
		{
			name:     "workflow found",
			tenantID: tenantID,
			id:       workflowID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "tenant_id", "name", "description", "status", "version",
					"input_schema", "output_schema", "draft", "created_by", "published_at",
					"created_at", "updated_at", "deleted_at", "is_system", "system_slug",
				}).AddRow(
					workflowID, tenantID, "Test Workflow", "Description",
					domain.WorkflowStatusDraft, 1, json.RawMessage(`{}`), json.RawMessage(`{}`),
					json.RawMessage(`null`), &createdBy, nil, now, now, nil, false, nil,
				)
				mock.ExpectQuery("SELECT .+ FROM workflows").
					WithArgs(workflowID, tenantID).
					WillReturnRows(rows)
			},
			wantErr: nil,
			want: &domain.Workflow{
				ID:          workflowID,
				TenantID:    tenantID,
				Name:        "Test Workflow",
				Description: "Description",
				Status:      domain.WorkflowStatusDraft,
				Version:     1,
			},
		},
		{
			name:     "workflow not found",
			tenantID: tenantID,
			id:       workflowID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .+ FROM workflows").
					WithArgs(workflowID, tenantID).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: domain.ErrWorkflowNotFound,
			want:    nil,
		},
		{
			name:     "database error",
			tenantID: tenantID,
			id:       workflowID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .+ FROM workflows").
					WithArgs(workflowID, tenantID).
					WillReturnError(errors.New("database connection error"))
			},
			wantErr: errors.New("database connection error"),
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewWorkflowRepositoryWithDB(mock)
			result, err := repo.GetByID(context.Background(), tt.tenantID, tt.id)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrWorkflowNotFound) {
					assert.True(t, errors.Is(err, domain.ErrWorkflowNotFound))
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.ID, result.ID)
				assert.Equal(t, tt.want.TenantID, result.TenantID)
				assert.Equal(t, tt.want.Name, result.Name)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWorkflowRepository_List(t *testing.T) {
	tenantID := uuid.New()
	workflowID1 := uuid.New()
	workflowID2 := uuid.New()
	createdBy1 := uuid.New()
	createdBy2 := uuid.New()
	now := time.Now()
	draftStatus := domain.WorkflowStatusDraft

	tests := []struct {
		name      string
		tenantID  uuid.UUID
		filter    repository.WorkflowFilter
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantCount int
		wantTotal int
		wantErr   bool
	}{
		{
			name:     "list all workflows",
			tenantID: tenantID,
			filter:   repository.WorkflowFilter{Page: 1, Limit: 10},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				// Count query
				countRows := pgxmock.NewRows([]string{"count"}).AddRow(2)
				mock.ExpectQuery("SELECT COUNT").
					WithArgs(tenantID).
					WillReturnRows(countRows)

				// List query
				rows := pgxmock.NewRows([]string{
					"id", "tenant_id", "name", "description", "status", "version",
					"input_schema", "output_schema", "draft", "created_by", "published_at",
					"created_at", "updated_at", "deleted_at", "is_system", "system_slug",
				}).
					AddRow(workflowID1, tenantID, "Workflow 1", "Desc 1", domain.WorkflowStatusDraft, 1,
						json.RawMessage(`{}`), json.RawMessage(`{}`), json.RawMessage(`null`),
						&createdBy1, nil, now, now, nil, false, nil).
					AddRow(workflowID2, tenantID, "Workflow 2", "Desc 2", domain.WorkflowStatusPublished, 2,
						json.RawMessage(`{}`), json.RawMessage(`{}`), json.RawMessage(`null`),
						&createdBy2, &now, now, now, nil, false, nil)
				mock.ExpectQuery("SELECT .+ FROM workflows").
					WithArgs(tenantID, 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 2,
			wantTotal: 2,
			wantErr:   false,
		},
		{
			name:     "filter by status",
			tenantID: tenantID,
			filter:   repository.WorkflowFilter{Page: 1, Limit: 10, Status: &draftStatus},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				// Count query with status filter
				countRows := pgxmock.NewRows([]string{"count"}).AddRow(1)
				mock.ExpectQuery("SELECT COUNT").
					WithArgs(tenantID, draftStatus).
					WillReturnRows(countRows)

				// List query with status filter
				rows := pgxmock.NewRows([]string{
					"id", "tenant_id", "name", "description", "status", "version",
					"input_schema", "output_schema", "draft", "created_by", "published_at",
					"created_at", "updated_at", "deleted_at", "is_system", "system_slug",
				}).AddRow(workflowID1, tenantID, "Workflow 1", "Desc 1", domain.WorkflowStatusDraft, 1,
					json.RawMessage(`{}`), json.RawMessage(`{}`), json.RawMessage(`null`),
					&createdBy1, nil, now, now, nil, false, nil)
				mock.ExpectQuery("SELECT .+ FROM workflows").
					WithArgs(tenantID, draftStatus, 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 1,
			wantTotal: 1,
			wantErr:   false,
		},
		{
			name:     "count query error",
			tenantID: tenantID,
			filter:   repository.WorkflowFilter{Page: 1, Limit: 10},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT COUNT").
					WithArgs(tenantID).
					WillReturnError(errors.New("count error"))
			},
			wantCount: 0,
			wantTotal: 0,
			wantErr:   true,
		},
		{
			name:     "list query error",
			tenantID: tenantID,
			filter:   repository.WorkflowFilter{Page: 1, Limit: 10},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				countRows := pgxmock.NewRows([]string{"count"}).AddRow(2)
				mock.ExpectQuery("SELECT COUNT").
					WithArgs(tenantID).
					WillReturnRows(countRows)

				mock.ExpectQuery("SELECT .+ FROM workflows").
					WithArgs(tenantID, 10, 0).
					WillReturnError(errors.New("list error"))
			},
			wantCount: 0,
			wantTotal: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewWorkflowRepositoryWithDB(mock)
			workflows, total, err := repo.List(context.Background(), tt.tenantID, tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, workflows, tt.wantCount)
				assert.Equal(t, tt.wantTotal, total)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWorkflowRepository_Update(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()

	tests := []struct {
		name      string
		workflow  *domain.Workflow
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name: "successful update",
			workflow: &domain.Workflow{
				ID:          workflowID,
				TenantID:    tenantID,
				Name:        "Updated Workflow",
				Description: "Updated Description",
				Status:      domain.WorkflowStatusPublished,
				Version:     2,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE workflows").
					WithArgs(
						"Updated Workflow", "Updated Description", domain.WorkflowStatusPublished, 2,
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						workflowID, tenantID,
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: nil,
		},
		{
			name: "workflow not found",
			workflow: &domain.Workflow{
				ID:       workflowID,
				TenantID: tenantID,
				Name:     "Not Found",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE workflows").
					WithArgs(
						"Not Found", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						workflowID, tenantID,
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			wantErr: domain.ErrWorkflowNotFound,
		},
		{
			name: "database error",
			workflow: &domain.Workflow{
				ID:       workflowID,
				TenantID: tenantID,
				Name:     "Error",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE workflows").
					WithArgs(
						"Error", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						workflowID, tenantID,
					).
					WillReturnError(errors.New("connection error"))
			},
			wantErr: errors.New("connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewWorkflowRepositoryWithDB(mock)
			err = repo.Update(context.Background(), tt.workflow)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrWorkflowNotFound) {
					assert.True(t, errors.Is(err, domain.ErrWorkflowNotFound))
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWorkflowRepository_Delete(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()

	tests := []struct {
		name      string
		tenantID  uuid.UUID
		id        uuid.UUID
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name:     "successful delete",
			tenantID: tenantID,
			id:       workflowID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE workflows SET deleted_at").
					WithArgs(pgxmock.AnyArg(), workflowID, tenantID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: nil,
		},
		{
			name:     "workflow not found",
			tenantID: tenantID,
			id:       workflowID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE workflows SET deleted_at").
					WithArgs(pgxmock.AnyArg(), workflowID, tenantID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			wantErr: domain.ErrWorkflowNotFound,
		},
		{
			name:     "database error",
			tenantID: tenantID,
			id:       workflowID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE workflows SET deleted_at").
					WithArgs(pgxmock.AnyArg(), workflowID, tenantID).
					WillReturnError(errors.New("connection error"))
			},
			wantErr: errors.New("connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewWorkflowRepositoryWithDB(mock)
			err = repo.Delete(context.Background(), tt.tenantID, tt.id)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrWorkflowNotFound) {
					assert.True(t, errors.Is(err, domain.ErrWorkflowNotFound))
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWorkflowRepository_GetSystemBySlug(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	createdBy := uuid.New()
	now := time.Now()
	slug := "system-workflow"

	tests := []struct {
		name      string
		slug      string
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
		want      *domain.Workflow
	}{
		{
			name: "system workflow found",
			slug: slug,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "tenant_id", "name", "description", "status", "version",
					"input_schema", "output_schema", "draft", "created_by", "published_at",
					"created_at", "updated_at", "deleted_at", "is_system", "system_slug",
				}).AddRow(
					workflowID, tenantID, "System Workflow", "System Description",
					domain.WorkflowStatusPublished, 1, json.RawMessage(`{}`), json.RawMessage(`{}`),
					json.RawMessage(`null`), &createdBy, &now, now, now, nil, true, &slug,
				)
				mock.ExpectQuery("SELECT .+ FROM workflows").
					WithArgs(slug).
					WillReturnRows(rows)
			},
			wantErr: nil,
			want: &domain.Workflow{
				ID:         workflowID,
				Name:       "System Workflow",
				IsSystem:   true,
				SystemSlug: &slug,
			},
		},
		{
			name: "system workflow not found",
			slug: "non-existent",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .+ FROM workflows").
					WithArgs("non-existent").
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: domain.ErrWorkflowNotFound,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewWorkflowRepositoryWithDB(mock)
			result, err := repo.GetSystemBySlug(context.Background(), tt.slug)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrWorkflowNotFound) {
					assert.True(t, errors.Is(err, domain.ErrWorkflowNotFound))
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.ID, result.ID)
				assert.Equal(t, tt.want.Name, result.Name)
				assert.Equal(t, tt.want.IsSystem, result.IsSystem)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWorkflowRepository_GetWithStepsAndEdges(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	stepID1 := uuid.New()
	stepID2 := uuid.New()
	edgeID := uuid.New()
	createdBy := uuid.New()
	now := time.Now()

	tests := []struct {
		name      string
		tenantID  uuid.UUID
		id        uuid.UUID
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
		wantSteps int
		wantEdges int
	}{
		{
			name:     "workflow with steps and edges",
			tenantID: tenantID,
			id:       workflowID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				// GetByID query
				workflowRows := pgxmock.NewRows([]string{
					"id", "tenant_id", "name", "description", "status", "version",
					"input_schema", "output_schema", "draft", "created_by", "published_at",
					"created_at", "updated_at", "deleted_at", "is_system", "system_slug",
				}).AddRow(
					workflowID, tenantID, "Test Workflow", "Description",
					domain.WorkflowStatusDraft, 1, json.RawMessage(`{}`), json.RawMessage(`{}`),
					json.RawMessage(`null`), &createdBy, nil, now, now, nil, false, nil,
				)
				mock.ExpectQuery("SELECT .+ FROM workflows WHERE id").
					WithArgs(workflowID, tenantID).
					WillReturnRows(workflowRows)

				// Steps query
				stepsRows := pgxmock.NewRows([]string{
					"id", "workflow_id", "name", "type", "config", "block_group_id", "group_role",
					"position_x", "position_y", "created_at", "updated_at",
				}).
					AddRow(stepID1, workflowID, "Step 1", "start", json.RawMessage(`{}`), nil, nil, 0.0, 0.0, now, now).
					AddRow(stepID2, workflowID, "Step 2", "tool", json.RawMessage(`{"adapter_id":"mock"}`), nil, nil, 100.0, 100.0, now, now)
				mock.ExpectQuery("SELECT .+ FROM steps WHERE workflow_id").
					WithArgs(workflowID).
					WillReturnRows(stepsRows)

				// Edges query
				edgesRows := pgxmock.NewRows([]string{
					"id", "workflow_id", "source_step_id", "target_step_id", "source_block_group_id", "target_block_group_id", "source_port", "target_port", "condition", "created_at",
				}).AddRow(edgeID, workflowID, &stepID1, &stepID2, nil, nil, "default", "input", nil, now)
				mock.ExpectQuery("SELECT .+ FROM edges WHERE workflow_id").
					WithArgs(workflowID).
					WillReturnRows(edgesRows)
			},
			wantErr:   nil,
			wantSteps: 2,
			wantEdges: 1,
		},
		{
			name:     "workflow not found",
			tenantID: tenantID,
			id:       workflowID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .+ FROM workflows WHERE id").
					WithArgs(workflowID, tenantID).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr:   domain.ErrWorkflowNotFound,
			wantSteps: 0,
			wantEdges: 0,
		},
		{
			name:     "steps query error",
			tenantID: tenantID,
			id:       workflowID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				workflowRows := pgxmock.NewRows([]string{
					"id", "tenant_id", "name", "description", "status", "version",
					"input_schema", "output_schema", "draft", "created_by", "published_at",
					"created_at", "updated_at", "deleted_at", "is_system", "system_slug",
				}).AddRow(
					workflowID, tenantID, "Test Workflow", "Description",
					domain.WorkflowStatusDraft, 1, json.RawMessage(`{}`), json.RawMessage(`{}`),
					json.RawMessage(`null`), &createdBy, nil, now, now, nil, false, nil,
				)
				mock.ExpectQuery("SELECT .+ FROM workflows WHERE id").
					WithArgs(workflowID, tenantID).
					WillReturnRows(workflowRows)

				mock.ExpectQuery("SELECT .+ FROM steps WHERE workflow_id").
					WithArgs(workflowID).
					WillReturnError(errors.New("steps query error"))
			},
			wantErr:   errors.New("steps query error"),
			wantSteps: 0,
			wantEdges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewWorkflowRepositoryWithDB(mock)
			result, err := repo.GetWithStepsAndEdges(context.Background(), tt.tenantID, tt.id)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrWorkflowNotFound) {
					assert.True(t, errors.Is(err, domain.ErrWorkflowNotFound))
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, result.Steps, tt.wantSteps)
				assert.Len(t, result.Edges, tt.wantEdges)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNewWorkflowRepository(t *testing.T) {
	// Test that the repository is created correctly
	// We can't test with an actual pool here, so we just verify the function signature works
	repo := NewWorkflowRepositoryWithDB(nil)
	assert.NotNil(t, repo)
}
