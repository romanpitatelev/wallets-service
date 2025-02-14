package main

import (
	"context"

	"github.com/romanpitatelev/wallets-service/configs"
	"github.com/romanpitatelev/wallets-service/internal/rest"
	"github.com/romanpitatelev/wallets-service/internal/service"
	"github.com/romanpitatelev/wallets-service/internal/store"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	conf := configs.LoadConfig()

	pgStore, err := store.New(ctx, conf)
	if err != nil {
		log.Panic().Err(err).Msg("filed to connect to database")
	}

	svc := service.New(pgStore)

	server, err := rest.New(svc)
	if err != nil {
		log.Panic().Msg("Failed to create new server")
	}

	err = server.Run()
	if err != nil {
		log.Panic().Msg("Failed to run the server")
	}
}
