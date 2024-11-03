package infrastructure

import (
	"database/sql"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
)

type PersonalTransactionRepository struct {
	db *sql.DB
}

func NewPersonalTransactionRepository(db *sql.DB) *PersonalTransactionRepository {
	return &PersonalTransactionRepository{db: db}
}

func (r *PersonalTransactionRepository) Save(transaction domain.PersonalTransaction) error {
	_, err := r.db.Exec(
		`INSERT INTO personal_transactions 
        (predefined_category_id, user_category_id, user_id, amount, type, date, description, payment_method_id, payment_source_id) 
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		transaction.PredefinedCategoryID, transaction.UserCategoryID, transaction.UserID, transaction.Amount,
		transaction.Type, transaction.Date, transaction.Description, transaction.PaymentMethodID, transaction.PaymentSourceID,
	)
	return err
}

func (r *PersonalTransactionRepository) FindByUser(userID string) ([]domain.PersonalTransaction, error) {
	rows, err := r.db.Query(`SELECT id, predefined_category_id, user_category_id, user_id, amount, type, date, description, payment_method_id, payment_source_id FROM personal_transactions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []domain.PersonalTransaction
	for rows.Next() {
		var transaction domain.PersonalTransaction
		if err := rows.Scan(&transaction.ID, &transaction.PredefinedCategoryID, &transaction.UserCategoryID, &transaction.UserID,
			&transaction.Amount, &transaction.Type, &transaction.Date, &transaction.Description, &transaction.PaymentMethodID, &transaction.PaymentSourceID); err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, nil
}

func (r *PersonalTransactionRepository) BeginTransaction() (*sql.Tx, error) {
	return r.db.Begin()
}

func (r *PersonalTransactionRepository) SaveWithTransaction(transaction domain.PersonalTransaction, tx *sql.Tx) error {
	_, err := tx.Exec(
		`INSERT INTO personal_transactions (predefined_category_id, user_category_id, user_id, amount, type, date, description, payment_method_id, payment_source_id) 
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		transaction.PredefinedCategoryID, transaction.UserCategoryID, transaction.UserID, transaction.Amount,
		transaction.Type, transaction.Date, transaction.Description, transaction.PaymentMethodID, transaction.PaymentSourceID,
	)
	return err
}

func (r *PersonalTransactionRepository) FindByID(transactionID int) (*domain.PersonalTransaction, error) {
	panic("Implement me")
}

func (r *PersonalTransactionRepository) Delete(transactionID int) error {
	panic("Implement me")
}
func (r *PersonalTransactionRepository) Update(transaction domain.PersonalTransaction) error {
	panic("Implement me")
}
