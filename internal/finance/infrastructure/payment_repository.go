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

func (r *PaymentRepository) GetAllPaymentMethods() ([]domain.PaymentMethod, error) {
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

func (r *PaymentRepository) GetUserPaymentSources(userID string) ([]domain.PaymentSource, error) {
	query := "SELECT id, name FROM payment_sources where user_id = $1"

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paymentMethods []domain.PaymentSource
	for rows.Next() {
		var method domain.PaymentSource
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
func (r *PaymentRepository) PaymentMethodExists(methodID int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM payment_methods WHERE id = $1)"
	err := r.db.QueryRow(query, methodID).Scan(&exists)
	return exists, err
}
func (r *PaymentRepository) UserPaymentSourceExists(sourceID int, userID string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM payment_sources WHERE id = $1 AND user_id = $2)"
	err := r.db.QueryRow(query, sourceID, userID).Scan(&exists)
	return exists, err
}
