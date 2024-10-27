package assets

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type Asset struct {
	ID                   uuid.UUID
	PortfolioID          uuid.UUID
	Name                 string
	Ticker               string
	AssetTypeID          int
	CouponRate           float64
	MaturityDate         *time.Time
	FaceValue            float64
	DividendYield        float64
	Accumulation         bool
	TotalQuantity        float64
	AveragePurchasePrice float64
	TotalInvested        float64
	CurrentValue         float64
	UnrealizedGainLoss   float64
	//RealizedGainLoss float64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AssetRepository interface {
	doesAssetExist(ctx context.Context, portfolioID uuid.UUID, assetName string) (bool, error)
	getAssetByID(ctx context.Context, assetID uuid.UUID) (*Asset, error)
	deleteAsset(ctx context.Context, assetID uuid.UUID) error
	createAsset(ctx context.Context, asset *Asset) error
	getAssetTypes(ctx context.Context) ([]AssetType, error)
	findByPortfolioID(ctx context.Context, portfolioID uuid.UUID, assets *[]Asset) error
	findAllByPortfolioID(ctx context.Context, portfolioID uuid.UUID) ([]Asset, error)
	doesAssetBelongToUser(ctx context.Context, assetID, portfolioID uuid.UUID, userID string) (bool, error)
	updateAsset(ctx context.Context, asset *Asset) error
}

type assetRepository struct {
	db *sql.DB
}

func NewAssetRepository(db *sql.DB) AssetRepository {
	return &assetRepository{db: db}
}

func (a *assetRepository) getAssetTypes(ctx context.Context) ([]AssetType, error) {
	var types []AssetType

	query := `SELECT id, type FROM asset_types`
	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var assetType AssetType
		if err := rows.Scan(&assetType.ID, &assetType.Type); err != nil {
			return nil, err
		}
		types = append(types, assetType)
	}

	return types, nil
}

func (a *assetRepository) findByPortfolioID(ctx context.Context, portfolioID uuid.UUID, assets *[]Asset) error {
	query := `SELECT id, portfolio_id, name, ticker, asset_type_id, coupon_rate, maturity_date, face_value, dividend_yield, accumulation, total_quantity, average_purchase_price, total_invested, unrealized_gain_loss, current_value ,created_at, updated_at 
              FROM assets WHERE portfolio_id = $1`
	rows, err := a.db.QueryContext(ctx, query, portfolioID)
	if err != nil {
		return err
	}

	defer rows.Close()
	for rows.Next() {
		var asset Asset
		if err := rows.Scan(&asset.ID,
			&asset.PortfolioID,
			&asset.Name,
			&asset.Ticker,
			&asset.AssetTypeID,
			&asset.CouponRate,
			&asset.MaturityDate,
			&asset.FaceValue,
			&asset.DividendYield,
			&asset.Accumulation,
			&asset.TotalQuantity,
			&asset.AveragePurchasePrice,
			&asset.TotalInvested,
			&asset.UnrealizedGainLoss,
			&asset.CurrentValue,
			&asset.CreatedAt, &asset.UpdatedAt); err != nil {
			return err
		}
		*assets = append(*assets, asset)
	}
	return nil

}

// Repository layer function for inserting a new asset into the database
func (a *assetRepository) createAsset(ctx context.Context, asset *Asset) error {
	query := `
        INSERT INTO assets (id, portfolio_id, name, ticker, asset_type_id, coupon_rate, maturity_date, face_value, dividend_yield, accumulation, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `

	_, err := a.db.ExecContext(ctx, query,
		asset.ID,
		asset.PortfolioID,
		asset.Name,
		asset.Ticker,
		asset.AssetTypeID,
		asset.CouponRate,
		asset.MaturityDate,
		asset.FaceValue,
		asset.DividendYield,
		asset.Accumulation,
		asset.CreatedAt,
		asset.UpdatedAt,
	)

	return err
}

func (a *assetRepository) findAllByPortfolioID(ctx context.Context, portfolioID uuid.UUID) ([]Asset, error) {
	//TODO implement me
	panic("implement me")
}

func (a *assetRepository) getAssetByID(ctx context.Context, assetID uuid.UUID) (*Asset, error) {
	query := `SELECT name, ticker, asset_type_id, coupon_rate, maturity_date, face_value, dividend_yield, accumulation, created_at, updated_at  from assets WHERE id = $1`
	asset := &Asset{}
	err := a.db.QueryRowContext(ctx, query, assetID).Scan(&asset.Name, &asset.Ticker, &asset.AssetTypeID, &asset.CouponRate, &asset.MaturityDate, &asset.FaceValue, &asset.DividendYield, &asset.Accumulation, &asset.CreatedAt, &asset.UpdatedAt)
	return asset, err
}

func (a *assetRepository) updateAsset(ctx context.Context, asset *Asset) error {
	fmt.Println("Gain Loss", asset.UnrealizedGainLoss)
	fmt.Println("Valeu current", asset.CurrentValue)
	query := `
        UPDATE assets
        SET
            total_quantity = $1,
            average_purchase_price = $2,
            total_invested = $3,
            current_value = $4,
            unrealized_gain_loss = $5,
            updated_at = NOW()
        WHERE id = $6
    `
	_, err := a.db.ExecContext(ctx, query,
		asset.TotalQuantity,
		asset.AveragePurchasePrice,
		asset.TotalInvested,
		asset.CurrentValue,
		asset.UnrealizedGainLoss,
		asset.ID,
	)
	return err
}

func (a *assetRepository) deleteAsset(ctx context.Context, assetID uuid.UUID) error {
	query := `
		DELETE FROM assets 
		WHERE id = $1
	`
	_, err := a.db.ExecContext(ctx, query, assetID)
	return err
}

func (a *assetRepository) doesAssetExist(ctx context.Context, portfolioID uuid.UUID, assetName string) (bool, error) {
	query := `SELECT COUNT(1) FROM assets WHERE portfolio_id = $1 AND name = $2`
	var count int
	err := a.db.QueryRowContext(ctx, query, portfolioID, assetName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if asset exists: %w", err)
	}
	return count > 0, nil
}

// Repository function using JOIN to check ownership and asset existence
func (a *assetRepository) doesAssetBelongToUser(ctx context.Context, assetID, portfolioID uuid.UUID, userID string) (bool, error) {
	query := `
		SELECT COUNT(1)
		FROM portfolios p
		JOIN assets a ON a.portfolio_id = p.id
		WHERE p.id = $1 AND p.user_id = $2 AND a.id = $3
	`
	var count int
	err := a.db.QueryRowContext(ctx, query, portfolioID, userID, assetID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check asset and portfolio ownership: %w", err)
	}
	return count > 0, nil
}
