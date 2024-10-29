package transactions

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/sebuszqo/FinanceManager/internal/investment/models"
)

type TransactionRepository interface {
	getTransactionTypes(ctx context.Context) ([]TransactionType, error)
	create(ctx context.Context, transaction *models.Transaction) error
	getTransactionsByAsset(ctx context.Context, assetID uuid.UUID) ([]models.Transaction, error)
}

type transactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) getTransactionTypes(ctx context.Context) ([]TransactionType, error) {
	var types []TransactionType

	query := `SELECT id, type FROM asset_types`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var transactionType TransactionType
		if err := rows.Scan(&transactionType.ID, &transactionType.Type); err != nil {
			return nil, err
		}
		types = append(types, transactionType)
	}

	return types, nil
}

func (r *transactionRepository) create(ctx context.Context, transaction *models.Transaction) error {
	query := `
        INSERT INTO transactions (id, asset_id, transaction_type_id, quantity, price, transaction_date, dividend_amount, coupon_amount, created_at) 
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `
	_, err := r.db.ExecContext(ctx, query, transaction.ID, transaction.AssetID, transaction.TransactionTypeID,
		transaction.Quantity, transaction.Price, transaction.TransactionDate, transaction.DividendAmount,
		transaction.CouponAmount, transaction.CreatedAt)
	return err
}

func (r *transactionRepository) getTransactionsByAsset(ctx context.Context, assetID uuid.UUID) ([]models.Transaction, error) {
	query := `SELECT id, asset_id, transaction_type_id, quantity, price, transaction_date, dividend_amount, coupon_amount, created_at 
              FROM transactions WHERE asset_id = $1 ORDER BY transaction_date DESC`
	rows, err := r.db.QueryContext(ctx, query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var t models.Transaction
		err := rows.Scan(&t.ID, &t.AssetID, &t.TransactionTypeID, &t.Quantity, &t.Price, &t.TransactionDate, &t.DividendAmount, &t.CouponAmount, &t.CreatedAt)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}

func (r *transactionRepository) getTransactionsByAssetID(ctx context.Context, assetID uuid.UUID) ([]models.Transaction, error) {
	query := `
		SELECT id, asset_id, transaction_type_id, quantity, price, transaction_date, dividend_amount, coupon_amount
		FROM transactions
		WHERE asset_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var t models.Transaction
		err := rows.Scan(&t.ID, &t.AssetID, &t.TransactionTypeID, &t.Quantity, &t.Price, &t.TransactionDate, &t.DividendAmount, &t.CouponAmount)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	return transactions, nil
}
