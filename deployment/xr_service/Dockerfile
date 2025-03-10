FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o xr_service ./cmd/xr_service/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/xr_service .

CMD ["./xr_service"]

# in order to build image use this command from the root directory of the project
# docker build -t xr_service -f deployment/xr_service/Dockerfile .