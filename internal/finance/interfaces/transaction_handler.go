package interfaces

import (
	"encoding/json"
	"errors"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	financeErrors "github.com/sebuszqo/FinanceManager/internal/finance/errors"
	"net/http"
)

type TransactionServiceInterface interface {
	CreateTransaction(transaction domain.PersonalTransaction) error
	CreateTransactionsBulk(transactions []domain.PersonalTransaction) error
	GetUserTransactions(userID string) ([]domain.PersonalTransaction, error)
	UpdateTransaction(transaction domain.PersonalTransaction) error
	DeleteTransaction(transactionID int) error
}

type PersonalTransactionHandler struct {
	service      TransactionServiceInterface
	respondJSON  func(w http.ResponseWriter, status int, payload interface{})
	respondError func(w http.ResponseWriter, status int, message string)
}

func NewPersonalTransactionHandler(
	service TransactionServiceInterface,
	respondJSON func(w http.ResponseWriter, status int, payload interface{}),
	respondError func(w http.ResponseWriter, status int, message string),
) *PersonalTransactionHandler {
	if service == nil || respondJSON == nil || respondError == nil {
		panic("Service and response functions must not be nil")
	}
	return &PersonalTransactionHandler{
		service:      service,
		respondJSON:  respondJSON,
		respondError: respondError,
	}
}

func (h *PersonalTransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var transaction domain.PersonalTransaction
	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.CreateTransaction(transaction); err != nil {
		if financeErrors.IsValidationError(err) {
			h.respondError(w, http.StatusBadRequest, err.Error())
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to create transaction")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *PersonalTransactionHandler) CreateTransactionsBulk(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Transactions []domain.PersonalTransaction `json:"transactions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Transactions == nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body - no transactions provided")
		return
	}

	if err := h.service.CreateTransactionsBulk(req.Transactions); err != nil {
		if financeErrors.IsValidationErrors(err) {
			var validationErrors *financeErrors.ValidationErrors
			errors.As(err, &validationErrors)
			errorMessages := make([]string, len(validationErrors.Errors))
			for i, vErr := range validationErrors.Errors {
				errorMessages[i] = vErr.Error()
			}
			h.respondJSON(w, http.StatusBadRequest, map[string][]string{"errors": errorMessages})
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to create transaction")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *PersonalTransactionHandler) GetUserTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	transactions, err := h.service.GetUserTransactions(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(transactions)
}
