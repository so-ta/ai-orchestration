package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestNewProjectTemplate(t *testing.T) {
	tenantID := uuid.New()
	name := "Test Template"
	description := "Test Description"
	definition := json.RawMessage(`{"steps": []}`)

	template := NewProjectTemplate(&tenantID, name, description, definition)

	if template.ID == uuid.Nil {
		t.Error("NewProjectTemplate() should generate a non-nil UUID")
	}
	if template.TenantID == nil || *template.TenantID != tenantID {
		t.Error("NewProjectTemplate() TenantID mismatch")
	}
	if template.Name != name {
		t.Errorf("NewProjectTemplate() Name = %v, want %v", template.Name, name)
	}
	if template.Description != description {
		t.Errorf("NewProjectTemplate() Description = %v, want %v", template.Description, description)
	}
	if template.Visibility != TemplateVisibilityPrivate {
		t.Errorf("NewProjectTemplate() Visibility = %v, want %v", template.Visibility, TemplateVisibilityPrivate)
	}
}

func TestNewProjectTemplate_System(t *testing.T) {
	template := NewProjectTemplate(nil, "System Template", "Desc", json.RawMessage("{}"))

	if template.TenantID != nil {
		t.Error("NewProjectTemplate() with nil tenantID should have nil TenantID")
	}
}

func TestProjectTemplate_GetDefinition(t *testing.T) {
	template := &ProjectTemplate{
		Definition: json.RawMessage(`{"name": "Test", "description": "Desc", "steps": [], "edges": []}`),
	}

	def, err := template.GetDefinition()
	if err != nil {
		t.Fatalf("GetDefinition() error = %v", err)
	}

	if def.Name != "Test" {
		t.Errorf("GetDefinition() Name = %v, want Test", def.Name)
	}
}

func TestProjectTemplate_SetDefinition(t *testing.T) {
	template := &ProjectTemplate{}
	def := &TemplateDefinition{
		Name:        "Test",
		Description: "Description",
		Steps:       []Step{},
		Edges:       []Edge{},
	}

	err := template.SetDefinition(def)
	if err != nil {
		t.Fatalf("SetDefinition() error = %v", err)
	}

	if template.Definition == nil {
		t.Error("SetDefinition() should set Definition")
	}
}

func TestProjectTemplate_IsPublic(t *testing.T) {
	tests := []struct {
		visibility TemplateVisibility
		want       bool
	}{
		{TemplateVisibilityPrivate, false},
		{TemplateVisibilityTenant, false},
		{TemplateVisibilityPublic, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.visibility), func(t *testing.T) {
			template := &ProjectTemplate{Visibility: tt.visibility}
			if got := template.IsPublic(); got != tt.want {
				t.Errorf("IsPublic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectTemplate_IsApproved(t *testing.T) {
	approved := TemplateReviewStatusApproved
	pending := TemplateReviewStatusPending
	rejected := TemplateReviewStatusRejected

	tests := []struct {
		name         string
		reviewStatus *TemplateReviewStatus
		want         bool
	}{
		{"approved", &approved, true},
		{"pending", &pending, false},
		{"rejected", &rejected, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := &ProjectTemplate{ReviewStatus: tt.reviewStatus}
			if got := template.IsApproved(); got != tt.want {
				t.Errorf("IsApproved() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectTemplate_CanView(t *testing.T) {
	ownerTenantID := uuid.New()
	otherTenantID := uuid.New()
	approved := TemplateReviewStatusApproved
	pending := TemplateReviewStatusPending

	tests := []struct {
		name         string
		tenantID     *uuid.UUID
		visibility   TemplateVisibility
		reviewStatus *TemplateReviewStatus
		viewerID     uuid.UUID
		want         bool
	}{
		{"system template", nil, TemplateVisibilityPrivate, nil, otherTenantID, true},
		{"owner viewing private", &ownerTenantID, TemplateVisibilityPrivate, nil, ownerTenantID, true},
		{"other viewing private", &ownerTenantID, TemplateVisibilityPrivate, nil, otherTenantID, false},
		{"public approved", &ownerTenantID, TemplateVisibilityPublic, &approved, otherTenantID, true},
		{"public pending", &ownerTenantID, TemplateVisibilityPublic, &pending, otherTenantID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := &ProjectTemplate{
				TenantID:     tt.tenantID,
				Visibility:   tt.visibility,
				ReviewStatus: tt.reviewStatus,
			}
			if got := template.CanView(tt.viewerID); got != tt.want {
				t.Errorf("CanView() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectTemplate_IncrementDownloadCount(t *testing.T) {
	template := &ProjectTemplate{DownloadCount: 5}

	template.IncrementDownloadCount()

	if template.DownloadCount != 6 {
		t.Errorf("IncrementDownloadCount() DownloadCount = %v, want 6", template.DownloadCount)
	}
}

func TestProjectTemplate_UpdateRating(t *testing.T) {
	// First rating
	template := &ProjectTemplate{}
	template.UpdateRating(5)

	if template.Rating == nil || *template.Rating != 5.0 {
		t.Errorf("UpdateRating() first Rating = %v, want 5.0", template.Rating)
	}
	if template.ReviewCount != 1 {
		t.Errorf("UpdateRating() ReviewCount = %v, want 1", template.ReviewCount)
	}

	// Second rating
	template.UpdateRating(3)

	expectedAvg := 4.0 // (5 + 3) / 2
	if template.Rating == nil || *template.Rating != expectedAvg {
		t.Errorf("UpdateRating() second Rating = %v, want %v", *template.Rating, expectedAvg)
	}
	if template.ReviewCount != 2 {
		t.Errorf("UpdateRating() ReviewCount = %v, want 2", template.ReviewCount)
	}
}

func TestNewTemplateReview(t *testing.T) {
	templateID := uuid.New()
	userID := uuid.New()
	rating := 4
	comment := "Great template!"

	review := NewTemplateReview(templateID, userID, rating, comment)

	if review.ID == uuid.Nil {
		t.Error("NewTemplateReview() should generate a non-nil UUID")
	}
	if review.TemplateID != templateID {
		t.Error("NewTemplateReview() TemplateID mismatch")
	}
	if review.UserID != userID {
		t.Error("NewTemplateReview() UserID mismatch")
	}
	if review.Rating != rating {
		t.Errorf("NewTemplateReview() Rating = %v, want %v", review.Rating, rating)
	}
	if review.Comment != comment {
		t.Errorf("NewTemplateReview() Comment = %v, want %v", review.Comment, comment)
	}
}

func TestDefaultTemplateCategories(t *testing.T) {
	categories := DefaultTemplateCategories()

	if len(categories) != 9 {
		t.Errorf("DefaultTemplateCategories() returned %d categories, want 9", len(categories))
	}

	slugs := make(map[string]bool)
	for _, cat := range categories {
		slugs[cat.Slug] = true
	}

	expectedSlugs := []string{"sales", "marketing", "support", "hr", "engineering", "data", "ai", "integration", "other"}
	for _, slug := range expectedSlugs {
		if !slugs[slug] {
			t.Errorf("DefaultTemplateCategories() missing category: %v", slug)
		}
	}
}
