package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// ProjectTemplateRepository handles project template persistence
type ProjectTemplateRepository struct {
	db *pgxpool.Pool
}

// NewProjectTemplateRepository creates a new ProjectTemplateRepository
func NewProjectTemplateRepository(db *pgxpool.Pool) *ProjectTemplateRepository {
	return &ProjectTemplateRepository{db: db}
}

// Create creates a new project template
func (r *ProjectTemplateRepository) Create(ctx context.Context, template *domain.ProjectTemplate) error {
	query := `
		INSERT INTO project_templates (
			id, tenant_id, name, description, category, tags, definition, variables,
			thumbnail_url, author_name, download_count, is_featured,
			visibility, review_status, price_usd, rating, review_count,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	_, err := r.db.Exec(ctx, query,
		template.ID,
		template.TenantID,
		template.Name,
		template.Description,
		template.Category,
		template.Tags,
		template.Definition,
		template.Variables,
		template.ThumbnailURL,
		template.AuthorName,
		template.DownloadCount,
		template.IsFeatured,
		template.Visibility,
		template.ReviewStatus,
		template.PriceUSD,
		template.Rating,
		template.ReviewCount,
		template.CreatedAt,
		template.UpdatedAt,
	)

	return err
}

// GetByID retrieves a project template by ID
func (r *ProjectTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectTemplate, error) {
	query := `
		SELECT id, tenant_id, name, description, category, tags, definition, variables,
		       thumbnail_url, author_name, download_count, is_featured,
		       visibility, review_status, price_usd, rating, review_count,
		       created_at, updated_at
		FROM project_templates
		WHERE id = $1
	`

	template := &domain.ProjectTemplate{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&template.ID,
		&template.TenantID,
		&template.Name,
		&template.Description,
		&template.Category,
		&template.Tags,
		&template.Definition,
		&template.Variables,
		&template.ThumbnailURL,
		&template.AuthorName,
		&template.DownloadCount,
		&template.IsFeatured,
		&template.Visibility,
		&template.ReviewStatus,
		&template.PriceUSD,
		&template.Rating,
		&template.ReviewCount,
		&template.CreatedAt,
		&template.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// List retrieves project templates with filtering
func (r *ProjectTemplateRepository) List(ctx context.Context, filter repository.TemplateFilter) ([]*domain.ProjectTemplate, int, error) {
	return r.listWithConditions(ctx, filter, "")
}

// ListPublic retrieves public approved templates
func (r *ProjectTemplateRepository) ListPublic(ctx context.Context, filter repository.TemplateFilter) ([]*domain.ProjectTemplate, int, error) {
	return r.listWithConditions(ctx, filter, "visibility = 'public' AND review_status = 'approved'")
}

// ListByTenant retrieves templates for a specific tenant
func (r *ProjectTemplateRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, filter repository.TemplateFilter) ([]*domain.ProjectTemplate, int, error) {
	return r.listWithConditions(ctx, filter, fmt.Sprintf("tenant_id = '%s'", tenantID))
}

func (r *ProjectTemplateRepository) listWithConditions(ctx context.Context, filter repository.TemplateFilter, baseCondition string) ([]*domain.ProjectTemplate, int, error) {
	// Build WHERE clause
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if baseCondition != "" {
		conditions = append(conditions, baseCondition)
	}

	if filter.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, *filter.Category)
		argIndex++
	}

	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+*filter.Search+"%")
		argIndex++
	}

	if filter.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("is_featured = $%d", argIndex))
		args = append(args, *filter.IsFeatured)
		argIndex++
	}

	if filter.MinRating != nil {
		conditions = append(conditions, fmt.Sprintf("rating >= $%d", argIndex))
		args = append(args, *filter.MinRating)
		argIndex++
	}

	if filter.Visibility != nil {
		conditions = append(conditions, fmt.Sprintf("visibility = $%d", argIndex))
		args = append(args, *filter.Visibility)
		argIndex++
	}

	if len(filter.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIndex))
		args = append(args, filter.Tags)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM project_templates " + whereClause
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := `
		SELECT id, tenant_id, name, description, category, tags, definition, variables,
		       thumbnail_url, author_name, download_count, is_featured,
		       visibility, review_status, price_usd, rating, review_count,
		       created_at, updated_at
		FROM project_templates
	` + whereClause + " ORDER BY is_featured DESC, download_count DESC, created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Page > 0 && filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var templates []*domain.ProjectTemplate
	for rows.Next() {
		template := &domain.ProjectTemplate{}
		err := rows.Scan(
			&template.ID,
			&template.TenantID,
			&template.Name,
			&template.Description,
			&template.Category,
			&template.Tags,
			&template.Definition,
			&template.Variables,
			&template.ThumbnailURL,
			&template.AuthorName,
			&template.DownloadCount,
			&template.IsFeatured,
			&template.Visibility,
			&template.ReviewStatus,
			&template.PriceUSD,
			&template.Rating,
			&template.ReviewCount,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		templates = append(templates, template)
	}

	return templates, total, rows.Err()
}

// Update updates a project template
func (r *ProjectTemplateRepository) Update(ctx context.Context, template *domain.ProjectTemplate) error {
	query := `
		UPDATE project_templates
		SET name = $2, description = $3, category = $4, tags = $5, definition = $6, variables = $7,
		    thumbnail_url = $8, author_name = $9, is_featured = $10,
		    visibility = $11, review_status = $12, price_usd = $13, rating = $14, review_count = $15,
		    updated_at = $16
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		template.ID,
		template.Name,
		template.Description,
		template.Category,
		template.Tags,
		template.Definition,
		template.Variables,
		template.ThumbnailURL,
		template.AuthorName,
		template.IsFeatured,
		template.Visibility,
		template.ReviewStatus,
		template.PriceUSD,
		template.Rating,
		template.ReviewCount,
		template.UpdatedAt,
	)

	return err
}

