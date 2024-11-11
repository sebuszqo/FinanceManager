package domain

type PredefinedCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "income" or "expense"
}

type UserCategory struct {
	ID     int
	Name   string
	UserID string // user UUID
}

type CategoryRepository interface {
	FindPredefinedCategories(categoryType string) ([]PredefinedCategory, error)
	FindUserCategories(userID string) ([]UserCategory, error)
	DoesPredefinedCategoryExistByID(categoryID int) (bool, error)
	DoesUserCategoryExistByID(categoryID int, userID string) (bool, error)
}
