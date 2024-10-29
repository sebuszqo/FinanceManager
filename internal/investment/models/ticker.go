package models

import "time"

type Ticker struct {
	Ticker    string
	Name      string
	AssetType string
}

type VerifiedTicker struct {
	Ticker         string
	Name           string
	AssetType      string
	Exchange       string // Short exchange name, np. "NYSE", "GPW"
	Currency       string // Currency of given asset, np. "USD", "PLN"
	LastVerifiedAt time.Time
}
