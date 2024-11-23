package infrastructure

import (
	"database/sql"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	"time"
)

type MockTransactionRepository struct {
	Transactions []domain.PersonalTransaction
}

func (m *MockTransactionRepository) GetTransactionSummaryByPaymentMethod(userID string, startDate, endDate time.Time, transactionType string) ([]domain.TransactionByPaymentMethodSummary, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionRepository) GetTransactionSummaryByCategory(userID string, startDate, endDate time.Time, transactionType string) ([]domain.TransactionByCategorySummary, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionRepository) GetTransactionsByType(userID string, transactionType string, startDate time.Time, endDate time.Time, limit int, page int) ([]domain.PersonalTransaction, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionRepository) Save(transaction domain.PersonalTransaction) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionRepository) FindByUser(userID string) ([]domain.PersonalTransaction, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionRepository) FindByID(transactionID int) (*domain.PersonalTransaction, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionRepository) Delete(transactionID int) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionRepository) Update(transaction domain.PersonalTransaction) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionRepository) SaveWithTransaction(transaction domain.PersonalTransaction, tx *sql.Tx) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionRepository) BeginTransaction() (*sql.Tx, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionRepository) GetTransactionsInDateRange(userID string, startDate time.Time, endDate time.Time) ([]domain.PersonalTransaction, error) {

	var filtered []domain.PersonalTransaction
	for _, transaction := range m.Transactions {
		if transaction.Date.After(startDate) && transaction.Date.Before(endDate) {
			filtered = append(filtered, transaction)
		}
	}
	return filtered, nil
}
