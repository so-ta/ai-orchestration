package domain

import (
	"time"

	"github.com/google/uuid"
)

// GitSyncDirection represents the sync direction for git integration
type GitSyncDirection string

const (
	GitSyncDirectionPush          GitSyncDirection = "push"          // Project -> Git
	GitSyncDirectionPull          GitSyncDirection = "pull"          // Git -> Project
	GitSyncDirectionBidirectional GitSyncDirection = "bidirectional" // Both ways
)

// ProjectGitSync represents git sync configuration for a project
type ProjectGitSync struct {
	ID            uuid.UUID        `json:"id"`
	TenantID      uuid.UUID        `json:"tenant_id"`
	ProjectID     uuid.UUID        `json:"project_id"`
	RepositoryURL string           `json:"repository_url"`
	Branch        string           `json:"branch"`
	FilePath      string           `json:"file_path"` // Path to workflow JSON in repo
	SyncDirection GitSyncDirection `json:"sync_direction"`
	AutoSync      bool             `json:"auto_sync"`
	LastSyncAt    *time.Time       `json:"last_sync_at,omitempty"`
	LastCommitSHA string           `json:"last_commit_sha,omitempty"`
	CredentialsID *uuid.UUID       `json:"credentials_id,omitempty"` // Git credentials
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// NewProjectGitSync creates a new git sync configuration
func NewProjectGitSync(tenantID, projectID uuid.UUID, repoURL string) *ProjectGitSync {
	now := time.Now().UTC()
	return &ProjectGitSync{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProjectID:     projectID,
		RepositoryURL: repoURL,
		Branch:        "main",
		FilePath:      "workflow.json",
		SyncDirection: GitSyncDirectionBidirectional,
		AutoSync:      false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// UpdateLastSync updates the last sync timestamp and commit SHA
func (g *ProjectGitSync) UpdateLastSync(commitSHA string) {
	now := time.Now().UTC()
	g.LastSyncAt = &now
	g.LastCommitSHA = commitSHA
	g.UpdatedAt = now
}

// SetCredentials sets the git credentials
func (g *ProjectGitSync) SetCredentials(credentialsID uuid.UUID) {
	g.CredentialsID = &credentialsID
	g.UpdatedAt = time.Now().UTC()
}

// ClearCredentials clears the git credentials
func (g *ProjectGitSync) ClearCredentials() {
	g.CredentialsID = nil
	g.UpdatedAt = time.Now().UTC()
}

// GitSyncOperation represents a git sync operation
type GitSyncOperation struct {
	ID          uuid.UUID `json:"id"`
	GitSyncID   uuid.UUID `json:"git_sync_id"`
	Operation   string    `json:"operation"` // "push" or "pull"
	Status      string    `json:"status"`    // "pending", "running", "completed", "failed"
	CommitSHA   string    `json:"commit_sha,omitempty"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}
