build:
	@echo 'Building binary ...'
	go build -o bin/main ./cmd/wallets-service/main.go

run: build
	@echo 'Running the project ...'
	./bin/main

up:
	docker compose up -d

down:
	docker compose down

lint:
	golangci-lint run