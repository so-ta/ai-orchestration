package domain

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrUserNotFound is returned when a user is not found
var ErrUserNotFound = errors.New("user not found")

// User represents a user in the system
type User struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Email       string          `json:"email"`
	Name        string          `json:"name,omitempty"`
	Role        string          `json:"role"`
	Variables   json.RawMessage `json:"variables"`
	LastLoginAt *time.Time      `json:"last_login_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// GetVariables parses and returns variables as a map
func (u *User) GetVariables() (map[string]interface{}, error) {
	var vars map[string]interface{}
	if u.Variables == nil || len(u.Variables) == 0 {
		return make(map[string]interface{}), nil
	}
	if err := json.Unmarshal(u.Variables, &vars); err != nil {
		return nil, err
	}
	return vars, nil
}
