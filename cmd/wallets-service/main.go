package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/romanpitatelev/wallets-service/configs"
	"github.com/romanpitatelev/wallets-service/internal/consumer"
	"github.com/romanpitatelev/wallets-service/internal/rest"
	"github.com/romanpitatelev/wallets-service/internal/service"
	"github.com/romanpitatelev/wallets-service/internal/store"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	conf := configs.NewConfig()

	pgStore, err := store.New(ctx, conf)
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to database")
	}

	if err := pgStore.Migrate(migrate.Up); err != nil {
		log.Panic().Err(err).Msg("failed to migrate")
	}

	log.Info().Msg("successful migration")

	kafkaConsumer, err := consumer.New(pgStore)
	if err != nil {
		log.Panic().Err(err).Msg("failed to create kafka consumer")
	}

	log.Trace().Msg("kafka consumer created")

	svc := service.New(pgStore, service.Config{
		StaleWalletDuration: conf.StaleWalletDuration,
		PerformCheckPeriod:  conf.PerformCheckPeriod,
	})

	server := rest.New(svc)

	errGr, ctx := errgroup.WithContext(ctx)

	errGr.Go(func() error {
		if err := kafkaConsumer.Run(ctx); err != nil {
			return fmt.Errorf("failed to run kafka consumer: %w", err)
		}

		return nil
	})

	errGr.Go(func() error {
		if err := svc.Run(ctx); err != nil {
			return fmt.Errorf("failed to run service: %w", err)
		}

		return nil
	})

	errGr.Go(func() error {
		if err := server.Run(ctx); err != nil {
			return fmt.Errorf("failed to run the server: %w", err)
		}

		return nil
	})

	if err = errGr.Wait(); err != nil {
		log.Panic().Err(err).Msg("failed to wait blocks")
	}
}
