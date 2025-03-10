FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/user_generator/main.go ./cmd/user_generator/main.go
RUN go build -o user_generator ./cmd/user_generator/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/user_generator .

CMD ["./user_generator"]


# in order to build image use this command from the root directory of the project
# docker build -t user_generator -f deployment/user_generator/Dockerfile .