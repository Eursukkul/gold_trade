# InterGold Software Engineer Assessment

Submission package for the InterGold Mid Software Engineer Assessment.

## Contents

- `internal/domain`: entities, value objects, and validation result types
- `internal/application`: order validation use case and ports for Parts 2 and 3
- `internal/inmemory`: in-memory driven adapters used for demo and tests
- `cmd/order-validator`: small executable showing a valid buy order
- `docs/answers.md`: written answers for Parts 1, 4, and 5

## Design Notes

The validation module follows a small Clean Architecture layout:

- `cmd/order-validator` is the delivery layer.
- `internal/application` is the use-case layer. It orchestrates validation and defines ports for balance, market price, and daily volume.
- `internal/domain` is the enterprise business layer. It owns order types and validation result objects.
- `internal/inmemory` is an infrastructure adapter for the application ports.

Dependencies point inward: infrastructure and delivery depend on application/domain, while domain does not know about storage, APIs, or frameworks.

All financial values use `github.com/shopspring/decimal`. Inputs should be parsed from strings, not `float64`, so prices, balances, and quantities are deterministic.

## Run Locally

```bash
go mod tidy
go test ./...
go run ./cmd/order-validator
```

For race detection:

```bash
go test -race ./...
```

## Run With Docker

```bash
docker compose run --rm validator
```

The compose command runs the unit tests and then executes the sample validator CLI.
