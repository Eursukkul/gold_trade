# วิธีรันโค้ด

## รันแบบ Local

```bash
go mod tidy
go test ./...
go run ./cmd/order-validator
```

## รันด้วย Docker

```bash
docker compose run --rm validator
```

## ตรวจ Race Condition

```bash
go test -race ./...
```
