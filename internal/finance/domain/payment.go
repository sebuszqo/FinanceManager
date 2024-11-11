package domain

type PaymentMethod struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type PaymentSource struct {
	ID              int
	UserID          string
	PaymentMethodID int
	Name            string
	Details         map[string]string // e.g. account number
}

type PaymentRepository interface {
	FindAllPaymentMethods() ([]PaymentMethod, error)
	FindUserSources(userID string) ([]PaymentSource, error)
}
