package models

import (
	"github.com/google/uuid"
	"time"
)

type Transaction struct {
	ID                uuid.UUID `json:"id"`
	AssetID           uuid.UUID `json:"asset_id"`
	TransactionTypeID int       `json:"transaction_type_id"`
	Quantity          float64   `json:"quantity"`
	Price             float64   `json:"price"`
	TransactionDate   time.Time `json:"transaction_date"`
	DividendAmount    *float64  `json:"dividend_amount,omitempty"`
	CouponAmount      *float64  `json:"coupon_amount,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}
