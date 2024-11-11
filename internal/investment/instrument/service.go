package instrument

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/sebuszqo/FinanceManager/internal/investment/models"
	"time"
)

type Service interface {
	ImportInstruments(ctx context.Context) error
	UpdateInstruments(ctx context.Context) error
	GetInstrumentPrice(ctx context.Context, symbol string) (float64, error)
	SearchInstruments(ctx context.Context, query string, assetTypeID int, limit int) (*[]models.Instrument, error)
	NeedsUpdate(ctx context.Context) (bool, error)
	GetTickerWithPriceInstruments(ctx context.Context) ([]models.InstrumentPriceWithSymbol, error)
}

type APIService interface {
	FetchAllInstruments() (*[]models.InstrumentDTO, error)
	//FetchInstrumentPrices(symbols []string) (map[string]float64, error)
}

type service struct {
	instrumentRepo Repository
	marketDataSvc  APIService
}

func (s *service) ImportInstruments(ctx context.Context) error {
	instrumentDTOs, err := s.marketDataSvc.FetchAllInstruments()
	if err != nil {
		return err
	}

	if instrumentDTOs == nil {
		return fmt.Errorf("error during instruments importing, instrument list is empty")
	}

	return s.importInstrumentDTOs(ctx, instrumentDTOs)
}

func (s *service) UpdateInstruments(ctx context.Context) error {
	return nil
}

func (s *service) GetInstrumentPrice(ctx context.Context, symbol string) (float64, error) {
	return s.instrumentRepo.getPriceBySymbol(ctx, symbol)
}

func (s *service) SearchInstruments(ctx context.Context, query string, assetTypeID int, limit int) (*[]models.Instrument, error) {
	return s.instrumentRepo.searchByNameOrSymbol(ctx, query, assetTypeID, limit)
}

func (s *service) importInstrumentDTOs(ctx context.Context, dtos *[]models.InstrumentDTO) error {
	var assetTypeID int
	var instruments []models.Instrument
	for _, dto := range *dtos {
		switch dto.Type {
		case "stock":
			assetTypeID = 1
		case "etf":
			fmt.Println("etf?")
			assetTypeID = 3
		default:
			fmt.Println("Wrong type of instrument. skipping")
			continue
		}

		instr := models.Instrument{
			Symbol:        dto.Symbol,
			Name:          dto.Name,
			Exchange:      dto.Exchange,
			ExchangeShort: dto.ExchangeShort,
			AssetTypeID:   assetTypeID,
			Price:         dto.Price,
			Currency:      dto.Currency,
		}
		instruments = append(instruments, instr)
	}

	return s.instrumentRepo.bulkInsertOrUpdate(ctx, &instruments)
}

func (s *service) NeedsUpdate(ctx context.Context) (bool, error) {
	lastUpdated, err := s.instrumentRepo.getLastUpdatedAt(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No data in the table, needs update
			return true, nil
		}
		return false, err
	}

	if time.Since(lastUpdated) > 6*time.Hour {
		return true, nil
	}
	return false, nil
}

func (s *service) GetTickerWithPriceInstruments(ctx context.Context) ([]models.InstrumentPriceWithSymbol, error) {
	instruments, err := s.instrumentRepo.getTickerWithPriceInstruments(ctx)
	if err != nil {
		return nil, err
	}

	return instruments, nil
}

func NewInstrumentService(repo Repository, marketDataSvc APIService) Service {
	return &service{instrumentRepo: repo, marketDataSvc: marketDataSvc}
}

func (s *service) GetCurrentInstrumentPrice(ticker string) (float64, error) {
	return 2.33, nil
}