// Delete deletes a project template
func (r *ProjectTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM project_templates WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// IncrementDownloadCount increments the download count for a template
func (r *ProjectTemplateRepository) IncrementDownloadCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE project_templates SET download_count = download_count + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// TemplateReviewRepository handles template review persistence
type TemplateReviewRepository struct {
	db *pgxpool.Pool
}

// NewTemplateReviewRepository creates a new TemplateReviewRepository
func NewTemplateReviewRepository(db *pgxpool.Pool) *TemplateReviewRepository {
	return &TemplateReviewRepository{db: db}
}

// Create creates a new template review
func (r *TemplateReviewRepository) Create(ctx context.Context, review *domain.TemplateReview) error {
	query := `
		INSERT INTO template_reviews (id, template_id, user_id, rating, comment, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		review.ID,
		review.TemplateID,
		review.UserID,
		review.Rating,
		review.Comment,
		review.CreatedAt,
	)

	return err
}

// GetByID retrieves a template review by ID
func (r *TemplateReviewRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TemplateReview, error) {
	query := `
		SELECT id, template_id, user_id, rating, comment, created_at
		FROM template_reviews
		WHERE id = $1
	`

	review := &domain.TemplateReview{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&review.ID,
		&review.TemplateID,
		&review.UserID,
		&review.Rating,
		&review.Comment,
		&review.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return review, nil
}

// ListByTemplate retrieves all reviews for a template
func (r *TemplateReviewRepository) ListByTemplate(ctx context.Context, templateID uuid.UUID) ([]*domain.TemplateReview, error) {
	query := `
		SELECT id, template_id, user_id, rating, comment, created_at
		FROM template_reviews
		WHERE template_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*domain.TemplateReview
	for rows.Next() {
		review := &domain.TemplateReview{}
		err := rows.Scan(
			&review.ID,
			&review.TemplateID,
			&review.UserID,
			&review.Rating,
			&review.Comment,
			&review.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}

	return reviews, rows.Err()
}

// GetByTemplateAndUser retrieves a review by template and user
func (r *TemplateReviewRepository) GetByTemplateAndUser(ctx context.Context, templateID, userID uuid.UUID) (*domain.TemplateReview, error) {
	query := `
		SELECT id, template_id, user_id, rating, comment, created_at
		FROM template_reviews
		WHERE template_id = $1 AND user_id = $2
	`

	review := &domain.TemplateReview{}
	err := r.db.QueryRow(ctx, query, templateID, userID).Scan(
		&review.ID,
		&review.TemplateID,
		&review.UserID,
		&review.Rating,
		&review.Comment,
		&review.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return review, nil
}

// Delete deletes a template review
func (r *TemplateReviewRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM template_reviews WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
