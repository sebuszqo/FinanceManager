package application

import (
	"fmt"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	financeErrors "github.com/sebuszqo/FinanceManager/internal/finance/errors"
)

type CategoryServiceInterface interface {
	DoesPredefinedCategoryExist(categoryID int) (bool, error)
	DoesUserCategoryExist(categoryID int, userID string) (bool, error)
	GetAllPredefinedCategories(categoryType string) ([]domain.PredefinedCategory, error)
	GetAllUserCategories(userID string) ([]domain.UserCategory, error)
}

type PersonalTransactionService struct {
	repo            domain.PersonalTransactionRepository
	categoryService CategoryServiceInterface
}

func NewPersonalTransactionService(repo domain.PersonalTransactionRepository, categoryService CategoryServiceInterface) *PersonalTransactionService {
	return &PersonalTransactionService{repo: repo, categoryService: categoryService}
}

func (s *PersonalTransactionService) CreateTransaction(transaction domain.PersonalTransaction) error {
	if err := transaction.Validate(); err != nil {
		return err
	}

	if transaction.PredefinedCategoryID != nil {
		exists, err := s.categoryService.DoesPredefinedCategoryExist(*transaction.PredefinedCategoryID)
		if err != nil {
			return err
		}
		if !exists {
			return financeErrors.ErrInvalidPredefinedCategory
		}
	}

	if transaction.UserCategoryID != nil {
		exists, err := s.categoryService.DoesUserCategoryExist(*transaction.UserCategoryID, transaction.UserID)
		if err != nil {
			return err
		}
		if !exists {
			return financeErrors.ErrInvalidUserCategory
		}
	}

	return s.repo.Save(transaction)
}

func (s *PersonalTransactionService) CreateTransactionsBulk(transactions []domain.PersonalTransaction) error {
	predefinedCategories, err := s.categoryService.GetAllPredefinedCategories("")
	if err != nil {
		return err
	}

	userCategories, err := s.categoryService.GetAllUserCategories(transactions[0].UserID)
	if err != nil {
		return err
	}

	predefinedCategoryMap := make(map[int]struct{})
	userCategoryMap := make(map[int]struct{})

	for _, category := range predefinedCategories {
		predefinedCategoryMap[category.ID] = struct{}{}
	}
	for _, category := range userCategories {
		userCategoryMap[category.ID] = struct{}{}
	}

	tx, err := s.repo.BeginTransaction()
	if err != nil {
		return err
	}
	var validationErrors = &financeErrors.ValidationErrors{}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	for i, transaction := range transactions {
		if err := transaction.Validate(); err != nil {
			return financeErrors.NewIndexedValidationError(i+1, err.Error())
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
		}

		if err := s.repo.SaveWithTransaction(transaction, tx); err != nil {
			return fmt.Errorf("database error at transaction %d: %w", i+1, err)
		}
	}

	if len(validationErrors.Errors) > 0 {
		return validationErrors
	}

	return nil
}

func (s *PersonalTransactionService) GetUserTransactions(userID string) ([]domain.PersonalTransaction, error) {
	return s.repo.FindByUser(userID)
}

func (s *PersonalTransactionService) UpdateTransaction(transaction domain.PersonalTransaction) error {
	return s.repo.Update(transaction)
}

func (s *PersonalTransactionService) DeleteTransaction(transactionID int) error {
	return s.repo.Delete(transactionID)
}
