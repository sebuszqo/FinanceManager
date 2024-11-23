package application

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	financeErrors "github.com/sebuszqo/FinanceManager/internal/finance/errors"
	"log"
	"time"
)

type CategoryServiceInterface interface {
	DoesPredefinedCategoryExist(categoryID int) (bool, error)
	DoesUserCategoryExist(categoryID int, userID string) (bool, error)
	GetAllPredefinedCategories(categoryType string) ([]domain.PredefinedCategory, error)
	GetAllUserCategories(userID string) ([]domain.UserCategory, error)
}

type PaymentServiceInterface interface {
	GetAllPaymentMethods() ([]domain.PaymentMethod, error)
	GetUserPaymentSources(userID string) ([]domain.PaymentSource, error)
	DoesPaymentMethodExistByID(methodID int) (bool, error)
	DoesUserPaymentSourceExistByID(sourceID int, userID string) (bool, error)
}

type PersonalTransactionService struct {
	repo            domain.PersonalTransactionRepository
	categoryService CategoryServiceInterface
	paymentService  PaymentServiceInterface
}

func NewPersonalTransactionService(repo domain.PersonalTransactionRepository, categoryService CategoryServiceInterface, paymentService PaymentServiceInterface) *PersonalTransactionService {
	return &PersonalTransactionService{repo: repo, categoryService: categoryService, paymentService: paymentService}
}

type TransactionSummary struct {
	Year         int
	IncomeTotal  float64
	ExpenseTotal float64
	Months       map[string]MonthSummary
}

type MonthSummary struct {
	IncomeTotal  float64
	ExpenseTotal float64
	Weeks        []WeekSummary
}

type WeekSummary struct {
	Week         int
	IncomeTotal  float64
	ExpenseTotal float64
}

