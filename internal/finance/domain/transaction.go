package domain

import (
	"database/sql"
	"github.com/sebuszqo/FinanceManager/internal/finance/errors"
	"time"
)

type PersonalTransactionRepository interface {
	Save(transaction PersonalTransaction) error
	FindByUser(userID string) ([]PersonalTransaction, error)
	FindByID(transactionID int) (*PersonalTransaction, error)
	Delete(transactionID int) error
	Update(transaction PersonalTransaction) error
	SaveWithTransaction(transaction PersonalTransaction, tx *sql.Tx) error
	BeginTransaction() (*sql.Tx, error)
	GetTransactionsInDateRange(startDate, endDate time.Time) ([]PersonalTransaction, error)
}

type PersonalTransaction struct {
	ID                   int
	UserID               string // user UUID
	Amount               float64
	Type                 string // "income" lub "expense"
	Date                 time.Time
	Description          string
	PredefinedCategoryID *int
	UserCategoryID       *int
	PaymentMethodID      *int
	PaymentSourceID      *int
}

func (t *PersonalTransaction) Validate() error {
	// it doesnt' need to be greater than zero :D
	//if t.Amount <= 0 {
	//	return errors.NewValidationError("Amount must be greater than zero")
	//}
	if t.Type != "income" && t.Type != "expense" {
		return errors.NewValidationError("Type must be 'income' or 'expense'")
	}
	if len(t.Description) > 200 {
		return errors.NewValidationError("Description must be of length less than 200")
	}
	// optional?
	//if (t.PredefinedCategoryID == nil && t.UserCategoryID == nil) || (t.PredefinedCategoryID != nil && t.UserCategoryID != nil) {
	//	return NewValidationError("Exactly one of PredefinedCategoryID or UserCategoryID must be set")
	//}
	return nil
}
