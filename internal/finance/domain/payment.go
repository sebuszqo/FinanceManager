package domain

type PaymentMethod struct {
	ID   int
	Name string
}

type PaymentSource struct {
	ID              int
	UserID          string
	PaymentMethodID int
	Name            string
	Details         map[string]string // e.g. account number
}

type PaymentRepository interface {
	FindAllMethods() ([]PaymentMethod, error)
	FindUserSources(userID string) ([]PaymentSource, error)
}