func (s *PersonalTransactionService) GetTransactionSummary(userID string, startDate, endDate time.Time) (map[int]TransactionSummary, error) {
	transactions, err := s.repo.GetTransactionsInDateRange(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	summary := make(map[int]TransactionSummary)

	for _, transaction := range transactions {
		year := transaction.Date.Year()
		month := transaction.Date.Month().String()
		_, week := transaction.Date.ISOWeek()

		if _, exists := summary[year]; !exists {
			summary[year] = TransactionSummary{
				Year:         year,
				Months:       make(map[string]MonthSummary),
				IncomeTotal:  0,
				ExpenseTotal: 0,
			}
		}

		yearSummary := summary[year]

		if _, exists := yearSummary.Months[month]; !exists {
			yearSummary.Months[month] = MonthSummary{
				IncomeTotal:  0,
				ExpenseTotal: 0,
				Weeks:        []WeekSummary{},
			}
		}

		monthSummary := yearSummary.Months[month]

		if transaction.Type == "income" {
			yearSummary.IncomeTotal += transaction.Amount
			monthSummary.IncomeTotal += transaction.Amount
		} else if transaction.Type == "expense" {
			yearSummary.ExpenseTotal += transaction.Amount
			monthSummary.ExpenseTotal += transaction.Amount
		}

		found := false
		for i, weekSummary := range monthSummary.Weeks {
			if weekSummary.Week == week {
				if transaction.Type == "income" {
					monthSummary.Weeks[i].IncomeTotal += transaction.Amount
				} else if transaction.Type == "expense" {
					monthSummary.Weeks[i].ExpenseTotal += transaction.Amount
				}
				found = true
				break
			}
		}
		if !found {
			weekSummary := WeekSummary{
				Week: week,
			}
			if transaction.Type == "income" {
				weekSummary.IncomeTotal = transaction.Amount
			} else if transaction.Type == "expense" {
				weekSummary.ExpenseTotal = transaction.Amount
			}
			monthSummary.Weeks = append(monthSummary.Weeks, weekSummary)
		}

		yearSummary.Months[month] = monthSummary
		summary[year] = yearSummary
	}

	return summary, nil
}

func (s *PersonalTransactionService) CreateTransaction(transaction *domain.PersonalTransaction) error {
	transaction.ID = uuid.NewString()
	transaction.RoundToTwoDecimalPlaces()
	if err := transaction.Validate(); err != nil {
		return err
	}

	exists, err := s.categoryService.DoesPredefinedCategoryExist(transaction.PredefinedCategoryID)
	if err != nil {
		return err
	}
	if !exists {
		return financeErrors.ErrInvalidPredefinedCategory
	}
	if transaction.UserCategoryID != nil {
		exists, err = s.categoryService.DoesUserCategoryExist(*transaction.UserCategoryID, transaction.UserID)
		if err != nil {
			return err
		}
		if !exists {
			return financeErrors.ErrInvalidUserCategory
		}
	}

	exists, err = s.paymentService.DoesPaymentMethodExistByID(transaction.PaymentMethodID)
	if err != nil {
		return err
	}
	if !exists {
		return financeErrors.ErrInvalidPaymentMethod
	}
	if transaction.PaymentSourceID != nil {
		exists, err = s.paymentService.DoesUserPaymentSourceExistByID(*transaction.PaymentSourceID, transaction.UserID)
		if err != nil {
			return err
		}
		if !exists {
			return financeErrors.ErrInvalidUserCategory
		}
	}

	return s.repo.Save(*transaction)
}

func (s *PersonalTransactionService) CreateTransactionsBulk(transactions []*domain.PersonalTransaction, userID string) error {
	predefinedCategories, err := s.categoryService.GetAllPredefinedCategories("")
	if err != nil {
		return err
	}

	userCategories, err := s.categoryService.GetAllUserCategories(userID)
	if err != nil {
		return err
	}

	predefinedCategoryMap := make(map[int]bool)
	userCategoryMap := make(map[int]bool)

	for _, category := range predefinedCategories {
		predefinedCategoryMap[category.ID] = true
	}
	for _, category := range userCategories {
		userCategoryMap[category.ID] = true
	}

	paymentMethods, err := s.paymentService.GetAllPaymentMethods()
	if err != nil {
		return err
	}

	paymentUserSource, err := s.paymentService.GetUserPaymentSources(userID)
	if err != nil {
		return err
	}

	paymentMethodsMap := make(map[int]bool)
	paymentSourceMap := make(map[int]bool)

	for _, method := range paymentMethods {
		paymentMethodsMap[method.ID] = true
	}
	for _, source := range paymentUserSource {
		paymentSourceMap[source.ID] = true
	}

	tx, err := s.repo.BeginTransaction()
	if err != nil {
		return err
	}
	var validationErrors = &financeErrors.ValidationErrors{}
	defer func() {
		if p := recover(); p != nil {
			safeRollback(tx)
			panic(p)
		} else if err != nil {
			safeRollback(tx)
		} else {
			err = tx.Commit()
		}
	}()

	for i, transaction := range transactions {
		transaction.ID = uuid.NewString()
		transaction.RoundToTwoDecimalPlaces()
		transaction.UserID = userID
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
		}
		if _, exists := paymentMethodsMap[transaction.PaymentMethodID]; !exists {
			validationErrors.Add(financeErrors.NewIndexedValidationError(i+1, financeErrors.ErrInvalidPaymentMethod.Error()))
			continue
		}
		if transaction.PaymentSourceID != nil {
			if _, exists := paymentSourceMap[*transaction.PaymentSourceID]; !exists {
				validationErrors.Add(financeErrors.NewIndexedValidationError(i+1, financeErrors.ErrInvalidPaymentSource.Error()))
				continue
			}
		}
		if err := s.repo.SaveWithTransaction(*transaction, tx); err != nil {
			return fmt.Errorf("database error at transaction %d: %w", i+1, err)
		}

	}

	if len(validationErrors.Errors) > 0 {
		safeRollback(tx)
		return validationErrors
	}
	return nil
}
func safeRollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil {
		log.Printf("Error during transaction rollback: %v", err)
	}
}

func (s *PersonalTransactionService) GetUserTransactions(userID, transactionType string, startDate, endDate time.Time, limit, page int) ([]domain.PersonalTransaction, error) {
	transactions, err := s.repo.GetTransactionsByType(userID, transactionType, startDate, endDate, limit, page)
	if err != nil {
		return nil, err
	}
	if transactions == nil {
		return []domain.PersonalTransaction{}, nil
	}
	return transactions, nil
}

func (s *PersonalTransactionService) UpdateTransaction(transaction domain.PersonalTransaction) error {
	return s.repo.Update(transaction)
}

func (s *PersonalTransactionService) DeleteTransaction(transactionID int) error {
	return s.repo.Delete(transactionID)
}

func (s *PersonalTransactionService) GetTransactionSummaryByCategory(userID string, startDate, endDate time.Time, transactionType string) ([]domain.TransactionByCategorySummary, error) {
	transactions, err := s.repo.GetTransactionSummaryByCategory(userID, startDate, endDate, transactionType)
	if err != nil {
		return nil, err
	}

	return transactions, nil
}
