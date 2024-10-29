package assets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/sebuszqo/FinanceManager/internal/investment/models"
	"log"
	"strings"
	"sync"
	"time"
)

var (
	ErrAssetsNotFound = errors.New("no assets in this portfolio")
	ErrAssetNotFound  = errors.New("asset doesn't exist in this portfolio")
	ErrNotValidTicker = errors.New("ticker of your asset is not valid")
)

type Service interface {
	CreateAsset(ctx context.Context, asset *Asset) error
	ListByPortfolioID(ctx context.Context, portfolioID uuid.UUID) ([]Asset, error)
	GetAssetTypes() []AssetType
	IsValidAssetType(assetTypeID int) bool
	DeleteAsset(ctx context.Context, assetID uuid.UUID) error
	GetAllAssets(ctx context.Context, portfolioID uuid.UUID) ([]Asset, error)
	DoesAssetExist(ctx context.Context, portfolioID uuid.UUID, assetName, ticker string) (bool, error)
	CheckAssetOwnership(ctx context.Context, assetID, portfolioID uuid.UUID, userID string) (bool, error)
	GetAssetByID(ctx context.Context, assetID uuid.UUID) (*Asset, error)
	GetAssetTypeName(assetTypeID int) string
	UpdateAssetAggregates(ctx context.Context, assetID uuid.UUID) error
	UpdateAssetPricing(ctx context.Context) error
}

type MarketDataService interface {
	GetCurrentPrice(ticker string) (float64, error)
	VerifyTicker(ticker, exchange, currency string) (*models.VerifiedTicker, error)
}

type Client interface {
	FetchBatchPrices(ctx context.Context, tickers []string) (map[string]float64, error)
}

type TransactionService interface {
	GetAllTransactions(ctx context.Context, assetID uuid.UUID) ([]models.Transaction, error)
}

type InstrumentService interface {
	GetTickerWithPriceInstruments(ctx context.Context) ([]models.InstrumentPriceWithSymbol, error)
	GetInstrumentPrice(ctx context.Context, symbol string) (float64, error)
}

type AssetType struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

type service struct {
	assetRepo          AssetRepository
	transactionService TransactionService
	instrumentService  InstrumentService
	marketDataSvc      MarketDataService
	assetTypeCache     map[int]string
	mu                 sync.RWMutex
}

func NewAssetService(repo AssetRepository, transactionService TransactionService, marketDataSvc MarketDataService, instrumentService InstrumentService) Service {
	service := &service{
		assetRepo:          repo,
		transactionService: transactionService,
		marketDataSvc:      marketDataSvc,
		instrumentService:  instrumentService,
		assetTypeCache:     make(map[int]string),
	}

	// Load asset types into cache when the service is created.
	if err := service.loadAssetTypesIntoCache(context.Background()); err != nil {
		log.Fatalf("Failed to load asset types into cache: %v", err)
	}

	go service.startPeriodicCacheRefresh(10 * time.Minute)
	return service
}

func (s *service) startPeriodicCacheRefresh(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			if err := s.loadAssetTypesIntoCache(context.Background()); err != nil {
				log.Println("Failed to refresh asset types cache:", err)
			} else {
				log.Println("Asset types cache refreshed")
			}
		}
	}
}

func (s *service) loadAssetTypesIntoCache(ctx context.Context) error {
	assetTypes, err := s.assetRepo.getAssetTypes(ctx)
	if err != nil {
		return err
	}

	// Lock for writing to the cache.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store asset types in the cache.
	for _, assetType := range assetTypes {
		s.assetTypeCache[assetType.ID] = assetType.Type
	}
	return nil
}

func (s *service) GetAssetTypes() []AssetType {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert cache map back to slice
	assetTypes := make([]AssetType, 0, len(s.assetTypeCache))
	for id, t := range s.assetTypeCache {
		assetTypes = append(assetTypes, AssetType{ID: id, Type: t})
	}
	return assetTypes
}

func (s *service) IsValidAssetType(assetTypeID int) bool {
	s.mu.RLock() // Use read lock for thread-safe access to cache
	defer s.mu.RUnlock()

	_, exists := s.assetTypeCache[assetTypeID]
	return exists
}

func (s *service) DoesAssetExist(ctx context.Context, portfolioID uuid.UUID, assetName, ticker string) (bool, error) {
	exists, err := s.assetRepo.doesAssetExist(ctx, portfolioID, assetName, ticker)
	if err != nil {
		return false, fmt.Errorf("failed to check asset existence: %w", err)
	}
	return exists, nil
}

