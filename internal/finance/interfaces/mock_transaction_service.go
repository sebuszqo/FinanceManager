package interfaces

import (
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	financeErrors "github.com/sebuszqo/FinanceManager/internal/finance/errors"
)

type MockTransactionService struct{}

func (m *MockTransactionService) CreateTransaction(transaction domain.PersonalTransaction) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockTransactionService) GetUserTransactions(userID string) ([]domain.PersonalTransaction, error) {
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

func (m *MockTransactionService) CreateTransactionsBulk(transactions []domain.PersonalTransaction) error {
	var validationErrors = &financeErrors.ValidationErrors{}

	for i, transaction := range transactions {
		if err := transaction.Validate(); err != nil {
			validationErrors.Add(financeErrors.NewIndexedValidationError(i+1, err.Error()))
			continue
		}

		if transaction.PredefinedCategoryID != nil {
			if _, exists := predefinedCategoryMap[*transaction.PredefinedCategoryID]; !exists {
				validationErrors.Add(financeErrors.NewIndexedValidationError(i+1, financeErrors.ErrInvalidPredefinedCategory.Error()))
				continue
			}
		} else if transaction.UserCategoryID != nil {
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
