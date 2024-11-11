package interfaces

import (
	"encoding/json"
	"errors"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPaymentMethods_Success(t *testing.T) {
	mockMethods := []domain.PaymentMethod{
		{ID: 1, Name: "Payment Card"},
		{ID: 2, Name: "Cash"},
	}

	mockService := NewMockPaymentService(mockMethods, nil)
	handler := NewPaymentHandler(mockService, respondJSON, respondError)

	req := httptest.NewRequest(http.MethodGet, "/api/protected/finance/payment/methods", nil)
	w := httptest.NewRecorder()

	handler.GetPaymentMethods(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "Methods retrieved successfully.", response["message"])

	methods, ok := response["methods"].([]interface{})
	assert.True(t, ok, "Expected 'methods' to be an array in the response")

	if assert.Len(t, methods, len(mockMethods)) {
		for i, method := range methods {
			methodMap, ok := method.(map[string]interface{})
			assert.True(t, ok, "Expected each method in 'methods' to be a map")

			id, idOk := methodMap["id"].(float64)
			if !idOk {
				t.Errorf("Expected id to be a float64, but got %v", methodMap["id"])
			}
			assert.Equal(t, float64(mockMethods[i].ID), id)

			name, nameOk := methodMap["name"].(string)
			if !nameOk {
				t.Errorf("Expected name to be a string, but got %v", methodMap["name"])
			}
			assert.Equal(t, mockMethods[i].Name, name)
		}
	}
}

func TestGetPaymentMethods_Error(t *testing.T) {

	mockService := NewMockPaymentService(nil, errors.New("database error"))
	handler := NewPaymentHandler(mockService, respondJSON, respondError)

	req := httptest.NewRequest(http.MethodGet, "/api/protected/finance/payment/methods", nil)
	w := httptest.NewRecorder()

	handler.GetPaymentMethods(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, "error", response["status"])
	assert.Equal(t, "Failed to retrieve payment methods", response["message"])
}
