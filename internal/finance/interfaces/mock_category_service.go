package interfaces

import (
	"errors"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
)

type MockCategoryService struct {
	categories []domain.PredefinedCategory
	shouldFail bool
}

func (m *MockCategoryService) GetAllPredefinedCategories(categoryType string) ([]domain.PredefinedCategory, error) {
	if m.shouldFail {
		return nil, errors.New("service error")
	}
	return m.categories, nil
}
