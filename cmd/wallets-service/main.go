package main

import (
	"context"

	"github.com/romanpitatelev/wallets-service/configs"
	"github.com/romanpitatelev/wallets-service/internal/rest"
	"github.com/romanpitatelev/wallets-service/internal/service"
	"github.com/romanpitatelev/wallets-service/internal/store"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
)

func main() {
	ctx := context.Background()

	conf := configs.NewConfig()

	pgStore, err := store.New(ctx, conf)
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to database")
	}

	if err := pgStore.Migrate(migrate.Up); err != nil {
		log.Panic().Err(err).Msg("failed to migrate")
	}

	log.Info().Msg("successful migration")

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
