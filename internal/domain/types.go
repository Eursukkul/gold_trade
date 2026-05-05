// Package domain contains gold order validation rules without infrastructure concerns.
package domain

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// OrderType represents a supported gold order side.
type OrderType string

const (
	// OrderTypeBuy is a customer buy order.
	OrderTypeBuy OrderType = "buy"
	// OrderTypeSell is a customer sell order.
	OrderTypeSell OrderType = "sell"
)

// Order is the input contract for validating a gold trade.
type Order struct {
	CustomerID  string
	Type        OrderType
	Quantity    decimal.Decimal
	QuotedPrice decimal.Decimal
	TradeDate   time.Time
}

// MarketPrice is the current reference price in THB per baht-weight.
type MarketPrice struct {
	BaseSellPrice decimal.Decimal
	AsOf          time.Time
}

// ValidationResult is returned for both accepted and rejected orders.
type ValidationResult struct {
	Accepted            bool
	Violations          []Violation
	TotalAmount         decimal.Decimal
	ExpectedPrice       decimal.Decimal
	SpreadAmount        decimal.Decimal
	DailyRemaining      decimal.Decimal
	DailyRemainingAfter decimal.Decimal
}

// Violation is a machine-readable validation failure with a human-readable message.
type Violation struct {
	Code    string
	Message string
}

// AddViolation marks the result as rejected and appends a failure reason.
func (r *ValidationResult) AddViolation(code string, format string, args ...any) {
	r.Accepted = false
	r.Violations = append(r.Violations, Violation{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	})
}
