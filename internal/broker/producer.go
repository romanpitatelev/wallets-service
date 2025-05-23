package broker

import (
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/romanpitatelev/wallets-service/internal/models"
)

const transactiontTopic = "transaction_update"

type ProducerConfig struct {
	Addr string
}

type Producer struct {
	producer sarama.SyncProducer
}

func NewProducer(cfg ProducerConfig) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{cfg.Addr}, config)
	if err != nil {
		return nil, fmt.Errorf("error creating producer: %w", err)
	}

	return &Producer{producer: producer}, nil
}

func (p *Producer) Close() error {
	if err := p.producer.Close(); err != nil {
		return fmt.Errorf("error closing producer: %w", err)
	}

	return nil
}

func (p *Producer) ProduceTxToKafka(transaction models.Transaction) error {
	bytes, err := json.Marshal(transaction)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON when sending tx to kafka: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: transactiontTopic,
		Value: sarama.StringEncoder(bytes),
	}

	if _, _, err := p.producer.SendMessage(message); err != nil {
		return fmt.Errorf("error sending message to Kafka: %w", err)
	}

	return nil
}
