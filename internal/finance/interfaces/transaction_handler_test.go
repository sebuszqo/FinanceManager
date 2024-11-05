package interfaces

import (
	"bytes"
	"encoding/json"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"status":  "error",
		"message": message,
		"code":    status,
	})
}

func TestCreateTransactionsBulk_WithValidationError(t *testing.T) {
	service := &MockTransactionService{}
	handler := NewPersonalTransactionHandler(service, respondJSON, respondError)

	transactions := []domain.PersonalTransaction{
		{Amount: -10, Description: "Invalid transaction", Type: "expense", PredefinedCategoryID: nil, UserCategoryID: nil},
		{Amount: 50, Description: "Invalid category", Type: "income", PredefinedCategoryID: new(int), UserCategoryID: nil},
		{Amount: 20, Description: "Test 3", Type: "income", PredefinedCategoryID: nil, UserCategoryID: nil},
		{Amount: 20, Description: "Without Type", PredefinedCategoryID: nil, UserCategoryID: nil},
	}

	body, err := json.Marshal(map[string]interface{}{
		"transactions": transactions,
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/transactions/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateTransactionsBulk(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	var response map[string][]string
	err = json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)

	expectedErrors := []string{
		"Validation error at transaction 1: Category ID must be provided",
		"Validation error at transaction 2: Invalid predefined category ID",
		"Validation error at transaction 3: Category ID must be provided",
		"Validation error at transaction 4: Type must be 'income' or 'expense'",
	}
	assert.Equal(t, expectedErrors, response["errors"])
}

func TestCreateTransactionsBulk_InvalidRequestBody(t *testing.T) {
	service := &MockTransactionService{}
	handler := NewPersonalTransactionHandler(service, respondJSON, respondError)

	// Test 1: Body nie jest w formacie JSON
	req := httptest.NewRequest(http.MethodPost, "/transactions/bulk", bytes.NewBufferString("invalid body"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateTransactionsBulk(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(res.Body).Decode(&response)
	assert.NoError(t, err)

	// Sprawdź pełną strukturę odpowiedzi
	assert.Equal(t, "error", response["status"])
	assert.Equal(t, "Invalid request body", response["message"])
	assert.Equal(t, float64(http.StatusBadRequest), response["code"]) // Kod statusu będzie typu float64 po dekodowaniu JSON

	// Test 2: Brakuje klucza "transactions" w JSON
	body, err := json.Marshal(map[string]interface{}{
		"wrongKey": []domain.PersonalTransaction{
			{Amount: 100, Type: "income", PredefinedCategoryID: nil, UserCategoryID: nil},
		},
	})
	assert.NoError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/transactions/bulk", bytes.NewBuffer(body))
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

	// Test 3: "transactions" nie jest tablicą
	body, err = json.Marshal(map[string]interface{}{
		"transactions": "this should be an array, not a string",
	})
	assert.NoError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/transactions/bulk", bytes.NewBuffer(body))
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
}
