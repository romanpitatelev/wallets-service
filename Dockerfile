FROM golang:1.24.0-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o wallets-service ./cmd/wallets-service/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/wallets-service .

EXPOSE 8081
CMD ["./wallets-service"]