// Service layer function for creating an asset
func (s *service) CreateAsset(ctx context.Context, asset *Asset) error {
	//_, exists := s.assetTypeCache[asset.AssetTypeID]
	if s.assetTypeCache[asset.AssetTypeID] == "Stock" || s.assetTypeCache[asset.AssetTypeID] == "ETF" {
		tickerInfo, err := s.assetRepo.verifyTicker(ctx, asset.Ticker)
		if err == nil {
			fmt.Println("Cache was used to get asset data")
			asset.Name = tickerInfo.Name
		} else {
			verifiedTicker, err := s.marketDataSvc.VerifyTicker(asset.Ticker, asset.Exchange, asset.Currency)
			if err != nil {
				return ErrNotValidTicker
			}
			if verifiedTicker == nil {
				return ErrNotValidTicker
			}
			err = s.assetRepo.addVerifiedTicker(ctx, *verifiedTicker)
			if err != nil {
				return err
			}
			asset.Name = verifiedTicker.Name
			asset.Currency = verifiedTicker.Currency
			asset.Exchange = verifiedTicker.Exchange
		}
	}

	// Call the repository to save the asset in the database
	err := s.assetRepo.createAsset(ctx, asset)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) ListByPortfolioID(ctx context.Context, portfolioID uuid.UUID) ([]Asset, error) {
	//return s.assetRepo.FindByPortfolioID(ctx, portfolioID)
	return nil, nil
}

// Service function that checks if the asset belongs to the portfolio and if the user owns the portfolio
func (s *service) CheckAssetOwnership(ctx context.Context, assetID, portfolioID uuid.UUID, userID string) (bool, error) {
	// Call the repository to check ownership and existence
	exists, err := s.assetRepo.doesAssetBelongToUser(ctx, assetID, portfolioID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check asset and portfolio ownership: %w", err)
	}
	return exists, nil
}

func (s *service) DeleteAsset(ctx context.Context, assetID uuid.UUID) error {
	err := s.assetRepo.deleteAsset(ctx, assetID)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) GetAllAssets(ctx context.Context, portfolioID uuid.UUID) ([]Asset, error) {
	assets := &[]Asset{}
	// check error when there is nothing under given portfolio :D
	err := s.assetRepo.findByPortfolioID(ctx, portfolioID, assets)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err

	}
	return *assets, nil
}

func (s *service) GetAssetByID(ctx context.Context, assetID uuid.UUID) (*Asset, error) {
	// Call the repository to get the asset
	asset, err := s.assetRepo.getAssetByID(ctx, assetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAssetsNotFound
		}
		return nil, err
	}
	return asset, nil
}

func (s *service) GetAssetTypeName(assetTypeID int) string {
	return s.assetTypeCache[assetTypeID]
}

func (s *service) UpdateAssetAggregates(ctx context.Context, assetID uuid.UUID) error {
	// Step 1: Fetch all transactions for the asset
	transactions, err := s.transactionService.GetAllTransactions(ctx, assetID)
	if err != nil {
		return err
	}

	// Step 2: Fetch the asset to get details
	asset, err := s.assetRepo.getAssetByID(ctx, assetID)
	if err != nil {
		return err
	}

	// Step 3: Initialize variables
	var totalQuantity, totalInvested, realizedGainLoss float64
	assetType := s.assetTypeCache[asset.AssetTypeID]
	// Step 4: Loop through transactions and calculate aggregates
	for _, t := range transactions {
		switch t.TransactionTypeID {
		// Buy
		case 1:
			if assetType == "ETF" || assetType == "Stock" || assetType == "Cryptocurrency" {
				totalQuantity += t.Quantity
				totalInvested += t.Quantity * t.Price
			} else if assetType == "Bond" {
				totalQuantity += t.Quantity
				totalInvested += t.Quantity * asset.FaceValue
			} else {
				totalQuantity += t.Quantity
				totalInvested += t.Quantity * t.Price
			}
		// Sell
		case 2:
			if totalQuantity >= t.Quantity {
				// Calculate average purchase price before this sale
				var averagePurchasePrice float64
				if totalQuantity > 0 {
					averagePurchasePrice = totalInvested / totalQuantity
				} else {
					averagePurchasePrice = 0
				}

				totalQuantity -= t.Quantity

				switch assetType {
				case "ETF", "Stock", "Cryptocurrency":
					totalInvested -= averagePurchasePrice * t.Quantity

					gainLoss := (t.Price - averagePurchasePrice) * t.Quantity
					realizedGainLoss += gainLoss

				case "Bond":
					totalInvested -= asset.FaceValue * t.Quantity
					gainLoss := (t.Price - asset.FaceValue) * t.Quantity
					realizedGainLoss += gainLoss

				default:
					totalInvested -= averagePurchasePrice * t.Quantity
					gainLoss := (t.Price - averagePurchasePrice) * t.Quantity
					realizedGainLoss += gainLoss
				}
			} else {
				// Handle selling more than owned, possibly return an error
				return fmt.Errorf("attempting to sell more than owned")
			}

		// Other cases (Dividends, Coupon Payments, etc.)
		default:
			continue
		}
	}

	// Step 5: Calculate average purchase price
	var averagePurchasePrice float64
	if totalQuantity > 0 {
		averagePurchasePrice = totalInvested / totalQuantity
	} else {
		averagePurchasePrice = 0
		totalInvested = 0
	}

	// Step 6: Calculate current value and unrealized gain/loss
	var currentValue, unrealizedGainLoss float64

	if assetType == "ETF" || assetType == "Stock" || assetType == "Cryptocurrency" {
		// Fetch current market price
		currentMarketPrice, err := s.instrumentService.GetInstrumentPrice(ctx, asset.Ticker)
		if err != nil {
			return err
		}
		currentValue = totalQuantity * currentMarketPrice
		unrealizedGainLoss = currentValue - totalInvested
	} else if assetType == "Bond" {
		// For bonds, use face value and accrued interest
		currentValue = totalQuantity*asset.FaceValue + asset.InterestAccrued
		unrealizedGainLoss = currentValue - totalInvested
	} else {
		// For other assets, implement appropriate logic
		currentValue = totalQuantity * averagePurchasePrice
		unrealizedGainLoss = currentValue - totalInvested
	}

	// Step 7: Update the asset record
	updatedAsset := &Asset{
		ID:                   assetID,
		TotalQuantity:        totalQuantity,
		AveragePurchasePrice: averagePurchasePrice,
		TotalInvested:        totalInvested,
		CurrentValue:         currentValue,
		UnrealizedGainLoss:   unrealizedGainLoss,
		UpdatedAt:            time.Now(),
	}

	err = s.assetRepo.updateAsset(ctx, updatedAsset)
	if err != nil {
		return err
	}

	return nil

}

