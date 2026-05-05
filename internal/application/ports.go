// Package application contains use cases that orchestrate domain rules and ports.
package application

import (
	"context"
	"time"

	"intergold-assessment/internal/domain"

	"github.com/shopspring/decimal"
)

// BalanceProvider returns available cash balance in THB.
type BalanceProvider interface {
	AvailableBalance(ctx context.Context, customerID string) (decimal.Decimal, error)
}

// MarketPriceProvider returns the latest market price.
type MarketPriceProvider interface {
	CurrentPrice(ctx context.Context) (domain.MarketPrice, error)
}

// DailyVolumeProvider returns already traded baht-weight for a customer and day.
type DailyVolumeProvider interface {
	TradedQuantity(ctx context.Context, customerID string, day time.Time) (decimal.Decimal, error)
}
