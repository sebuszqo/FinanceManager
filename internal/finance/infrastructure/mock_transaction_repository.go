package infrastructure

import (
	"database/sql"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	"time"
)

type MockTransactionRepository struct {
	Transactions []domain.PersonalTransaction
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

func (m *MockTransactionRepository) GetTransactionsInDateRange(startDate, endDate time.Time) ([]domain.PersonalTransaction, error) {

	var filtered []domain.PersonalTransaction
	for _, transaction := range m.Transactions {
		if transaction.Date.After(startDate) && transaction.Date.Before(endDate) {
			filtered = append(filtered, transaction)
		}
	}
	return filtered, nil
}
