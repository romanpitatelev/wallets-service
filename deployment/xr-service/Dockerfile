FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o xr-service ./cmd/xr-service/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/xr-service .

CMD ["./xr-service"]