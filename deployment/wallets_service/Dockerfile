FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY ../../go.mod ../../go.sum ./
RUN go mod download

COPY ../.. .
RUN go build -o wallets_service ./cmd/wallets_service/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/wallets_service .

EXPOSE 8081
CMD ["./wallets-service"]