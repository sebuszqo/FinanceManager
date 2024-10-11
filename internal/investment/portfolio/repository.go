package portfolios

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"time"
)

type Portfolio struct {
	ID          uuid.UUID
	UserID      string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PortfolioDTO struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PortfolioRepository interface {
	Create(ctx context.Context, portfolio *Portfolio) error
	FindByID(ctx context.Context, portfolioID uuid.UUID, portfolio *Portfolio) error
	ExistsByName(ctx context.Context, userID string, name string) (bool, error)
	FindByUserID(ctx context.Context, userID string, portfolios *[]PortfolioDTO) error
	Update(ctx context.Context, portfolio *Portfolio) (int64, error)
	DeletePortfolio(ctx context.Context, portfolioID uuid.UUID) error
}

type portfolioRepository struct {
	db *sql.DB
}

func NewPortfolioRepository(db *sql.DB) PortfolioRepository {
	return &portfolioRepository{db: db}
}

func (r *portfolioRepository) ExistsByName(ctx context.Context, userID string, name string) (bool, error) {
	parsedID, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	query := `SELECT COUNT(1) 
              FROM portfolios 
              WHERE user_id = $1 AND name = $2`

	var count int
	err = r.db.QueryRowContext(ctx, query, parsedID, name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *portfolioRepository) Create(ctx context.Context, portfolio *Portfolio) error {
	query := `INSERT INTO portfolios (id, user_id, name, description, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, portfolio.ID, portfolio.UserID, portfolio.Name, portfolio.Description, portfolio.CreatedAt, portfolio.UpdatedAt)
	return err
}

func (r *portfolioRepository) FindByID(ctx context.Context, portfolioID uuid.UUID, portfolio *Portfolio) error {
	query := `SELECT id, user_id, name, description, created_at, updated_at 
              FROM portfolios WHERE id = $1`

	return r.db.QueryRowContext(ctx, query, portfolioID).Scan(
		&portfolio.ID, &portfolio.UserID, &portfolio.Name, &portfolio.Description, &portfolio.CreatedAt, &portfolio.UpdatedAt)
}

func (r *portfolioRepository) FindByUserID(ctx context.Context, userID string, portfolios *[]PortfolioDTO) error {
	query := `SELECT id, name, description, created_at, updated_at FROM portfolios WHERE user_id = $1`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var portfolio PortfolioDTO
		if err := rows.Scan(&portfolio.ID, &portfolio.Name, &portfolio.Description, &portfolio.CreatedAt, &portfolio.UpdatedAt); err != nil {
			return err
		}
		*portfolios = append(*portfolios, portfolio)
	}
	return nil
}

func (r *portfolioRepository) Update(ctx context.Context, portfolio *Portfolio) (int64, error) {
	query := `
        UPDATE portfolios 
        SET name = $1, description = $2, updated_at = $3 
        WHERE id = $4 AND user_id = $5
    `

	result, err := r.db.ExecContext(ctx, query, portfolio.Name, portfolio.Description, time.Now(), portfolio.ID, portfolio.UserID)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return affected, nil
}

func (r *portfolioRepository) DeletePortfolio(ctx context.Context, portfolioID uuid.UUID) error {
	query := `
        DELETE FROM portfolios 
        WHERE id = $1
    `
	_, err := r.db.ExecContext(ctx, query, portfolioID)
	return err
}