func (s *service) UpdateAssetPricing(ctx context.Context) error {
	assets, err := s.assetRepo.getAllAssets(ctx)
	if err != nil {
		return err
	}
	if len(assets) == 0 {
		log.Println("No assets to be updated")
		return nil
	}

	instruments, err := s.instrumentService.GetTickerWithPriceInstruments(ctx)
	if err != nil {
		log.Printf("Błąd podczas pobierania instrumentów: %v", err)
		return err
	}

	priceMap := make(map[string]float64, len(instruments))
	for _, instrument := range instruments {
		priceMap[strings.ToUpper(instrument.Symbol)] = instrument.Price
	}

	updatedAssets := make([]Asset, 0, len(assets))
	var mu sync.Mutex
	var wg sync.WaitGroup

	maxGoroutines := 10
	sem := make(chan struct{}, maxGoroutines)

	for _, asset := range assets {
		wg.Add(1)
		sem <- struct{}{}
		go func(a Asset) {
			defer wg.Done()
			defer func() { <-sem }()

			assetType, exists := s.assetTypeCache[a.AssetTypeID]
			if !exists {
				log.Printf("Asset type not know for AssetTypeID: %s", a.AssetTypeID)
				return
			}

			switch assetType {
			case "Bond":
				annualCouponRate := a.CouponRate
				dailyInterestRate := annualCouponRate / 100 / 365
				interestAccrued := a.FaceValue * dailyInterestRate * a.TotalQuantity
				a.InterestAccrued += interestAccrued
				a.CurrentValue = a.FaceValue*a.TotalQuantity + a.InterestAccrued
				a.UnrealizedGainLoss = a.CurrentValue - a.TotalInvested
				mu.Lock()
				updatedAssets = append(updatedAssets, a)
				mu.Unlock()
			case "Stock", "ETF":
				updatedPrice, exists := priceMap[a.Ticker]
				if !exists {
					log.Printf("No price for this ticker: %s", a.Ticker)
					return
				}

				currentValue := a.TotalQuantity * updatedPrice
				unrealizedGainLoss := currentValue - a.TotalInvested

				a.CurrentValue = currentValue
				a.UnrealizedGainLoss = unrealizedGainLoss

				mu.Lock()
				updatedAssets = append(updatedAssets, a)
				mu.Unlock()

			default:
				log.Printf("Not supported type of asset: %s", assetType)
			}
		}(asset)
	}

	wg.Wait()

	if len(updatedAssets) == 0 {
		log.Println("No assets were updated")
		return nil
	}

	err = s.assetRepo.updateAssets(ctx, updatedAssets)
	if err != nil {
		log.Printf("Error during updating asset pricing %v", err)
		return err
	}

	log.Printf("Assets were updated sucessfully: %v.", len(updatedAssets))
	return nil
}
