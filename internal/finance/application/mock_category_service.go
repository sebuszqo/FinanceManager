package application

import "github.com/sebuszqo/FinanceManager/internal/finance/domain"

type MockCategoryService struct{}

func (m *MockCategoryService) GetAllUserCategories(userID string) ([]domain.UserCategory, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockCategoryService) GetAllPredefinedCategories(categoryType string) ([]domain.PredefinedCategory, error) {
	return []domain.PredefinedCategory{}, nil
}

func (m *MockCategoryService) DoesPredefinedCategoryExist(id int) (bool, error) {
	return true, nil
}

func (m *MockCategoryService) DoesUserCategoryExist(id int, userID string) (bool, error) {
	return true, nil
}
