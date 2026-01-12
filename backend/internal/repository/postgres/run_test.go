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

func TestRunRepository_Create(t *testing.T) {
	tests := []struct {
		name      string
		run       *domain.Run
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "successful creation",
			run: &domain.Run{
				ID:              uuid.New(),
				TenantID:        uuid.New(),
				WorkflowID:      uuid.New(),
				WorkflowVersion: 1,
				Status:          domain.RunStatusPending,
				Mode:            domain.RunModeTest,
				Input:           json.RawMessage(`{}`),
				CreatedAt:       time.Now(),
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO runs").
					WithArgs(
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			run: &domain.Run{
				ID:       uuid.New(),
				TenantID: uuid.New(),
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("INSERT INTO runs").
					WithArgs(
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
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

			repo := NewRunRepositoryWithDB(mock)
			err = repo.Create(context.Background(), tt.run)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRunRepository_GetByID(t *testing.T) {
	tenantID := uuid.New()
	runID := uuid.New()
	workflowID := uuid.New()
	now := time.Now()

	tests := []struct {
		name      string
		tenantID  uuid.UUID
		id        uuid.UUID
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name:     "run found",
			tenantID: tenantID,
			id:       runID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "tenant_id", "workflow_id", "workflow_version", "status", "mode",
					"input", "output", "error", "triggered_by", "triggered_by_user",
					"started_at", "completed_at", "created_at", "trigger_source", "trigger_metadata",
				}).AddRow(
					runID, tenantID, workflowID, 1, domain.RunStatusPending, domain.RunModeTest,
					json.RawMessage(`{}`), nil, nil, "manual", nil,
					nil, nil, now, nil, nil,
				)
				mock.ExpectQuery("SELECT .+ FROM runs").
					WithArgs(runID, tenantID).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name:     "run not found",
			tenantID: tenantID,
			id:       runID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .+ FROM runs").
					WithArgs(runID, tenantID).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: domain.ErrRunNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewRunRepositoryWithDB(mock)
			_, err = repo.GetByID(context.Background(), tt.tenantID, tt.id)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRunRepository_ListByWorkflow(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	runID1 := uuid.New()
	runID2 := uuid.New()
	now := time.Now()

	tests := []struct {
		name      string
		tenantID  uuid.UUID
		workflowID uuid.UUID
		filter    repository.RunFilter
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantCount int
		wantTotal int
		wantErr   bool
	}{
		{
			name:       "list runs",
			tenantID:   tenantID,
			workflowID: workflowID,
			filter:     repository.RunFilter{Page: 1, Limit: 10},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				countRows := pgxmock.NewRows([]string{"count"}).AddRow(2)
				mock.ExpectQuery("SELECT COUNT").
					WithArgs(tenantID, workflowID).
					WillReturnRows(countRows)

				rows := pgxmock.NewRows([]string{
					"id", "tenant_id", "workflow_id", "workflow_version", "status", "mode",
					"input", "output", "error", "triggered_by", "triggered_by_user",
					"started_at", "completed_at", "created_at", "trigger_source", "trigger_metadata",
				}).
					AddRow(runID1, tenantID, workflowID, 1, domain.RunStatusCompleted, domain.RunModeTest,
						json.RawMessage(`{}`), nil, nil, "manual", nil, nil, nil, now, nil, nil).
					AddRow(runID2, tenantID, workflowID, 1, domain.RunStatusPending, domain.RunModeTest,
						json.RawMessage(`{}`), nil, nil, "manual", nil, nil, nil, now, nil, nil)
				mock.ExpectQuery("SELECT .+ FROM runs").
					WithArgs(tenantID, workflowID, 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 2,
			wantTotal: 2,
			wantErr:   false,
		},
		{
			name:       "count error",
			tenantID:   tenantID,
			workflowID: workflowID,
			filter:     repository.RunFilter{Page: 1, Limit: 10},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT COUNT").
					WithArgs(tenantID, workflowID).
					WillReturnError(errors.New("count error"))
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

			repo := NewRunRepositoryWithDB(mock)
			runs, total, err := repo.ListByWorkflow(context.Background(), tt.tenantID, tt.workflowID, tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, runs, tt.wantCount)
				assert.Equal(t, tt.wantTotal, total)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRunRepository_Update(t *testing.T) {
	tenantID := uuid.New()
	runID := uuid.New()

	tests := []struct {
		name      string
		run       *domain.Run
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name: "successful update",
			run: &domain.Run{
				ID:       runID,
				TenantID: tenantID,
				Status:   domain.RunStatusCompleted,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE runs").
					WithArgs(
						domain.RunStatusCompleted, pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), runID, tenantID,
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: nil,
		},
		{
			name: "run not found",
			run: &domain.Run{
				ID:       runID,
				TenantID: tenantID,
				Status:   domain.RunStatusCompleted,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE runs").
					WithArgs(
						domain.RunStatusCompleted, pgxmock.AnyArg(), pgxmock.AnyArg(),
						pgxmock.AnyArg(), pgxmock.AnyArg(), runID, tenantID,
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			wantErr: domain.ErrRunNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewRunRepositoryWithDB(mock)
			err = repo.Update(context.Background(), tt.run)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNewRunRepository(t *testing.T) {
	repo := NewRunRepositoryWithDB(nil)
	assert.NotNil(t, repo)
}
