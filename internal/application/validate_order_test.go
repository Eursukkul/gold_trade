package application_test

import (
	"context"
	"testing"
	"time"

	"intergold-assessment/internal/application"
	"intergold-assessment/internal/domain"
	"intergold-assessment/internal/inmemory"

	"github.com/shopspring/decimal"
)

func TestValidatorValidate(t *testing.T) {
	t.Parallel()

	tradeDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		order      domain.Order
		balance    string
		usedDaily  string
		wantOK     bool
		wantCode   string
		wantSpread string
	}{
		{
			name: "valid buy includes spread and daily limit",
			order: domain.Order{
				CustomerID:  "C001",
				Type:        domain.OrderTypeBuy,
				Quantity:    dec("1.5"),
				QuotedPrice: dec("42360.75"),
				TradeDate:   tradeDate,
			},
			balance:    "500000",
			usedDaily:  "1.0",
			wantOK:     true,
			wantSpread: "210.75",
		},
		{
			name: "valid sell has no spread",
			order: domain.Order{
				CustomerID:  "C001",
				Type:        domain.OrderTypeSell,
				Quantity:    dec("1.0"),
				QuotedPrice: dec("42150"),
				TradeDate:   tradeDate,
			},
			balance:    "500000",
			usedDaily:  "0",
			wantOK:     true,
			wantSpread: "0",
		},
		{
			name: "rejects empty customer id",
			order: domain.Order{
				CustomerID:  "",
				Type:        domain.OrderTypeBuy,
				Quantity:    dec("1.0"),
				QuotedPrice: dec("42360.75"),
				TradeDate:   tradeDate,
			},
			balance:   "500000",
			usedDaily: "0",
			wantCode:  "missing_customer_id",
		},
		{
			name: "rejects invalid order type",
			order: domain.Order{
				CustomerID:  "C001",
				Type:        domain.OrderType("hold"),
				Quantity:    dec("1.0"),
				QuotedPrice: dec("42150"),
				TradeDate:   tradeDate,
			},
			balance:   "500000",
			usedDaily: "0",
			wantCode:  "invalid_order_type",
		},
		{
			name: "rejects zero quantity",
			order: domain.Order{
				CustomerID:  "C001",
				Type:        domain.OrderTypeBuy,
				Quantity:    dec("0"),
				QuotedPrice: dec("42360.75"),
				TradeDate:   tradeDate,
			},
			balance:   "500000",
			usedDaily: "0",
			wantCode:  "invalid_quantity",
		},
		{
			name: "rejects quantity not multiple of half baht-weight",
			order: domain.Order{
				CustomerID:  "C001",
				Type:        domain.OrderTypeSell,
				Quantity:    dec("1.25"),
				QuotedPrice: dec("42150"),
				TradeDate:   tradeDate,
			},
			balance:   "500000",
			usedDaily: "0",
			wantCode:  "invalid_quantity_step",
		},
		{
			name: "rejects zero quoted price",
			order: domain.Order{
				CustomerID:  "C001",
				Type:        domain.OrderTypeBuy,
				Quantity:    dec("1.0"),
				QuotedPrice: dec("0"),
				TradeDate:   tradeDate,
			},
			balance:   "500000",
			usedDaily: "0",
			wantCode:  "invalid_price",
		},
		{
			name: "rejects insufficient balance",
			order: domain.Order{
				CustomerID:  "C001",
				Type:        domain.OrderTypeBuy,
				Quantity:    dec("2.0"),
				QuotedPrice: dec("42360.75"),
				TradeDate:   tradeDate,
			},
			balance:   "1000",
			usedDaily: "0",
			wantCode:  "insufficient_balance",
		},
		{
			name: "rejects stale sell quote outside two percent",
			order: domain.Order{
				CustomerID:  "C001",
				Type:        domain.OrderTypeSell,
				Quantity:    dec("1.0"),
				QuotedPrice: dec("38000"),
				TradeDate:   tradeDate,
			},
			balance:   "500000",
			usedDaily: "0",
			wantCode:  "stale_or_inconsistent_price",
		},
		{
			name: "rejects daily limit exceeded",
			order: domain.Order{
				CustomerID:  "C001",
				Type:        domain.OrderTypeSell,
				Quantity:    dec("1.5"),
				QuotedPrice: dec("42150"),
				TradeDate:   tradeDate,
			},
			balance:   "500000",
			usedDaily: "4.0",
			wantCode:  "daily_limit_exceeded",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := inmemory.NewStore(
				map[string]decimal.Decimal{"C001": dec(tt.balance)},
				domain.MarketPrice{BaseSellPrice: dec("42150"), AsOf: tradeDate},
				map[string]decimal.Decimal{"C001:2026-04-01": dec(tt.usedDaily)},
			)
			validator := application.NewValidator(store, store, store, application.Config{})

			got, err := validator.Validate(context.Background(), tt.order)
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if got.Accepted != tt.wantOK {
				t.Fatalf("Accepted = %v, want %v; violations=%+v", got.Accepted, tt.wantOK, got.Violations)
			}
			if tt.wantCode != "" && !hasViolation(got, tt.wantCode) {
				t.Fatalf("missing violation %q in %+v", tt.wantCode, got.Violations)
			}
			if tt.wantSpread != "" && !got.SpreadAmount.Equal(dec(tt.wantSpread)) {
				t.Fatalf("SpreadAmount = %s, want %s", got.SpreadAmount, tt.wantSpread)
			}
		})
	}
}

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func hasViolation(result domain.ValidationResult, code string) bool {
	for _, violation := range result.Violations {
		if violation.Code == code {
			return true
		}
	}
	return false
}
