package infrastructure

import (
	"database/sql"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
)

type CategoryRepository struct {
	db *sql.DB
}

func (r *CategoryRepository) DoesPredefinedCategoryExistByID(categoryID int) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (r *CategoryRepository) DoesUserCategoryExistByID(categoryID int, userID string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) doesPredefinedCategoryExistByID(categoryID int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM predefined_categories WHERE id = $1)"
	err := r.db.QueryRow(query, categoryID).Scan(&exists)
	return exists, err
}

func (r *CategoryRepository) doesUserCategoryExistByID(categoryID int, userID string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM user_categories WHERE id = $1 AND user_id = $2)"
	err := r.db.QueryRow(query, categoryID, userID).Scan(&exists)
	return exists, err
}

func (r *CategoryRepository) FindPredefinedCategories() ([]domain.PredefinedCategory, error) {
	rows, err := r.db.Query("SELECT id FROM predefined_categories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.PredefinedCategory
	for rows.Next() {
		var category domain.PredefinedCategory
		if err := rows.Scan(&category.ID); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}

func (r *CategoryRepository) FindUserCategories(userID string) ([]domain.UserCategory, error) {
	rows, err := r.db.Query("SELECT id FROM user_categories WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.UserCategory
	for rows.Next() {
		var category domain.UserCategory
		if err := rows.Scan(&category.ID); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}
