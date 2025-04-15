package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/romanpitatelev/wallets-service/internal/configs"
	"github.com/romanpitatelev/wallets-service/internal/controller/consumer"
	"github.com/romanpitatelev/wallets-service/internal/controller/rest"
	transactionshandler "github.com/romanpitatelev/wallets-service/internal/controller/rest/transactions-handler"
	walletshandler "github.com/romanpitatelev/wallets-service/internal/controller/rest/wallets-handler"
	"github.com/romanpitatelev/wallets-service/internal/repository/producer"
	"github.com/romanpitatelev/wallets-service/internal/repository/store"
	transactionsrepo "github.com/romanpitatelev/wallets-service/internal/repository/transactions-repo"
	walletsrepo "github.com/romanpitatelev/wallets-service/internal/repository/wallets-repo"
	xrgrpcclient "github.com/romanpitatelev/wallets-service/internal/repository/xr-grpc-client"
	transactionsservice "github.com/romanpitatelev/wallets-service/internal/usecase/transactions-service"
	walletsservice "github.com/romanpitatelev/wallets-service/internal/usecase/wallets-service"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
	"golang.org/x/sync/errgroup"
)

func Run(cfg *configs.Config) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	db, err := store.New(ctx, store.Config{Dsn: cfg.GetPostgresDSN()})
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to database")
	}

	if err := db.Migrate(migrate.Up); err != nil {
		log.Panic().Err(err).Msg("failed to migrate")
	}

	log.Info().Msg("successful migration")

	walletsRepo := walletsrepo.New(db)
	transactionsRepo := transactionsrepo.New(db, walletsRepo)

	kafkaRepo, err := consumer.New(db, consumer.ConsumerConfig{Addr: cfg.GetKafkaAddress()})
	if err != nil {
		log.Panic().Err(err).Msg("failed to create kafka consumer")
	}

	log.Info().Msg("kafka consumer created")

	defer func() {
		if err = kafkaRepo.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close kafka consumer")
		}
	}()

	kafkaTxProducer, err := producer.New(producer.ProducerConfig{Addr: cfg.GetKafkaAddress()})
	if err != nil {
		log.Panic().Err(err).Msg("failed to create kafka transactions producer")
	}

	log.Info().Msg("kafka transactions producer created")

	defer func() {
		if err = kafkaTxProducer.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close kafka transactions producer")
		}
	}()

	xrRepo, err := xrgrpcclient.New(xrgrpcclient.Config{Host: cfg.GetXRgRPCServerAddress()})
	if err != nil {
		log.Panic().Err(err).Msg("failed to create xr gRPC client")
	}

	walletsService := walletsservice.New(
		walletsservice.Config{
			StaleWalletDuration: cfg.GetStaleWalletDuration(),
			PerformCheckPeriod:  cfg.GetPerformCheckPeriod(),
		},
		walletsRepo,
		xrRepo,
		db,
	)

	transactionsService := transactionsservice.New(
		walletsRepo,
		transactionsRepo,
		xrRepo,
		db,
		kafkaTxProducer,
	)

	walletsHandler := walletshandler.New(walletsService)
	transactionsHandler := transactionshandler.New(transactionsService)

	server := rest.New(
		rest.Config{Port: cfg.GetAppPort()},
		walletsHandler,
		transactionsHandler,
		rest.GetPublicKey())

	errGr, ctx := errgroup.WithContext(ctx)

	errGr.Go(func() error {
		if err := kafkaRepo.Run(ctx); err != nil {
			return fmt.Errorf("failed to run kafka consumer: %w", err)
		}

		return nil
	})

	errGr.Go(func() error {
		if err := walletsService.Run(ctx); err != nil {
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

	return nil
}
