package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"intergold-assessment/internal/application"
	"intergold-assessment/internal/domain"
	"intergold-assessment/internal/inmemory"

	"github.com/shopspring/decimal"
)

func main() {
	tradeDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	store := inmemory.NewStore(
		map[string]decimal.Decimal{
			"C001": decimal.RequireFromString("500000"),
		},
		domain.MarketPrice{
			BaseSellPrice: decimal.RequireFromString("42150"),
			AsOf:          tradeDate,
		},
		map[string]decimal.Decimal{
			"C001:2026-04-01": decimal.RequireFromString("1.0"),
		},
	)

	validator := application.NewValidator(store, store, store, application.Config{})
	result, err := validator.Validate(context.Background(), domain.Order{
		CustomerID:  "C001",
		Type:        domain.OrderTypeBuy,
		Quantity:    decimal.RequireFromString("1.5"),
		QuotedPrice: decimal.RequireFromString("42360.75"),
		TradeDate:   tradeDate,
	})
	if err != nil {
		log.Fatal(err)
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(output))
}
