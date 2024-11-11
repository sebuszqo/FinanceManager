package interfaces

import (
	"encoding/json"
	"errors"
	"github.com/sebuszqo/FinanceManager/internal/finance/application"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	financeErrors "github.com/sebuszqo/FinanceManager/internal/finance/errors"
	"net/http"
	"time"
)

type TransactionServiceInterface interface {
	CreateTransaction(transaction domain.PersonalTransaction) error
	CreateTransactionsBulk(transactions []domain.PersonalTransaction) error
	GetUserTransactions(userID string) ([]domain.PersonalTransaction, error)
	UpdateTransaction(transaction domain.PersonalTransaction) error
	DeleteTransaction(transactionID int) error
	GetTransactionSummary(startDate, endDate time.Time) (map[int]application.TransactionSummary, error)
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

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Transaction successfully created.",
		"data":    transaction,
	})
}

func (h *PersonalTransactionHandler) CreateTransactionsBulk(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Transactions []domain.PersonalTransaction `json:"transactions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.Transactions) == 0 {
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
	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Transactions successfully created.",
		"data":    req.Transactions,
	})
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

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Transactions retrieved successfully.",
		"data":    transactions,
	})
}

func (h *PersonalTransactionHandler) GetTransactionSummary(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time
	var err error

	if startDateStr == "" {
		startDate = time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "Invalid start date format")
			return
		}
	}

	if endDateStr == "" {
		endDate = time.Now()
	} else {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "Invalid end date format")
			return
		}
	}
	summary, err := h.service.GetTransactionSummary(startDate, endDate)
	if err != nil {
		http.Error(w, "Failed to retrieve transaction summary", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Transactions summary retrieved successfully.",
		"data":    summary,
	})
}
