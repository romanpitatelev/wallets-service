package consumer

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
)

const (
	kafkaConsumer = "localhost:9092"
	topic         = "users"
)

type Consumer struct {
	group  sarama.ConsumerGroup
	topics []string
}

func NewConsumer(brokers []string, groupID string, topics []string) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRange()}
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	group, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		group:  group,
		topics: topics,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	handler := &consumerHandler{}
	for {
		if err := c.group.Consume(ctx, c.topics, handler); err != nil {
			return fmt.Errorf("error from consumer: %w", err)
		}

		if ctx.Err() != nil {
			return fmt.Errorf("context was canceled: %w", ctx.Err())
		}
	}

	return nil
}

func (c *Consumer) Close() error {
	return c.group.Close()
}

type consumerHandler struct{}

func (h *consumerHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info().Msg("Consumer group session initiated")
	return nil
}
