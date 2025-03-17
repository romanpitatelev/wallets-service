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

lint:
	golangci-lint run ./...

test:
	go test ./... -v -coverpkg=./... -coverprofile=coverage.txt -covermode atomic
	go tool cover -func=coverage.txt | grep 'total'
	gocover-cobertura < coverage.txt > coverage.xml

image_usergen:
	docker build -t user-generator -f deployment/user-generator/Dockerfile .

image_xr:
	docker build -t xr-service -f deployment/xr-service/Dockerfile .