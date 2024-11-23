package infrastructure

import (
	"database/sql"
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	"time"
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
        (id, name, predefined_category_id, user_category_id, user_id, amount, type, date, description, payment_method_id, payment_source_id) 
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		transaction.ID, transaction.Name, transaction.PredefinedCategoryID, transaction.UserCategoryID, transaction.UserID, transaction.Amount,
		transaction.Type, transaction.Date, transaction.Description, transaction.PaymentMethodID, transaction.PaymentSourceID,
	)
	return err
}

func (r *PersonalTransactionRepository) GetTransactionsByType(userID string, transactionType string, startDate time.Time, endDate time.Time, limit int, page int) ([]domain.PersonalTransaction, error) {
	query := `
		SELECT id, name, user_id, amount, type, date, description, predefined_category_id, user_category_id, payment_method_id, payment_source_id 
		FROM personal_transactions 
		WHERE user_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date DESC LIMIT $4 OFFSET $5
		`

	args := []interface{}{userID, startDate, endDate}

	if transactionType != "" {
		query = `
		SELECT id, name, user_id, amount, type, date, description, predefined_category_id, user_category_id, payment_method_id, payment_source_id 
		FROM personal_transactions 
		WHERE user_id = $1 AND date >= $2 AND date <= $3 AND type = $4
		ORDER BY date DESC LIMIT $5 OFFSET $6
		`
		args = append(args, transactionType)
	}

	args = append(args, limit, (page-1)*limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []domain.PersonalTransaction
	for rows.Next() {
		var transaction domain.PersonalTransaction

		var userCategoryID sql.NullInt32
		var paymentSourceID sql.NullInt32

		if err := rows.Scan(
			&transaction.ID,
			&transaction.Name,
			&transaction.UserID,
			&transaction.Amount,
			&transaction.Type,
			&transaction.Date,
			&transaction.Description,
			&transaction.PredefinedCategoryID,
			&userCategoryID,
			&transaction.PaymentMethodID,
			&paymentSourceID,
		); err != nil {
			return nil, err
		}

		if userCategoryID.Valid {
			value := int(userCategoryID.Int32)
			transaction.UserCategoryID = &value
		} else {
			transaction.UserCategoryID = nil
		}

		if paymentSourceID.Valid {
			value := int(paymentSourceID.Int32)
			transaction.PaymentSourceID = &value
		} else {
			transaction.PaymentSourceID = nil
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
		`INSERT INTO personal_transactions (id, name, predefined_category_id, user_category_id, user_id, amount, type, date, description, payment_method_id, payment_source_id) 
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		transaction.ID, transaction.Name, transaction.PredefinedCategoryID, transaction.UserCategoryID, transaction.UserID, transaction.Amount,
		transaction.Type, transaction.Date, transaction.Description, transaction.PaymentMethodID, transaction.PaymentSourceID,
	)
	return err
}

func (r *PersonalTransactionRepository) GetTransactionsInDateRange(userID string, startDate, endDate time.Time) ([]domain.PersonalTransaction, error) {
	rows, err := r.db.Query(`
			SELECT id, name, amount, date, type
			FROM personal_transactions
			WHERE user_id = $1 AND date >= $2 AND date <= $3
			ORDER BY date
		`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []domain.PersonalTransaction
	for rows.Next() {
		var transaction domain.PersonalTransaction
		if err := rows.Scan(&transaction.ID, &transaction.Name, &transaction.Amount, &transaction.Date, &transaction.Type); err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, nil
}

func (r *PersonalTransactionRepository) GetTransactionSummaryByCategory(userID string, startDate, endDate time.Time, transactionType string) ([]domain.TransactionByCategorySummary, error) {
	query := `
	SELECT c.name AS category_name, 
           t.predefined_category_id AS category_id, 
           SUM(t.amount) AS total_amount
	FROM personal_transactions t
	LEFT JOIN predefined_categories c ON t.predefined_category_id = c.id
	WHERE t.user_id = $1
	AND t.date >= $2
	AND t.date <= $3
	GROUP BY category_id, category_name ORDER BY total_amount DESC
	 `

	args := []interface{}{userID, startDate, endDate}

	if transactionType != "" {
		query = `
	SELECT c.name AS category_name, 
           t.predefined_category_id AS category_id, 
           SUM(t.amount) AS total_amount
	FROM personal_transactions t
	LEFT JOIN predefined_categories c ON t.predefined_category_id = c.id
	WHERE t.user_id = $1
	AND t.date >= $2
	AND t.date <= $3
	AND t.type = $4
	GROUP BY category_id, category_name ORDER BY total_amount DESC
	 `
		args = append(args, transactionType)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []domain.TransactionByCategorySummary
	for rows.Next() {
		var summary domain.TransactionByCategorySummary
		if err := rows.Scan(&summary.CategoryName, &summary.CategoryID, &summary.TotalAmount); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (r *PersonalTransactionRepository) GetTransactionSummaryByPaymentMethod(userID string, startDate time.Time, endDate time.Time, transactionType string) ([]domain.TransactionByPaymentMethodSummary, error) {
	query := `
	SELECT c.name AS method_name, 
           t.payment_method_id AS method_id, 
           SUM(t.amount) AS total_amount
	FROM personal_transactions t
	LEFT JOIN payment_methods c ON t.payment_method_id = c.id
	WHERE t.user_id = $1
	AND t.date >= $2
	AND t.date <= $3
	GROUP BY method_id, method_name  ORDER BY total_amount DESC
	 `

	args := []interface{}{userID, startDate, endDate}

	if transactionType != "" {
		query = `
	SELECT c.name AS method_name, 
           t.payment_method_id AS method_id, 
           SUM(t.amount) AS total_amount
	FROM personal_transactions t
	LEFT JOIN payment_methods c ON t.payment_method_id = c.id
	WHERE t.user_id = $1
	AND t.date >= $2
	AND t.date <= $3
	AND t.type = $4
	GROUP BY method_id, method_name  ORDER BY total_amount DESC
	 `
		args = append(args, transactionType)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []domain.TransactionByPaymentMethodSummary
	for rows.Next() {
		var summary domain.TransactionByPaymentMethodSummary
		if err := rows.Scan(&summary.PaymentMethodName, &summary.PaymentMethodID, &summary.TotalAmount); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
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
