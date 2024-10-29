package instrument

import (
	"context"
	"database/sql"
	"github.com/sebuszqo/FinanceManager/internal/investment/models"
	"time"
)

type Repository interface {
	bulkInsertOrUpdate(ctx context.Context, instruments *[]models.Instrument) error
	getPriceBySymbol(ctx context.Context, symbol string) (float64, error)
	searchByNameOrSymbol(ctx context.Context, query string, assetTypeID int, limit int) (*[]models.Instrument, error)
	getAllSymbols(ctx context.Context) ([]string, error)
	getLastUpdatedAt(ctx context.Context) (time.Time, error)
	getTickerWithPriceInstruments(ctx context.Context) ([]models.InstrumentPriceWithSymbol, error)
}

type instrumentRepository struct {
	db *sql.DB
}

func NewInstrumentRepository(db *sql.DB) Repository {
	return &instrumentRepository{db: db}
}

func (r *instrumentRepository) getLastUpdatedAt(ctx context.Context) (time.Time, error) {

	var lastUpdated sql.NullTime
	err := r.db.QueryRowContext(ctx, `
        SELECT MAX(updated_at) FROM instruments
    `).Scan(&lastUpdated)
	if err != nil {
		return time.Time{}, err
	}

	if !lastUpdated.Valid {
		return time.Time{}, nil
	}

	return lastUpdated.Time, nil
}

func (r *instrumentRepository) bulkInsertOrUpdate(ctx context.Context, instruments *[]models.Instrument) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO instruments (symbol, name, exchange, exchange_short, asset_type_id, price, currency, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
        ON CONFLICT (symbol, exchange_short) DO UPDATE SET
            name = EXCLUDED.name,
            exchange = EXCLUDED.exchange,
            asset_type_id = EXCLUDED.asset_type_id,
            price = EXCLUDED.price,
            currency = EXCLUDED.currency,
            updated_at = NOW();
    `)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, instr := range *instruments {
		_, err := stmt.ExecContext(ctx,
			instr.Symbol,
			instr.Name,
			instr.Exchange,
			instr.ExchangeShort,
			instr.AssetTypeID,
			instr.Price,
			instr.Currency,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (r *instrumentRepository) getPriceBySymbol(ctx context.Context, symbol string) (float64, error) {
	var price float64
	err := r.db.QueryRowContext(ctx, `
        SELECT price FROM instruments WHERE symbol = $1
    `, symbol).Scan(&price)
	if err != nil {
		return 0, err
	}
	return price, nil
}

func (r *instrumentRepository) searchByNameOrSymbol(ctx context.Context, query string, assetTypeID int, limit int) (*[]models.Instrument, error) {
	q := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, symbol, name, exchange, exchange_short, asset_type_id, price, currency
        FROM instruments
        WHERE asset_type_id = $1 AND (symbol ILIKE $2 OR name ILIKE $2)
        LIMIT $3
    `, assetTypeID, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instruments []models.Instrument
	for rows.Next() {
		var instr models.Instrument
		err := rows.Scan(
			&instr.ID,
			&instr.Symbol,
			&instr.Name,
			&instr.Exchange,
			&instr.ExchangeShort,
			&instr.AssetTypeID,
			&instr.Price,
			&instr.Currency,
		)
		if err != nil {
			return nil, err
		}
		instruments = append(instruments, instr)
	}
	return &instruments, nil
}

func (r *instrumentRepository) getAllSymbols(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT symbol FROM instruments`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		err := rows.Scan(&symbol)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}
	return symbols, nil
}

func (r *instrumentRepository) getTickerWithPriceInstruments(ctx context.Context) ([]models.InstrumentPriceWithSymbol, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT symbol, price
        FROM instruments
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instruments []models.InstrumentPriceWithSymbol
	for rows.Next() {
		var instr models.InstrumentPriceWithSymbol
		if err := rows.Scan(&instr.Symbol, &instr.Price); err != nil {
			return nil, err
		}
		instruments = append(instruments, instr)
	}

	return instruments, nil
}
