package models

type Instrument struct {
	ID            int
	Symbol        string
	Name          string
	Exchange      string
	ExchangeShort string
	AssetTypeID   int
	Price         float64
	Currency      string
}

type InstrumentDTO struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Exchange      string  `json:"exchange"`
	ExchangeShort string  `json:"exchangeShortName"`
	Type          string  `json:"type"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
}

type InstrumentPriceWithSymbol struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
}
