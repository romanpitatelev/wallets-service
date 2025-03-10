build:
	@echo 'Building binary ...'
	go build -o bin/main ./cmd/wallets_service/main.go

run: build
	@echo 'Running the project ...'
	./bin/main

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