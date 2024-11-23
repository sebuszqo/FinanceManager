package marketdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sebuszqo/FinanceManager/internal/investment/models"
	"net/http"
	"net/url"
	"time"
)

type FinancialModelingPrepClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewFMPClient(apiKey string) *FinancialModelingPrepClient {
	return &FinancialModelingPrepClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *FinancialModelingPrepClient) GetCurrentPrice(ticker string) (float64, error) {
	return 2.33, nil
}

func (c *FinancialModelingPrepClient) VerifyTicker(ticker, exchange, currency string) (*models.VerifiedTicker, error) {
	baseURL := "https://financialmodelingprep.com/api/v3/search-ticker"
	params := fmt.Sprintf("?query=%s&limit=1&exchange=%s&apikey=%s", ticker, exchange, c.apiKey)
	fullURL := baseURL + params
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return nil, errors.New("invalid URL")
	}
	resp, err := http.Get(parsedURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error querying API: %s", resp.Status)
	}

	var results []struct {
		Symbol            string `json:"symbol"`
		Name              string `json:"name"`
		Currency          string `json:"currency"`
		StockExchange     string `json:"stockExchange"`
		ExchangeShortName string `json:"exchangeShortName"`
	}
	err = json.NewDecoder(resp.Body).Decode(&results)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("ticker %s not found on exchange %s with currency %s", ticker, exchange, currency)
	}

	var verifiedTicker models.VerifiedTicker
	for _, tickerEntry := range results {
		if tickerEntry.Symbol == ticker && tickerEntry.ExchangeShortName == exchange && tickerEntry.Currency == currency {
			verifiedTicker = models.VerifiedTicker{
				Ticker:    tickerEntry.Symbol,
				Name:      tickerEntry.Name,
				AssetType: "",
				Currency:  tickerEntry.Currency,
				Exchange:  exchange,
			}
			return &verifiedTicker, nil
		}
	}
	return nil, fmt.Errorf("ticker %s not found on exchange %s with currency %s", ticker, exchange, currency)

}

func (c *FinancialModelingPrepClient) FetchAllInstruments() (*[]models.InstrumentDTO, error) {
	url := fmt.Sprintf("https://financialmodelingprep.com/api/v3/stock/list?apikey=%s", c.apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error querying API: %s", resp.Status)
	}

	var results []models.InstrumentDTO
	err = json.NewDecoder(resp.Body).Decode(&results)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	return &results, nil
}
