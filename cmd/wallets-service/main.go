package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/romanpitatelev/wallets-service/internal/broker"
	"github.com/romanpitatelev/wallets-service/internal/configs"
	"github.com/romanpitatelev/wallets-service/internal/rest"
	"github.com/romanpitatelev/wallets-service/internal/service"
	"github.com/romanpitatelev/wallets-service/internal/store"
	xrgrpcclient "github.com/romanpitatelev/wallets-service/internal/xr/xr-grpc/xr-client"
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

	kafkaConsumer, err := broker.NewConsumer(pgStore, broker.ConsumerConfig{Addr: conf.GetKafkaAddress()})
	if err != nil {
		log.Panic().Err(err).Msg("failed to create kafka consumer")
	}

	defer func() {
		if err = kafkaConsumer.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close kafka consumer")
		}
	}()

	log.Info().Msg("kafka consumer created")

	kafkaTxProducer, err := broker.NewProducer(broker.ProducerConfig{Addr: conf.GetKafkaAddress()})
	if err != nil {
		log.Panic().Err(err).Msg("failed to create kafka transactions producer")
	}

	log.Info().Msg("kafka producer created")

	defer func() {
		if err = kafkaTxProducer.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close kafka transactions producer")
		}
	}()

	//	xrClient := xrhttpclient.New(xrhttpclient.Config{ServerAddress: conf.GetXRHttpServerAddress()})

	xrClient, err := xrgrpcclient.New(xrgrpcclient.Config{Host: conf.GetXRgRPCServerAddress()})
	if err != nil {
		log.Panic().Err(err).Msg("failed to create xr gRPC client")
	}

	svc := service.New(
		service.Config{
			StaleWalletDuration: conf.GetStaleWalletDuration(),
			PerformCheckPeriod:  conf.GetPerformCheckPeriod(),
		},
		pgStore,
		xrClient,
		kafkaTxProducer,
	)

	server := rest.New(rest.Config{Port: conf.GetAppPort()}, svc, rest.GetPublicKey())

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
