package usecase

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// TemplateUsecase handles template business logic
type TemplateUsecase struct {
	templateRepo repository.ProjectTemplateRepository
	reviewRepo   repository.TemplateReviewRepository
	projectRepo  repository.ProjectRepository
	stepRepo     repository.StepRepository
	edgeRepo     repository.EdgeRepository
}

// NewTemplateUsecase creates a new TemplateUsecase
func NewTemplateUsecase(
	templateRepo repository.ProjectTemplateRepository,
	reviewRepo repository.TemplateReviewRepository,
	projectRepo repository.ProjectRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
) *TemplateUsecase {
	return &TemplateUsecase{
		templateRepo: templateRepo,
		reviewRepo:   reviewRepo,
		projectRepo:  projectRepo,
		stepRepo:     stepRepo,
		edgeRepo:     edgeRepo,
	}
}

// CreateTemplateInput represents input for creating a template
type CreateTemplateInput struct {
	TenantID    uuid.UUID
	Name        string
	Description string
	Category    string
	Tags        []string
	Definition  json.RawMessage
	Variables   json.RawMessage
	AuthorName  string
	Visibility  domain.TemplateVisibility
}

// Create creates a new template
func (u *TemplateUsecase) Create(ctx context.Context, input CreateTemplateInput) (*domain.ProjectTemplate, error) {
	if input.Name == "" {
		return nil, domain.NewValidationError("name", "name is required")
	}

	template := domain.NewProjectTemplate(&input.TenantID, input.Name, input.Description, input.Definition)
	template.Category = input.Category
	template.Tags = input.Tags
	template.Variables = input.Variables
	template.AuthorName = input.AuthorName
	template.Visibility = input.Visibility

	if err := u.templateRepo.Create(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// CreateFromProjectInput represents input for creating a template from a project
type CreateFromProjectInput struct {
	TenantID    uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Description string
	Category    string
	Tags        []string
	AuthorName  string
	Visibility  domain.TemplateVisibility
}

// CreateFromProject creates a template from an existing project
func (u *TemplateUsecase) CreateFromProject(ctx context.Context, input CreateFromProjectInput) (*domain.ProjectTemplate, error) {
	// Get the project
	project, err := u.projectRepo.GetByID(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Get steps and edges
	steps, err := u.stepRepo.ListByProject(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}

	edges, err := u.edgeRepo.ListByProject(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Create template definition
	def := domain.TemplateDefinition{
		Name:        project.Name,
		Description: project.Description,
		Variables:   project.Variables,
		Steps:       make([]domain.Step, len(steps)),
		Edges:       make([]domain.Edge, len(edges)),
	}

	for i, step := range steps {
		def.Steps[i] = *step
	}
	for i, edge := range edges {
		def.Edges[i] = *edge
	}

	defJSON, err := json.Marshal(def)
	if err != nil {
		return nil, err
	}

	// Create template
	name := input.Name
	if name == "" {
		name = project.Name
	}
	description := input.Description
	if description == "" {
		description = project.Description
	}

	template := domain.NewProjectTemplate(&input.TenantID, name, description, defJSON)
	template.Category = input.Category
	template.Tags = input.Tags
	template.Variables = project.Variables
	template.AuthorName = input.AuthorName
	template.Visibility = input.Visibility

	if err := u.templateRepo.Create(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// GetByID retrieves a template by ID
func (u *TemplateUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectTemplate, error) {
	return u.templateRepo.GetByID(ctx, id)
}

// ListTemplatesInput represents input for listing templates
type ListTemplatesInput struct {
	TenantID   *uuid.UUID
	Category   *string
	Tags       []string
	Search     *string
	IsFeatured *bool
	MinRating  *float64
	Visibility *domain.TemplateVisibility
	Page       int
	Limit      int
}

// ListTemplatesOutput represents output for listing templates
type ListTemplatesOutput struct {
	Templates []*domain.ProjectTemplate
	Page      int
	Limit     int
	Total     int
}

// List lists templates
func (u *TemplateUsecase) List(ctx context.Context, input ListTemplatesInput) (*ListTemplatesOutput, error) {
	filter := repository.TemplateFilter{
		Category:   input.Category,
		Tags:       input.Tags,
		Search:     input.Search,
		IsFeatured: input.IsFeatured,
		MinRating:  input.MinRating,
		Visibility: input.Visibility,
		Page:       input.Page,
		Limit:      input.Limit,
	}

	var templates []*domain.ProjectTemplate
	var total int
	var err error

	if input.TenantID != nil {
		templates, total, err = u.templateRepo.ListByTenant(ctx, *input.TenantID, filter)
	} else {
		templates, total, err = u.templateRepo.ListPublic(ctx, filter)
	}

	if err != nil {
		return nil, err
	}

	return &ListTemplatesOutput{
		Templates: templates,
		Page:      input.Page,
		Limit:     input.Limit,
		Total:     total,
	}, nil
}

// UpdateTemplateInput represents input for updating a template
type UpdateTemplateInput struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	Category    string
	Tags        []string
	Definition  json.RawMessage
	Variables   json.RawMessage
	Visibility  *domain.TemplateVisibility
}

// Update updates a template
func (u *TemplateUsecase) Update(ctx context.Context, input UpdateTemplateInput) (*domain.ProjectTemplate, error) {
	template, err := u.templateRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if template.TenantID == nil || *template.TenantID != input.TenantID {
		return nil, domain.ErrForbidden
	}

	if input.Name != "" {
		template.Name = input.Name
	}
	if input.Description != "" {
		template.Description = input.Description
	}
	if input.Category != "" {
		template.Category = input.Category
	}
	if input.Tags != nil {
		template.Tags = input.Tags
	}
	if input.Definition != nil {
		template.Definition = input.Definition
	}
	if input.Variables != nil {
		template.Variables = input.Variables
	}
	if input.Visibility != nil {
		template.Visibility = *input.Visibility
	}

	if err := u.templateRepo.Update(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// Delete deletes a template
func (u *TemplateUsecase) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	template, err := u.templateRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check ownership
	if template.TenantID == nil || *template.TenantID != tenantID {
		return domain.ErrForbidden
	}

	return u.templateRepo.Delete(ctx, id)
}

// UseTemplate creates a new project from a template
func (u *TemplateUsecase) UseTemplate(ctx context.Context, tenantID uuid.UUID, templateID uuid.UUID, projectName string) (*domain.Project, error) {
	template, err := u.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Check if user can view this template
	if !template.CanView(tenantID) {
		return nil, domain.ErrTemplateNotFound
	}

	// Parse template definition
	def, err := template.GetDefinition()
	if err != nil {
		return nil, err
	}

	// Create project
	name := projectName
	if name == "" {
		name = def.Name
	}
	project := domain.NewProject(tenantID, name, def.Description)
	project.Variables = def.Variables

	if err := u.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}

	// Create steps with new UUIDs
	stepIDMap := make(map[uuid.UUID]uuid.UUID) // old ID -> new ID
	for _, step := range def.Steps {
		newID := uuid.New()
		stepIDMap[step.ID] = newID

		newStep := domain.NewStep(tenantID, project.ID, step.Name, step.Type, step.Config)
		newStep.ID = newID
		newStep.SetPosition(step.PositionX, step.PositionY)
		newStep.TriggerType = step.TriggerType
		newStep.TriggerConfig = step.TriggerConfig
		newStep.CredentialBindings = step.CredentialBindings

		if err := u.stepRepo.Create(ctx, newStep); err != nil {
			return nil, err
		}
	}

	// Create edges with mapped step IDs
	for _, edge := range def.Edges {
		if edge.SourceStepID == nil || edge.TargetStepID == nil {
			continue
		}
		sourceID, ok := stepIDMap[*edge.SourceStepID]
		if !ok {
			continue
		}
		targetID, ok := stepIDMap[*edge.TargetStepID]
		if !ok {
			continue
		}

		condition := ""
		if edge.Condition != nil {
			condition = *edge.Condition
		}
		newEdge := domain.NewEdge(tenantID, project.ID, sourceID, targetID, condition)
		if err := u.edgeRepo.Create(ctx, newEdge); err != nil {
			return nil, err
		}
	}

	// Increment download count
	if err := u.templateRepo.IncrementDownloadCount(ctx, templateID); err != nil {
		// Non-critical error, log but don't fail
	}

	return project, nil
}

// AddReview adds a review to a template
func (u *TemplateUsecase) AddReview(ctx context.Context, tenantID uuid.UUID, templateID uuid.UUID, userID uuid.UUID, rating int, comment string) (*domain.TemplateReview, error) {
	template, err := u.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Check if user can view this template
	if !template.CanView(tenantID) {
		return nil, domain.ErrTemplateNotFound
	}

	// Validate rating
	if rating < 1 || rating > 5 {
		return nil, domain.NewValidationError("rating", "rating must be between 1 and 5")
	}

	// Check if user already reviewed
	existing, _ := u.reviewRepo.GetByTemplateAndUser(ctx, templateID, userID)
	if existing != nil {
		return nil, domain.NewValidationError("review", "user already reviewed this template")
	}

	review := domain.NewTemplateReview(templateID, userID, rating, comment)
	if err := u.reviewRepo.Create(ctx, review); err != nil {
		return nil, err
	}

	// Update template rating
	template.UpdateRating(rating)
	if err := u.templateRepo.Update(ctx, template); err != nil {
		return nil, err
	}

	return review, nil
}

// GetReviews gets reviews for a template
func (u *TemplateUsecase) GetReviews(ctx context.Context, templateID uuid.UUID) ([]*domain.TemplateReview, error) {
	return u.reviewRepo.ListByTemplate(ctx, templateID)
}

// GetCategories returns available template categories
func (u *TemplateUsecase) GetCategories() []domain.TemplateCategory {
	return domain.DefaultTemplateCategories()
}
