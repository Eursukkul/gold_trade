package application

import (
	"context"
	"fmt"
	"time"

	"intergold-assessment/internal/domain"

	"github.com/shopspring/decimal"
)

var (
	quantityStep      = decimal.RequireFromString("0.5")
	defaultTolerance  = decimal.RequireFromString("0.02")
	defaultSpread     = decimal.RequireFromString("0.005")
	defaultDailyLimit = decimal.RequireFromString("5")
	one               = decimal.NewFromInt(1)
	zero              = decimal.Zero
)

// MarketPrice is the current reference price in THB per baht-weight.
type MarketPrice struct {
	BaseSellPrice decimal.Decimal
	AsOf          time.Time
}

// Validator validates gold orders using injected data sources.
type Validator struct {
	balances BalanceProvider
	prices   MarketPriceProvider
	volumes  DailyVolumeProvider
	cfg      Config
}

// Config controls business thresholds for validation.
type Config struct {
	PriceTolerance decimal.Decimal
	BuySpreadRate  decimal.Decimal
	DailyLimit     decimal.Decimal
}

// NewValidator creates a Validator with conservative defaults if config fields are zero.
func NewValidator(balances BalanceProvider, prices MarketPriceProvider, volumes DailyVolumeProvider, cfg Config) *Validator {
	if cfg.PriceTolerance.IsZero() {
		cfg.PriceTolerance = defaultTolerance
	}
	if cfg.BuySpreadRate.IsZero() {
		cfg.BuySpreadRate = defaultSpread
	}
	if cfg.DailyLimit.IsZero() {
		cfg.DailyLimit = defaultDailyLimit
	}

	return &Validator{
		balances: balances,
		prices:   prices,
		volumes:  volumes,
		cfg:      cfg,
	}
}

// Validate checks an order and returns all validation failures found.
func (v *Validator) Validate(ctx context.Context, order domain.Order) (domain.ValidationResult, error) {
	result := domain.ValidationResult{Accepted: true}

	v.validateStaticFields(order, &result)
	if !result.Accepted {
		return result, nil
	}

	market, err := v.prices.CurrentPrice(ctx)
	if err != nil {
		return result, fmt.Errorf("load current market price: %w", err)
	}
	if !market.BaseSellPrice.GreaterThan(zero) {
		result.AddViolation("invalid_market_price", "current market price must be positive")
		return result, nil
	}

	v.validatePrice(order, market, &result)
	v.validateDailyLimit(ctx, order, &result)

	if order.Type == domain.OrderTypeBuy {
		v.validateBalance(ctx, order, &result)
	}

	return result, nil
}

func (v *Validator) validateStaticFields(order domain.Order, result *domain.ValidationResult) {
	if order.CustomerID == "" {
		result.AddViolation("missing_customer_id", "customer_id is required")
	}

	if order.Type != domain.OrderTypeBuy && order.Type != domain.OrderTypeSell {
		result.AddViolation("invalid_order_type", "order_type must be either buy or sell")
	}

	if !order.Quantity.GreaterThan(zero) {
		result.AddViolation("invalid_quantity", "quantity must be positive")
	} else if !isMultipleOf(order.Quantity, quantityStep) {
		result.AddViolation("invalid_quantity_step", "quantity must be a multiple of 0.5 baht-weight")
	}

	if !order.QuotedPrice.GreaterThan(zero) {
		result.AddViolation("invalid_price", "quoted_price must be positive")
	}
}

func (v *Validator) validatePrice(order domain.Order, market MarketPrice, result *domain.ValidationResult) {
	expectedPrice := market.BaseSellPrice
	if order.Type == domain.OrderTypeBuy {
		expectedPrice = market.BaseSellPrice.Mul(one.Add(v.cfg.BuySpreadRate))
		result.SpreadAmount = expectedPrice.Sub(market.BaseSellPrice)
	}

	result.ExpectedPrice = expectedPrice
	result.TotalAmount = order.Quantity.Mul(order.QuotedPrice)

	if !withinTolerance(order.QuotedPrice, expectedPrice, v.cfg.PriceTolerance) {
		result.AddViolation(
			"stale_or_inconsistent_price",
			"quoted_price %s is outside %s tolerance from expected price %s",
			order.QuotedPrice.StringFixedBank(2),
			v.cfg.PriceTolerance.Mul(decimal.NewFromInt(100)).String(),
			expectedPrice.StringFixedBank(2),
		)
	}
}

func (v *Validator) validateDailyLimit(ctx context.Context, order domain.Order, result *domain.ValidationResult) {
	tradeDate := order.TradeDate
	if tradeDate.IsZero() {
		tradeDate = time.Now().UTC()
	}

	used, err := v.volumes.TradedQuantity(ctx, order.CustomerID, tradeDate)
	if err != nil {
		result.AddViolation("daily_volume_unavailable", "could not load daily trading volume")
		return
	}

	remaining := v.cfg.DailyLimit.Sub(used)
	if remaining.LessThan(zero) {
		remaining = zero
	}
	result.DailyRemaining = remaining

	if order.Quantity.GreaterThan(remaining) {
		result.AddViolation(
			"daily_limit_exceeded",
			"daily trading limit exceeded; remaining allowance is %s baht-weight",
			remaining.String(),
		)
		return
	}

	result.DailyRemainingAfter = remaining.Sub(order.Quantity)
}

func (v *Validator) validateBalance(ctx context.Context, order domain.Order, result *domain.ValidationResult) {
	balance, err := v.balances.AvailableBalance(ctx, order.CustomerID)
	if err != nil {
		result.AddViolation("balance_unavailable", "could not load available balance")
		return
	}

	totalCost := order.Quantity.Mul(order.QuotedPrice)
	if balance.LessThan(totalCost) {
		result.AddViolation(
			"insufficient_balance",
			"available balance %s THB is less than required amount %s THB",
			balance.StringFixedBank(2),
			totalCost.StringFixedBank(2),
		)
	}
}

func isMultipleOf(value decimal.Decimal, step decimal.Decimal) bool {
	return value.Div(step).IsInteger()
}

func withinTolerance(actual decimal.Decimal, expected decimal.Decimal, tolerance decimal.Decimal) bool {
	if expected.IsZero() {
		return false
	}

	diff := actual.Sub(expected).Abs().Div(expected)
	return diff.LessThanOrEqual(tolerance)
}
