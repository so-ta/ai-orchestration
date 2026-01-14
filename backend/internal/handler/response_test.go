package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestJSON_Success tests successful JSON encoding
func TestJSON_Success(t *testing.T) {
	rec := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	JSON(rec, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var result map[string]string
	err := json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "value", result["key"])
}

// TestJSON_EncodeError tests JSON encoding error handling
// When json.Encode fails, the function should log the error (not panic)
func TestJSON_EncodeError(t *testing.T) {
	rec := httptest.NewRecorder()

	// Channels cannot be marshaled to JSON
	ch := make(chan int)

	// Should not panic, just log the error
	assert.NotPanics(t, func() {
		JSON(rec, http.StatusOK, ch)
	})

	// Status should still be set
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

// TestJSONData_Success tests JSONData helper
func TestJSONData_Success(t *testing.T) {
	rec := httptest.NewRecorder()
	data := "test value"

	JSONData(rec, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result Response
	err := json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "test value", result.Data)
}

// TestJSONList_Success tests JSONList helper
func TestJSONList_Success(t *testing.T) {
	rec := httptest.NewRecorder()
	data := []string{"a", "b", "c"}

	JSONList(rec, http.StatusOK, data, 1, 10, 100)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result Response
	err := json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result.Meta)
	assert.Equal(t, 1, result.Meta.Page)
	assert.Equal(t, 10, result.Meta.Limit)
	assert.Equal(t, 100, result.Meta.Total)
}

// TestError_Success tests Error helper
func TestError_Success(t *testing.T) {
	rec := httptest.NewRecorder()

	Error(rec, http.StatusBadRequest, "TEST_ERROR", "Test message", map[string]string{"field": "value"})

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "TEST_ERROR", result.Error.Code)
	assert.Equal(t, "Test message", result.Error.Message)
	assert.NotNil(t, result.Error.Details)
}

// TestHandleError_ValidationError tests HandleError with ValidationError
func TestHandleError_ValidationError(t *testing.T) {
	rec := httptest.NewRecorder()
	validationErr := domain.NewValidationError("name", "Name is required")

	HandleError(rec, validationErr)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", result.Error.Code)
}

// TestHandleError_NotFoundErrors tests HandleError with various not found errors
func TestHandleError_NotFoundErrors(t *testing.T) {
	notFoundErrors := []error{
		domain.ErrWorkflowNotFound,
		domain.ErrStepNotFound,
		domain.ErrRunNotFound,
		domain.ErrScheduleNotFound,
		domain.ErrWebhookNotFound,
		domain.ErrTenantNotFound,
	}

	for _, domainErr := range notFoundErrors {
		t.Run(domainErr.Error(), func(t *testing.T) {
			rec := httptest.NewRecorder()
			HandleError(rec, domainErr)

			assert.Equal(t, http.StatusNotFound, rec.Code)

			var result ErrorResponse
			err := json.NewDecoder(rec.Body).Decode(&result)
			assert.NoError(t, err)
			assert.Equal(t, "NOT_FOUND", result.Error.Code)
		})
	}
}

// TestHandleError_UnknownError tests HandleError with unknown error (internal server error)
func TestHandleError_UnknownError(t *testing.T) {
	rec := httptest.NewRecorder()
	unknownErr := errors.New("some unexpected error")

	HandleError(rec, unknownErr)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var result ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "INTERNAL_ERROR", result.Error.Code)
	assert.Equal(t, "internal server error", result.Error.Message)
}

// TestHandleError_InputValidationErrors tests HandleError with InputValidationErrors
func TestHandleError_InputValidationErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	inputValidationErr := &domain.InputValidationErrors{
		Errors: []domain.InputValidationError{
			{Field: "email", Message: "email is required"},
			{Field: "age", Message: "age must be of type integer"},
		},
	}

	HandleError(rec, inputValidationErr)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "SCHEMA_VALIDATION_ERROR", result.Error.Code)
	assert.Equal(t, "Input validation failed", result.Error.Message)
	assert.NotNil(t, result.Error.Details)

	// Check details structure
	details, ok := result.Error.Details.(map[string]interface{})
	assert.True(t, ok)
	errorsArr, ok := details["errors"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, errorsArr, 2)
}
