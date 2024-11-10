package interfaces

import (
	"encoding/json"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCategories_ValidTypeIncome(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/categories?type=income", nil)
	w := httptest.NewRecorder()

	mockService := &MockCategoryService{
		categories: []domain.PredefinedCategory{
			{ID: 1, Name: "Salary", Type: "income"},
			{ID: 2, Name: "Bonus", Type: "income"},
		},
	}
	handler := NewCategoryHandler(mockService, respondJSON, respondError)
	handler.GetCategories(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	var categories []domain.PredefinedCategory
	err := json.NewDecoder(res.Body).Decode(&categories)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(categories))
}

func TestGetCategories_InvalidType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/categories?type=invalidType", nil)
	w := httptest.NewRecorder()

	mockService := &MockCategoryService{}
	handler := NewCategoryHandler(mockService, respondJSON, respondError)
	handler.GetCategories(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, "Invalid category type", response["message"])
}

func TestGetCategories_ErrorFromService(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()

	mockService := &MockCategoryService{
		shouldFail: true,
	}
	handler := NewCategoryHandler(mockService, respondJSON, respondError)
	handler.GetCategories(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, "Failed to retrieve categories", response["message"])
}
