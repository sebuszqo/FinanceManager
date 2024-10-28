package marketdata

import (
	"context"
	"github.com/sebuszqo/FinanceManager/internal/investment/models"
)

type YahooFinanceService struct {
	// Fields specific to Yahoo Finance
}

type Client interface {
	VerifyTicker(ctx context.Context, ticker string) (*models.Ticker, error)
	FetchBatchPrices(ctx context.Context, tickers []string) (map[string]float64, error)
}

func (s *YahooFinanceService) GetCurrentPrice(ticker string) (float64, error) {
	return 2.33, nil
}
