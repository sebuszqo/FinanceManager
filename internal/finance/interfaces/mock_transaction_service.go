package interfaces

import (
	"fmt"
	"github.com/sebuszqo/FinanceManager/internal/finance/application"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	financeErrors "github.com/sebuszqo/FinanceManager/internal/finance/errors"
	"github.com/stretchr/testify/mock"
	"time"
)

type MockTransactionService struct {
	mock.Mock
}

func (m *MockTransactionService) GetTransactionSummaryByCategory(userID string, startDate, endDate time.Time, transactionType string) ([]domain.TransactionByCategorySummary, error) {
	summary := []domain.TransactionByCategorySummary{
		{
			CategoryID:   1,
			CategoryName: "Food",
			TotalAmount:  150.00,
		},
		{
			CategoryID:   2,
			CategoryName: "Transport",
			TotalAmount:  50.00,
		},
	}

	if transactionType == "income" {
		return summary, nil
	} else if transactionType == "expense" {
		return summary, nil
	}

	return nil, fmt.Errorf("invalid transaction type: %s", transactionType)
}

func (m *MockTransactionService) GetUserTransactions(userID string, transactionType string, startDate time.Time, endDate time.Time, limit int, page int) ([]domain.PersonalTransaction, error) {
	args := m.Called(userID, transactionType)

	transactions := args.Get(0)
	if transactions != nil {
		return transactions.([]domain.PersonalTransaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTransactionService) GetTransactionSummary(userID string, startDate time.Time, endDate time.Time) (map[int]application.TransactionSummary, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionService) CreateTransaction(transaction *domain.PersonalTransaction) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionService) UpdateTransaction(transaction domain.PersonalTransaction) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionService) DeleteTransaction(transactionID int) error {
	//TODO implement me
	panic("implement me")
}

var predefinedCategoryMap = map[int]struct{}{
	1: {},
	2: {},
}

var userCategoryMap = map[int]struct{}{
	10: {},
	20: {},
}

func (m *MockTransactionService) CreateTransactionsBulk(transactions []*domain.PersonalTransaction, userID string) error {
	var validationErrors = &financeErrors.ValidationErrors{}

	for i, transaction := range transactions {
		if err := transaction.Validate(); err != nil {
			validationErrors.Add(financeErrors.NewIndexedValidationError(i+1, err.Error()))
			continue
		}

		if _, exists := predefinedCategoryMap[transaction.PredefinedCategoryID]; !exists {
			validationErrors.Add(financeErrors.NewIndexedValidationError(i+1, financeErrors.ErrInvalidPredefinedCategory.Error()))
			continue
		}

		if transaction.UserCategoryID != nil {
			if _, exists := userCategoryMap[*transaction.UserCategoryID]; !exists {
				validationErrors.Add(financeErrors.NewIndexedValidationError(i+1, financeErrors.ErrInvalidUserCategory.Error()))
				continue
			}
		} else {
			validationErrors.Add(financeErrors.NewIndexedValidationError(i+1, "Category ID must be provided"))
		}
	}

	if len(validationErrors.Errors) > 0 {
		return validationErrors
	}
	return nil
}
