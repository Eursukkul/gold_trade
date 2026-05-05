FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

CMD ["sh", "-c", "go test ./... && go run ./cmd/order-validator"]
