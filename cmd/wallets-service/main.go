package main

import (
	"github.com/romanpitatelev/wallets-service/internal/app"
	"github.com/romanpitatelev/wallets-service/internal/configs"
)

func main() {
	cfg := configs.New()

	if err := app.Run(cfg); err != nil {
		panic(err)
	}
}
