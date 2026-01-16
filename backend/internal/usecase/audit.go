package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// AuditService handles audit logging
type AuditService struct {
	repo repository.AuditLogRepository
}

// NewAuditService creates a new AuditService
func NewAuditService(repo repository.AuditLogRepository) *AuditService {
	return &AuditService{repo: repo}
}

// LogAuditInput represents input for logging an audit event
type LogAuditInput struct {
	TenantID     uuid.UUID
	ActorID      *uuid.UUID
	ActorEmail   string
	Action       domain.AuditAction
	ResourceType domain.AuditResourceType
	ResourceID   *uuid.UUID
	Metadata     map[string]interface{}
	IPAddress    string
	UserAgent    string
}

// Log logs an audit event
func (s *AuditService) Log(ctx context.Context, input LogAuditInput) error {
	var metadata json.RawMessage
	if input.Metadata != nil {
		var err error
		metadata, err = json.Marshal(input.Metadata)
		if err != nil {
			return err
		}
	}

	log := domain.NewAuditLog(
		input.TenantID,
		input.ActorID,
		input.ActorEmail,
		input.Action,
		input.ResourceType,
		input.ResourceID,
		metadata,
	)
	log.SetRequestInfo(input.IPAddress, input.UserAgent)

	return s.repo.Create(ctx, log)
}

// ListAuditLogsInput represents input for listing audit logs
type ListAuditLogsInput struct {
	TenantID     uuid.UUID
	ActorID      *uuid.UUID
	Action       *domain.AuditAction
	ResourceType *domain.AuditResourceType
	ResourceID   *uuid.UUID
	StartTime    *time.Time
	EndTime      *time.Time
	Page         int
	Limit        int
}

// ListAuditLogsOutput represents output for listing audit logs
type ListAuditLogsOutput struct {
	Logs  []*domain.AuditLog
	Total int
	Page  int
	Limit int
}

// List lists audit logs with pagination
func (s *AuditService) List(ctx context.Context, input ListAuditLogsInput) (*ListAuditLogsOutput, error) {
	input.Page, input.Limit = NormalizePaginationWithLimit(input.Page, input.Limit, DefaultAuditLimit)

	filter := repository.AuditLogFilter{
		ActorID:      input.ActorID,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		StartTime:    input.StartTime,
		EndTime:      input.EndTime,
		Page:         input.Page,
		Limit:        input.Limit,
	}

	logs, total, err := s.repo.ListByTenant(ctx, input.TenantID, filter)
	if err != nil {
		return nil, err
	}

	return &ListAuditLogsOutput{
		Logs:  logs,
		Total: total,
		Page:  input.Page,
		Limit: input.Limit,
	}, nil
}

// ListByResource lists audit logs for a specific resource
func (s *AuditService) ListByResource(ctx context.Context, tenantID uuid.UUID, resourceType domain.AuditResourceType, resourceID uuid.UUID) ([]*domain.AuditLog, error) {
	return s.repo.ListByResource(ctx, tenantID, resourceType, resourceID)
}
