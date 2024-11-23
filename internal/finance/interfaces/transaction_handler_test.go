package interfaces

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestCreateTransactionsBulk_WithValidationError(t *testing.T) {
	service := &MockTransactionService{}
	handler := NewPersonalTransactionHandler(service, respondJSON, respondError)

	ctx := context.WithValue(context.Background(), "userID", "test-user-id")

	body, err := json.Marshal(map[string]interface{}{
		"transactions": []domain.PersonalTransaction{
			{Name: "Transaction number1", Amount: -10, Type: "income"},                          // Invalid Amount
			{Name: "Transaction number2", Amount: 100, Type: "income", PredefinedCategoryID: 0}, // Invalid Category ID
			{Name: "Transaction number3", Amount: 50, Type: "income", PredefinedCategoryID: 3},  // Missing PaymentMethodID
			{Name: "Transaction number4", Amount: 60, Type: "invalid_type"},
			{Amount: 10, Type: "income"},
		},
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/transactions/bulk", bytes.NewBuffer(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateTransactionsBulk(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, "error", response["status"])
	assert.Equal(t, "Validation errors occurred", response["message"])

	if errorsList, ok := response["errors"].([]interface{}); ok {
		expectedErrors := []string{
			"Validation error at transaction 1: Amount must be greater than zero",
			"Validation error at transaction 2: PredefinedCategoryID must be provided and must be greater than zero",
			"Validation error at transaction 3: PaymentMethodID must be provided and must be greater than zero",
			"Validation error at transaction 4: Type must be either 'income' or 'expense'",
			"Validation error at transaction 5: Name should be between 0 and 50",
		}

		actualErrors := make([]string, len(errorsList))
		for i, err := range errorsList {
			actualErrors[i] = err.(string)
		}

		assert.Equal(t, expectedErrors, actualErrors)
	} else {
		t.Fatalf("Expected errors list, got: %v", response["errors"])
	}
}

func TestCreateTransactionsBulk_InvalidRequestBody(t *testing.T) {
	service := &MockTransactionService{}
	handler := NewPersonalTransactionHandler(service, respondJSON, respondError)

	ctx := context.WithValue(context.Background(), "userID", "test-user-id")

	req := httptest.NewRequest(http.MethodPost, "/transactions/bulk", bytes.NewBufferString("invalid body")).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateTransactionsBulk(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "error", response["status"])
	assert.Equal(t, "Invalid request body", response["message"])
	assert.Equal(t, float64(http.StatusBadRequest), response["code"])

	body, err := json.Marshal(map[string]interface{}{
		"wrongKey": []domain.PersonalTransaction{
			{Amount: 100, Type: "income", PredefinedCategoryID: 2, UserCategoryID: nil},
		},
	})
	assert.NoError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/transactions/bulk", bytes.NewBuffer(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.CreateTransactionsBulk(w, req)

	res = w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	err = json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "error", response["status"])
	assert.Equal(t, "Invalid request body - no transactions provided", response["message"])
	assert.Equal(t, float64(http.StatusBadRequest), response["code"])

	body, err = json.Marshal(map[string]interface{}{
		"transactions": "this should be an array, not a string",
	})
	assert.NoError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/transactions/bulk", bytes.NewBuffer(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.CreateTransactionsBulk(w, req)

	res = w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	err = json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "error", response["status"])
	assert.Equal(t, "Invalid request body", response["message"])
	assert.Equal(t, float64(http.StatusBadRequest), response["code"])

	body, err = json.Marshal(map[string]interface{}{
		"transactions": []domain.PersonalTransaction{
			{Amount: -100, Type: "income", PredefinedCategoryID: 2, UserCategoryID: nil}, // Invalid Amount
			{Amount: 50, Type: "expense", PredefinedCategoryID: 0, UserCategoryID: nil},  // Invalid CategoryID
		},
	})
	assert.NoError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/transactions/bulk", bytes.NewBuffer(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.CreateTransactionsBulk(w, req)

	res = w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	err = json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "error", response["status"])
	assert.Contains(t, response["message"], "Validation errors occurred")
	assert.Equal(t, float64(http.StatusBadRequest), response["code"])

	if errorsList, ok := response["errors"].([]interface{}); ok {
		assert.Greater(t, len(errorsList), 0)
	}
}

func TestGetUserTransactions(t *testing.T) {
	mockService := &MockTransactionService{}
	handler := NewPersonalTransactionHandler(mockService, respondJSON, respondError)

	t.Run("Unauthorized if userID is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
		w := httptest.NewRecorder()

		handler.GetUserTransactions(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("Bad request if transaction type is invalid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions?type=invalidType", nil)
		ctx := context.WithValue(req.Context(), "userID", "valid-user-id")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUserTransactions(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		var response map[string]interface{}
		json.NewDecoder(res.Body).Decode(&response)
		assert.Equal(t, "Invalid transaction type", response["message"])
	})

	t.Run("Returns transactions on valid request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions?type=income", nil)
		ctx := context.WithValue(req.Context(), "userID", "valid-user-id")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		mockService.On("GetUserTransactions", "valid-user-id", "income").Return([]domain.PersonalTransaction{
			{ID: strconv.Itoa(1), Amount: 100, Type: "income"},
		}, nil)

		handler.GetUserTransactions(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		var response map[string]interface{}
		json.NewDecoder(res.Body).Decode(&response)
		assert.Equal(t, "success", response["status"])
		assert.Equal(t, "Transactions retrieved successfully.", response["message"])
	})

	t.Run("Internal server error if service fails", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions?type=expense", nil)
		ctx := context.WithValue(req.Context(), "userID", "valid-user-id")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		mockService.On("GetUserTransactions", "valid-user-id", "expense").Return(nil, errors.New("service error"))

		handler.GetUserTransactions(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		var response map[string]interface{}
		json.NewDecoder(res.Body).Decode(&response)
		assert.Equal(t, "Failed to retrieve transactions", response["message"])
	})
}

func TestGetTransactionSummaryByCategory_ValidRequest(t *testing.T) {
	mockService := &MockTransactionService{}

	handler := &PersonalTransactionHandler{
		service:      mockService,
		respondJSON:  respondJSON,
		respondError: respondError,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/protected/finance/transactions/summary?type=income", nil)
	req = req.WithContext(context.WithValue(req.Context(), "userID", "valid-user-id"))
	w := httptest.NewRecorder()

	mockService.On("GetTransactionSummaryByCategory", "valid-user-id", mock.Anything, mock.Anything, "income").
		Return([]domain.TransactionByCategorySummary{
			{CategoryID: 1, CategoryName: "Food", TotalAmount: 100.0},
			{CategoryID: 2, CategoryName: "Transport", TotalAmount: 50.0},
		}, nil)

	handler.GetTransactionSummaryByCategory(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	var response map[string]interface{}
	json.NewDecoder(res.Body).Decode(&response)
	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "Category summary retrieved successfully.", response["message"])
}

func TestGetTransactionSummaryByCategory_InvalidTransactionType(t *testing.T) {

	mockService := &MockTransactionService{}
	handler := NewPersonalTransactionHandler(mockService, respondJSON, respondError)

	req := httptest.NewRequest(http.MethodGet, "/transactions/summary?type=invalid", nil)
	req = req.WithContext(context.WithValue(req.Context(), "userID", "valid-user-id"))
	w := httptest.NewRecorder()

	handler.GetTransactionSummaryByCategory(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	var response map[string]interface{}
	err := json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, "Invalid transaction type", response["message"])
}
