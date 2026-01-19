package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TemplateVisibility represents the visibility level of a template
type TemplateVisibility string

const (
	TemplateVisibilityPrivate TemplateVisibility = "private" // Only owner can see
	TemplateVisibilityTenant  TemplateVisibility = "tenant"  // All users in tenant can see
	TemplateVisibilityPublic  TemplateVisibility = "public"  // Published to marketplace
)

// TemplateReviewStatus represents the review status for marketplace templates
type TemplateReviewStatus string

const (
	TemplateReviewStatusPending  TemplateReviewStatus = "pending"
	TemplateReviewStatusApproved TemplateReviewStatus = "approved"
	TemplateReviewStatusRejected TemplateReviewStatus = "rejected"
)

// ProjectTemplate represents a reusable workflow template
type ProjectTemplate struct {
	ID            uuid.UUID          `json:"id"`
	TenantID      *uuid.UUID         `json:"tenant_id,omitempty"` // NULL for system templates
	Name          string             `json:"name"`
	Description   string             `json:"description,omitempty"`
	Category      string             `json:"category,omitempty"`
	Tags          []string           `json:"tags,omitempty"`
	Definition    json.RawMessage    `json:"definition"` // Snapshot of steps, edges, block_groups
	Variables     json.RawMessage    `json:"variables,omitempty"`
	ThumbnailURL  string             `json:"thumbnail_url,omitempty"`
	AuthorName    string             `json:"author_name,omitempty"`
	DownloadCount int                `json:"download_count"`
	IsFeatured    bool               `json:"is_featured"`

	// Marketplace fields
	Visibility   TemplateVisibility    `json:"visibility"`
	ReviewStatus *TemplateReviewStatus `json:"review_status,omitempty"`
	PriceUSD     float64               `json:"price_usd"`
	Rating       *float64              `json:"rating,omitempty"`
	ReviewCount  int                   `json:"review_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TemplateDefinition contains the complete template structure
type TemplateDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Variables   json.RawMessage `json:"variables,omitempty"`
	Steps       []Step          `json:"steps"`
	Edges       []Edge          `json:"edges"`
	BlockGroups []BlockGroup    `json:"block_groups,omitempty"`
}

// NewProjectTemplate creates a new project template
func NewProjectTemplate(tenantID *uuid.UUID, name, description string, definition json.RawMessage) *ProjectTemplate {
	now := time.Now().UTC()
	return &ProjectTemplate{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Definition:  definition,
		Variables:   json.RawMessage(`{}`),
		Tags:        []string{},
		Visibility:  TemplateVisibilityPrivate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// GetDefinition unmarshals and returns the template definition
func (t *ProjectTemplate) GetDefinition() (*TemplateDefinition, error) {
	var def TemplateDefinition
	if err := json.Unmarshal(t.Definition, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

// SetDefinition marshals and sets the template definition
func (t *ProjectTemplate) SetDefinition(def *TemplateDefinition) error {
	data, err := json.Marshal(def)
	if err != nil {
		return err
	}
	t.Definition = data
	return nil
}

// IsPublic returns true if the template is published to marketplace
func (t *ProjectTemplate) IsPublic() bool {
	return t.Visibility == TemplateVisibilityPublic
}

// IsApproved returns true if the template is approved for marketplace
func (t *ProjectTemplate) IsApproved() bool {
	return t.ReviewStatus != nil && *t.ReviewStatus == TemplateReviewStatusApproved
}

// CanView checks if a tenant can view this template
func (t *ProjectTemplate) CanView(tenantID uuid.UUID) bool {
	// System templates (no tenant) are visible to all
	if t.TenantID == nil {
		return true
	}

	// Owner can always view
	if *t.TenantID == tenantID {
		return true
	}

	// Public approved templates are visible to all
	if t.IsPublic() && t.IsApproved() {
		return true
	}

	return false
}

// IncrementDownloadCount increments the download count
func (t *ProjectTemplate) IncrementDownloadCount() {
	t.DownloadCount++
	t.UpdatedAt = time.Now().UTC()
}

// UpdateRating updates the average rating based on a new review
func (t *ProjectTemplate) UpdateRating(newRating int) {
	if t.Rating == nil {
		rating := float64(newRating)
		t.Rating = &rating
		t.ReviewCount = 1
	} else {
		// Calculate new average
		total := *t.Rating * float64(t.ReviewCount)
		t.ReviewCount++
		newAvg := (total + float64(newRating)) / float64(t.ReviewCount)
		t.Rating = &newAvg
	}
	t.UpdatedAt = time.Now().UTC()
}

// TemplateReview represents a user review for a template
type TemplateReview struct {
	ID         uuid.UUID `json:"id"`
	TemplateID uuid.UUID `json:"template_id"`
	UserID     uuid.UUID `json:"user_id"`
	Rating     int       `json:"rating"` // 1-5
	Comment    string    `json:"comment,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewTemplateReview creates a new template review
func NewTemplateReview(templateID, userID uuid.UUID, rating int, comment string) *TemplateReview {
	return &TemplateReview{
		ID:         uuid.New(),
		TemplateID: templateID,
		UserID:     userID,
		Rating:     rating,
		Comment:    comment,
		CreatedAt:  time.Now().UTC(),
	}
}

// TemplateCategory represents a template category
type TemplateCategory struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// DefaultTemplateCategories returns the default template categories
func DefaultTemplateCategories() []TemplateCategory {
	return []TemplateCategory{
		{Slug: "sales", Name: "Sales", Description: "Sales automation workflows", Icon: "dollar-sign"},
		{Slug: "marketing", Name: "Marketing", Description: "Marketing automation workflows", Icon: "megaphone"},
		{Slug: "support", Name: "Support", Description: "Customer support workflows", Icon: "headphones"},
		{Slug: "hr", Name: "HR", Description: "Human resources workflows", Icon: "users"},
		{Slug: "engineering", Name: "Engineering", Description: "Development workflows", Icon: "code"},
		{Slug: "data", Name: "Data", Description: "Data processing workflows", Icon: "database"},
		{Slug: "ai", Name: "AI", Description: "AI and ML workflows", Icon: "brain"},
		{Slug: "integration", Name: "Integration", Description: "System integration workflows", Icon: "plug"},
		{Slug: "other", Name: "Other", Description: "Other workflows", Icon: "folder"},
	}
}
