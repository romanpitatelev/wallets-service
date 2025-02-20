package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/romanpitatelev/wallets-service/internal/store"
	"github.com/rs/zerolog/log"
)

const (
	port  = "localhost:9094"
	topic = "users"
)

type Consumer struct {
	consumer sarama.Consumer
	store    *store.DataStore
}

func New(store *store.DataStore) (*Consumer, error) {
	consumer, err := sarama.NewConsumer([]string{port}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer in sarama.NewConsumer(): %w", err)
	}

	return &Consumer{
		consumer: consumer,
		store:    store,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	partConsumer, err := c.consumer.ConsumePartition(topic, 0, sarama.OffsetOldest)
	if err != nil {
		return fmt.Errorf("failed to initiate consumer in Run(): %w", err)
	}
	defer partConsumer.AsyncClose()

	// sigs := make(chan os.Signal, 1)
	// signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case message := <-partConsumer.Messages():
			log.Trace().Msg("message received")

			var user models.User

			if err := json.Unmarshal(message.Value, &user); err != nil {
				log.Panic().Err(err).Msg("failed to unmarshal message in for loop")
			}

			if err := c.store.UpsertUser(ctx, user); err != nil {
				log.Panic().Err(err).Msg("failed to upsert user in for loop")
			}
		case err = <-partConsumer.Errors():
			log.Panic().Err(err).Msg("error from consumer in for loop")
		case <-ctx.Done():
			log.Info().Msg("shutting down from ctx.Done()")
			// case <-sigs:
			// 	log.Info().Msg("some signal received from sigs channel")
		}
	}
}

func (c *Consumer) Close() error {
	if err := c.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close Kafka consumer: %w", err)
	}

	return nil
}
