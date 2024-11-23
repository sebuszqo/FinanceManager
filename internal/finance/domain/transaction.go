package domain

import (
	"database/sql"
	"github.com/sebuszqo/FinanceManager/internal/finance/errors"
	"math"
	"time"
)

type PersonalTransactionRepository interface {
	Save(transaction PersonalTransaction) error
	GetTransactionsByType(userID string, transactionType string, startDate time.Time, endDate time.Time, limit int, page int) ([]PersonalTransaction, error)
	FindByID(transactionID int) (*PersonalTransaction, error)
	Delete(transactionID int) error
	Update(transaction PersonalTransaction) error
	SaveWithTransaction(transaction PersonalTransaction, tx *sql.Tx) error
	BeginTransaction() (*sql.Tx, error)
	GetTransactionsInDateRange(userID string, startDate, endDate time.Time) ([]PersonalTransaction, error)
	GetTransactionSummaryByCategory(userID string, startDate, endDate time.Time, transactionType string) ([]TransactionByCategorySummary, error)
	GetTransactionSummaryByPaymentMethod(userID string, startDate, endDate time.Time, transactionType string) ([]TransactionByPaymentMethodSummary, error)
}

type PersonalTransaction struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	UserID               string    // user UUID
	Amount               float64   `json:"amount"`
	Type                 string    `json:"type"` // "income" lub "expense"
	Date                 time.Time `json:"date"`
	Description          *string   `json:"description"`
	PredefinedCategoryID int       `json:"predefined_category_id"`
	UserCategoryID       *int      `json:"user_category_id"`
	PaymentMethodID      int       `json:"payment_method_id"`
	PaymentSourceID      *int      `json:"payment_source_id"`
}

func (t *PersonalTransaction) RoundToTwoDecimalPlaces() {
	t.Amount = math.Round(t.Amount*100) / 100
}

func (t *PersonalTransaction) Validate() error {
	if len(t.Name) <= 0 || len(t.Name) > 50 {
		return errors.NewValidationError("Name should be between 0 and 50")
	}

	if t.Amount <= 0 {
		return errors.NewValidationError("Amount must be greater than zero")
	}

	if t.Type != "income" && t.Type != "expense" {
		return errors.NewValidationError("Type must be either 'income' or 'expense'")
	}

	if t.PredefinedCategoryID <= 0 {
		return errors.NewValidationError("PredefinedCategoryID must be provided and must be greater than zero")
	}

	if t.UserCategoryID != nil && *t.UserCategoryID <= 0 {
		return errors.NewValidationError("UserCategoryID, if provided, must be greater than zero")
	}

	if t.PaymentMethodID <= 0 {
		return errors.NewValidationError("PaymentMethodID must be provided and must be greater than zero")
	}

	if t.PaymentSourceID != nil && *t.PaymentSourceID <= 0 {
		return errors.NewValidationError("PaymentSourceID, if provided, must be greater than zero")
	}

	if t.Date.IsZero() {
		return errors.NewValidationError("Date is required")
	}

	if t.Description != nil && (len(*t.Description) > 200 || len(*t.Description) <= 0) {
		return errors.NewValidationError("Description if provided length must be less than or equal to 200 characters and greater than 0")
	}

	return nil
}

type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "income"
	TransactionTypeExpense TransactionType = "expense"
)

func IsValidTransactionType(t string) bool {
	return t == string(TransactionTypeIncome) || t == string(TransactionTypeExpense) || t == ""
}

type TransactionByCategorySummary struct {
	CategoryID   int     `json:"category_id"`
	CategoryName string  `json:"category_name"`
	TotalAmount  float64 `json:"total_amount"`
}

type TransactionByPaymentMethodSummary struct {
	PaymentMethodID   int     `json:"payment_method_id"`
	PaymentMethodName string  `json:"payment_method_name"`
	TotalAmount       float64 `json:"total_amount"`
}
