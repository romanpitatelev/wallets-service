package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/romanpitatelev/wallets-service/internal/entity"
	"github.com/rs/zerolog/log"
)

const (
	topic = "users"
)

type Consumer struct {
	consumer sarama.Consumer
	store    userStore
}

type ConsumerConfig struct {
	Addr string
}

type userStore interface {
	UpsertUser(ctx context.Context, users entity.User) error
}

func New(store userStore, conf ConsumerConfig) (*Consumer, error) {
	var consumer sarama.Consumer

	var err error

	maxRetries := 10
	delay := time.Second

	for i := range maxRetries {
		consumer, err = sarama.NewConsumer([]string{conf.Addr}, nil)
		if err == nil {
			break
		}

		log.Warn().Err(err).Msgf("failed to create Kafka consumer (attempt%d/%d), retrying ...", i+1, maxRetries)
		time.Sleep(delay * 1)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer in sarama.NewConsumer(): %w", err)
	}

	return &Consumer{
		consumer: consumer,
		store:    store,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	partConsumer, err := c.consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
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

			var user entity.User

			if err := json.Unmarshal(message.Value, &user); err != nil {
				return fmt.Errorf("failed to unmarshal message in the for loop: %w", err)
			}

			if err := c.store.UpsertUser(ctx, user); err != nil {
				return fmt.Errorf("failed to upsert user in for loop: %w", err)
			}
		case err = <-partConsumer.Errors():
			return fmt.Errorf("error from consumer in for loop: %w", err)
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
