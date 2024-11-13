package interfaces

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sebuszqo/FinanceManager/internal/finance/application"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	financeErrors "github.com/sebuszqo/FinanceManager/internal/finance/errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

type TransactionServiceInterface interface {
	CreateTransaction(transaction *domain.PersonalTransaction) error
	CreateTransactionsBulk(transactions []*domain.PersonalTransaction, userID string) error
	GetUserTransactions(userID, transactionType string, startDate, endDate time.Time, limit, page int) ([]domain.PersonalTransaction, error)
	UpdateTransaction(transaction domain.PersonalTransaction) error
	DeleteTransaction(transactionID int) error
	GetTransactionSummary(userID string, startDate, endDate time.Time) (map[int]application.TransactionSummary, error)
	GetTransactionSummaryByCategory(userID string, startDate, endDate time.Time, transactionType string) ([]domain.TransactionByCategorySummary, error)
}

type PersonalTransactionHandler struct {
	service      TransactionServiceInterface
	respondJSON  func(w http.ResponseWriter, status int, payload interface{})
	respondError func(w http.ResponseWriter, status int, message string, errors ...[]string)
}

func NewPersonalTransactionHandler(
	service TransactionServiceInterface,
	respondJSON func(w http.ResponseWriter, status int, payload interface{}),
	respondError func(w http.ResponseWriter, status int, message string, errors ...[]string),
) *PersonalTransactionHandler {
	if service == nil {
		log.Fatal("Service must not be nil")
		return nil
	}
	if respondJSON == nil {
		log.Fatal("RespondJSON function must not be nil")
		return nil
	}
	if respondError == nil {
		log.Fatal("RespondError function must not be nil")
		return nil
	}
	return &PersonalTransactionHandler{
		service:      service,
		respondJSON:  respondJSON,
		respondError: respondError,
	}
}

func (h *PersonalTransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var transaction domain.PersonalTransaction
	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	transaction.UserID = userID
	if err := h.service.CreateTransaction(&transaction); err != nil {
		if financeErrors.IsValidationError(err) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		fmt.Println("Error during transaction creation:", err.Error())
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
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req struct {
		Transactions []*domain.PersonalTransaction `json:"transactions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.Transactions) == 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid request body - no transactions provided")
		return
	}

	if err := h.service.CreateTransactionsBulk(req.Transactions, userID); err != nil {
		if financeErrors.IsValidationErrors(err) {
			var validationErrors *financeErrors.ValidationErrors
			errors.As(err, &validationErrors)
			errorMessages := make([]string, len(validationErrors.Errors))
			for i, vErr := range validationErrors.Errors {
				errorMessages[i] = vErr.Error()
			}
			h.respondError(w, http.StatusBadRequest, "Validation errors occurred", errorMessages)
			return
		}
		fmt.Println("Error during transaction creation:", err.Error())
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

	transactionType := r.URL.Query().Get("type")
	if !domain.IsValidTransactionType(transactionType) {
		h.respondError(w, http.StatusBadRequest, "Invalid transaction type")
		return
	}

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

	limitStr := r.URL.Query().Get("limit")
	pageStr := r.URL.Query().Get("page")
	var limit, page int
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			h.respondError(w, http.StatusBadRequest, "Invalid limit value")
			return
		}
	} else {
		limit = 20
	}

	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page <= 0 {
			h.respondError(w, http.StatusBadRequest, "Invalid page value")
			return
		}
	} else {
		page = 1
	}

	transactions, err := h.service.GetUserTransactions(userID, transactionType, startDate, endDate, limit, page)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve transactions")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Transactions retrieved successfully.",
		"data":    transactions,
	})
}

func (h *PersonalTransactionHandler) GetTransactionSummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
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
	summary, err := h.service.GetTransactionSummary(userID, startDate, endDate)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve transaction summary")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Transactions summary retrieved successfully.",
		"data":    summary,
	})
}

func (h *PersonalTransactionHandler) GetTransactionSummaryByCategory(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	transactionType := r.URL.Query().Get("type")

	if !domain.IsValidTransactionType(transactionType) {
		h.respondError(w, http.StatusBadRequest, "Invalid transaction type")
		return
	}

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

	summary, err := h.service.GetTransactionSummaryByCategory(userID, startDate, endDate, transactionType)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve category summary")
		return
	}
	if summary == nil {
		summary = []domain.TransactionByCategorySummary{}
	}
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Category summary retrieved successfully.",
		"data":    summary,
	})
}
