package engine

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateToolArgs(t *testing.T) {
	t.Run("nil schema returns nil error", func(t *testing.T) {
		args := map[string]interface{}{"foo": "bar"}
		err := validateToolArgs(args, nil)
		assert.Nil(t, err)
	})

	t.Run("empty args with no required fields passes", func(t *testing.T) {
		schema := map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
		err := validateToolArgs(map[string]interface{}{}, schema)
		assert.Nil(t, err)
	})

	t.Run("missing required field returns error", func(t *testing.T) {
		schema := map[string]interface{}{
			"type":     "object",
			"required": []interface{}{"slug"},
			"properties": map[string]interface{}{
				"slug": map[string]interface{}{"type": "string"},
			},
		}
		args := map[string]interface{}{}

		err := validateToolArgs(args, schema)

		assert.NotNil(t, err)
		assert.Contains(t, err.Message, "Missing required fields")
		assert.Contains(t, err.MissingFields, "slug")
	})

	t.Run("empty string for required field returns error", func(t *testing.T) {
		schema := map[string]interface{}{
			"type":     "object",
			"required": []interface{}{"slug"},
			"properties": map[string]interface{}{
				"slug": map[string]interface{}{"type": "string"},
			},
		}
		args := map[string]interface{}{"slug": ""}

		err := validateToolArgs(args, schema)

		assert.NotNil(t, err)
		assert.Contains(t, err.MissingFields, "slug")
	})

	t.Run("provided required field passes", func(t *testing.T) {
		schema := map[string]interface{}{
			"type":     "object",
			"required": []interface{}{"slug"},
			"properties": map[string]interface{}{
				"slug": map[string]interface{}{"type": "string"},
			},
		}
		args := map[string]interface{}{"slug": "llm"}

		err := validateToolArgs(args, schema)

		assert.Nil(t, err)
	})

	t.Run("multiple missing required fields", func(t *testing.T) {
		schema := map[string]interface{}{
			"type":     "object",
			"required": []interface{}{"name", "type", "config"},
			"properties": map[string]interface{}{
				"name":   map[string]interface{}{"type": "string"},
				"type":   map[string]interface{}{"type": "string"},
				"config": map[string]interface{}{"type": "object"},
			},
		}
		args := map[string]interface{}{"name": "test"}

		err := validateToolArgs(args, schema)

		assert.NotNil(t, err)
		assert.Len(t, err.MissingFields, 2)
		assert.Contains(t, err.MissingFields, "type")
		assert.Contains(t, err.MissingFields, "config")
	})

	t.Run("json.RawMessage schema", func(t *testing.T) {
		schema := json.RawMessage(`{
			"type": "object",
			"required": ["workflow_id"],
			"properties": {
				"workflow_id": {"type": "string"}
			}
		}`)
		args := map[string]interface{}{}

		err := validateToolArgs(args, schema)

		assert.NotNil(t, err)
		assert.Contains(t, err.MissingFields, "workflow_id")
	})

	t.Run("schema with optional fields only", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit":    map[string]interface{}{"type": "integer"},
				"category": map[string]interface{}{"type": "string"},
			},
		}
		args := map[string]interface{}{}

		err := validateToolArgs(args, schema)

		assert.Nil(t, err)
	})
}
