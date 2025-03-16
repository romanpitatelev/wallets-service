package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/romanpitatelev/wallets-service/configs"
	"github.com/romanpitatelev/wallets-service/internal/consumer"
	jwtclaims "github.com/romanpitatelev/wallets-service/internal/jwt-claims"
	"github.com/romanpitatelev/wallets-service/internal/rest"
	"github.com/romanpitatelev/wallets-service/internal/service"
	"github.com/romanpitatelev/wallets-service/internal/store"
	xrclient "github.com/romanpitatelev/wallets-service/internal/xr/xr-client"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
	"golang.org/x/sync/errgroup"
)

//nolint:funlen
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	conf := configs.New()

	pgStore, err := store.New(ctx, store.Config{Dsn: conf.GetPostgresDSN()})
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to database")
	}

	if err := pgStore.Migrate(migrate.Up); err != nil {
		log.Panic().Err(err).Msg("failed to migrate")
	}

	log.Info().Msg("successful migration")

	kafkaConsumer, err := consumer.New(pgStore, consumer.Config{Port: conf.GetKafkaPort()})
	if err != nil {
		log.Panic().Err(err).Msg("failed to create kafka consumer")
	}

	defer func() {
		if err = kafkaConsumer.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close kafka consumer")
		}
	}()

	log.Info().Msg("kafka consumer created")

	client := xrclient.New(xrclient.Config{ServerAddress: conf.GetXRServerAddress()})

	svc := service.New(
		pgStore,
		service.Config{
			StaleWalletDuration: conf.GetStaleWalletDuration(),
			PerformCheckPeriod:  conf.GetPerformCheckPeriod(),
		},
		client,
	)

	jwtClaims := jwtclaims.New()

	server := rest.New(rest.Config{Port: conf.GetAppPort()}, svc, jwtClaims.GetPublicKey())

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
