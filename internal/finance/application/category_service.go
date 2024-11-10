package application

import "github.com/sebuszqo/FinanceManager/internal/finance/domain"

type CategoryService struct {
	repo domain.CategoryRepository
}

func NewCategoryService(repo domain.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) DoesPredefinedCategoryExist(categoryID int) (bool, error) {
	return s.repo.DoesPredefinedCategoryExistByID(categoryID)
}

func (s *CategoryService) DoesUserCategoryExist(categoryID int, userID string) (bool, error) {
	return s.repo.DoesUserCategoryExistByID(categoryID, userID)
}

func (s *CategoryService) GetAllPredefinedCategories(categoryType string) ([]domain.PredefinedCategory, error) {
	return s.repo.FindPredefinedCategories(categoryType)
}

func (s *CategoryService) GetAllUserCategories(userID string) ([]domain.UserCategory, error) {
	return s.repo.FindUserCategories(userID)
}
