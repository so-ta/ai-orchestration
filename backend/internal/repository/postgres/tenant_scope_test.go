package postgres

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewTenantScope(t *testing.T) {
	tests := []struct {
		name     string
		tenantID uuid.UUID
		wantErr  bool
	}{
		{
			name:     "valid tenant ID",
			tenantID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			wantErr:  false,
		},
		{
			name:     "nil tenant ID",
			tenantID: uuid.Nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, err := NewTenantScope(nil, tt.tenantID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTenantScope() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && scope == nil {
				t.Error("NewTenantScope() returned nil scope for valid tenant ID")
			}
		})
	}
}

func TestTenantFilter(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	resourceID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	t.Run("basic filter", func(t *testing.T) {
		filter := NewTenantFilter(tenantID)
		where := filter.Where()
		args := filter.Args()

		expectedWhere := "WHERE tenant_id = $1"
		if where != expectedWhere {
			t.Errorf("Where() = %q, want %q", where, expectedWhere)
		}
		if len(args) != 1 || args[0] != tenantID {
			t.Errorf("Args() = %v, want [%v]", args, tenantID)
		}
	})

	t.Run("filter with NotDeleted", func(t *testing.T) {
		filter := NewTenantFilter(tenantID).NotDeleted()
		where := filter.Where()

		expectedWhere := "WHERE tenant_id = $1 AND deleted_at IS NULL"
		if where != expectedWhere {
			t.Errorf("Where() = %q, want %q", where, expectedWhere)
		}
	})

	t.Run("filter with ByID", func(t *testing.T) {
		filter := NewTenantFilter(tenantID).NotDeleted().ByID(resourceID)
		where := filter.Where()
		args := filter.Args()

		expectedWhere := "WHERE tenant_id = $1 AND deleted_at IS NULL AND id = $2"
		if where != expectedWhere {
			t.Errorf("Where() = %q, want %q", where, expectedWhere)
		}
		if len(args) != 2 {
			t.Errorf("Args() length = %d, want 2", len(args))
		}
		if args[0] != tenantID {
			t.Errorf("Args()[0] = %v, want %v", args[0], tenantID)
		}
		if args[1] != resourceID {
			t.Errorf("Args()[1] = %v, want %v", args[1], resourceID)
		}
	})

	t.Run("filter with And", func(t *testing.T) {
		filter := NewTenantFilter(tenantID).
			NotDeleted().
			And("status = $1", "active")
		where := filter.Where()
		args := filter.Args()

		expectedWhere := "WHERE tenant_id = $1 AND deleted_at IS NULL AND status = $2"
		if where != expectedWhere {
			t.Errorf("Where() = %q, want %q", where, expectedWhere)
		}
		if len(args) != 2 {
			t.Errorf("Args() length = %d, want 2", len(args))
		}
	})
}

func TestEnsureTenantMatch(t *testing.T) {
	tenantA := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tenantB := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	tests := []struct {
		name     string
		expected uuid.UUID
		actual   uuid.UUID
		wantErr  bool
	}{
		{
			name:     "matching tenants",
			expected: tenantA,
			actual:   tenantA,
			wantErr:  false,
		},
		{
			name:     "mismatched tenants",
			expected: tenantA,
			actual:   tenantB,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureTenantMatch(tt.expected, tt.actual)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureTenantMatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTenantScopedQuery(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	resourceID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	t.Run("basic query", func(t *testing.T) {
		query := NewTenantScopedQuery("SELECT * FROM projects WHERE tenant_id = $1", tenantID)

		if query.SQL() != "SELECT * FROM projects WHERE tenant_id = $1" {
			t.Errorf("SQL() unexpected value")
		}
		args := query.Args()
		if len(args) != 1 || args[0] != tenantID {
			t.Errorf("Args() = %v, want [%v]", args, tenantID)
		}
	})

	t.Run("query with additional args", func(t *testing.T) {
		query := NewTenantScopedQuery(
			"SELECT * FROM projects WHERE tenant_id = $1 AND id = $2",
			tenantID,
		).WithArgs(resourceID)

		args := query.Args()
		if len(args) != 2 {
			t.Errorf("Args() length = %d, want 2", len(args))
		}
		if args[0] != tenantID {
			t.Errorf("Args()[0] = %v, want %v", args[0], tenantID)
		}
		if args[1] != resourceID {
			t.Errorf("Args()[1] = %v, want %v", args[1], resourceID)
		}
	})
}
