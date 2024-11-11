package infrastructure

import (
	"database/sql"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) FindAllPaymentMethods() ([]domain.PaymentMethod, error) {
	query := "SELECT id, name FROM payment_methods"

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paymentMethods []domain.PaymentMethod
	for rows.Next() {
		var method domain.PaymentMethod
		if err := rows.Scan(&method.ID, &method.Name); err != nil {
			return nil, err
		}
		paymentMethods = append(paymentMethods, method)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return paymentMethods, nil
}

func (r *PaymentRepository) FindUserSources(userID string) ([]domain.PaymentSource, error) {
	panic("Implement me in the future :D")
}
