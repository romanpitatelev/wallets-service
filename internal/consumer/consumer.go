package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

const (
	port  = "localhost:9094"
	topic = "users"
)

type userStore interface {
	UpsertUser(ctx context.Context, users models.User) error
}

type Consumer struct {
	consumer sarama.Consumer
	store    userStore
}

func New(store userStore) (*Consumer, error) {
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
	defer func() {
		if err := partConsumer.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close partConsumer")
		}
	}()

	for {
		select {
		case message := <-partConsumer.Messages():
			log.Trace().Msg("message received")

			var user models.User

			if err := json.Unmarshal(message.Value, &user); err != nil {
				return fmt.Errorf("failed to unmarshal message in for loop: %w", err)
			}
			// TODO нигде кроме main паники быть не должно
			if err := c.store.UpsertUser(ctx, user); err != nil {
				log.Panic().Err(err).Msg("failed to upsert user in for loop")
			}
		case err = <-partConsumer.Errors():
			log.Panic().Err(err).Msg("error from consumer in for loop")
		case <-ctx.Done():
			log.Info().Msg("shutting down from ctx.Done()")

			return nil
		}
	}
}

func (c *Consumer) Close() error {
	if err := c.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close Kafka consumer: %w", err)
	}

	return nil
}
