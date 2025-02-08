BINARY_NAME=wallets-service

build:
	@echo 'Building binary ...'
	GOOS=linux go build -o wallets-service/cmd/wallets-service/main.go

run: build
	@echo 'Running the project ...'
	./wallets-service/cmd/wallets-service/main.go

up:
	docker compose up -d

down:
	docker compose down