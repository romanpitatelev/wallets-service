run:
	@echo 'Running the project ...'
	go build -o bin/main ./cmd/wallets-service/main.go
	./bin/main

run_usergen:
	@echo 'Running user-generator ...'
	go build -o bin/user_generator ./cmd/user-generator/main.go
	./bin/user_generator

run_xr:
	@echo 'Running xr-service ...'
	go build -o bin/xr_service ./cmd/xr-service/main.go
	./bin/xr_service

up:
	docker compose up -d

down:
	docker compose down

tidy:
	go mod tidy

lint: tidy
	gofumpt -w .
	gci write . --skip-generated -s standard -s default
	golangci-lint run ./...

test: up
	go test -race ./... -v -coverpkg=./... -coverprofile=coverage.txt -covermode atomic
	go tool cover -func=coverage.txt | grep 'total'
	which gocover-cobertura || go install github.com/t-yuki/gocover-cobertura@latest
	gocover-cobertura < coverage.txt > coverage.xml

image_usergen:
	docker build -t user-generator -f deployment/user-generator/Dockerfile .

image_xr:
	docker build -t xr-service -f deployment/xr-service/Dockerfile .

generate:
	go generate ./...


PROTO_PATH=internal/xr-grpc
PROTO_FILE=xr.proto
OUT_DIR=internal/xr-grpc/gen/go
generate_grpc:
	protoc --proto_path=$(PROTO_PATH) \
	--go_out=$(OUT_DIR) --go_opt=paths=source_relative \
	--go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
	$(PROTO_PATH)/$(PROTO_FILE)
