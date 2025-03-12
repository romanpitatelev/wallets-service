FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/user-generator/main.go ./cmd/user-generator/main.go
RUN go build -o user-generator ./cmd/user-generator/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/user-generator .

CMD ["./user-generator"]