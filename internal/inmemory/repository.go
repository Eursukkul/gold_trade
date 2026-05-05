// Package inmemory contains simple adapters for assessment and tests.
package inmemory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"intergold-assessment/internal/domain"

	"github.com/shopspring/decimal"
)

// Store is an in-memory implementation of domain data ports.
type Store struct {
	mu       sync.RWMutex
	balances map[string]decimal.Decimal
	price    domain.MarketPrice
	volumes  map[string]decimal.Decimal
}

// NewStore creates an in-memory store with caller-provided data.
func NewStore(balances map[string]decimal.Decimal, price domain.MarketPrice, volumes map[string]decimal.Decimal) *Store {
	return &Store{
		balances: cloneMap(balances),
		price:    price,
		volumes:  cloneMap(volumes),
	}
}

// AvailableBalance returns a customer balance.
func (s *Store) AvailableBalance(_ context.Context, customerID string) (decimal.Decimal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	balance, ok := s.balances[customerID]
	if !ok {
		return decimal.Zero, fmt.Errorf("customer balance not found: %s", customerID)
	}

	return balance, nil
}

// CurrentPrice returns the configured market price.
func (s *Store) CurrentPrice(_ context.Context) (domain.MarketPrice, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.price, nil
}

// TradedQuantity returns the customer daily volume for the requested date.
func (s *Store) TradedQuantity(_ context.Context, customerID string, day time.Time) (decimal.Decimal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.volumes[volumeKey(customerID, day)], nil
}

// SetTradedQuantity updates daily volume for a customer and date.
func (s *Store) SetTradedQuantity(customerID string, day time.Time, quantity decimal.Decimal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.volumes[volumeKey(customerID, day)] = quantity
}

func volumeKey(customerID string, day time.Time) string {
	return customerID + ":" + day.UTC().Format("2006-01-02")
}

func cloneMap(in map[string]decimal.Decimal) map[string]decimal.Decimal {
	out := make(map[string]decimal.Decimal, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
