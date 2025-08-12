package consumer

import (
	"context"
	"github.com/segmentio/kafka-go"
	"imageProcessor/internal/config"
	"log/slog"
	"time"
)

type Consumer struct {
	reader *kafka.Reader
	log    *slog.Logger
}

func NewConsumer(kafkaCfg *config.Kafka, log *slog.Logger) (*Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        kafkaCfg.Brokers,
		Topic:          kafkaCfg.Topic,
		GroupID:        kafkaCfg.GroupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		MaxWait:        1 * time.Second,
		CommitInterval: 1 * time.Second,
	})

	return &Consumer{
		reader: reader,
		log:    log,
	}, nil
}

func (c *Consumer) ReadMessages(ctx context.Context, handler func(context.Context, []byte) error) {
	c.log.Info("kafka consumer started")

	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			c.log.Error("error reading message from kafka", slog.String("error", err.Error()))
			continue
		}

		c.log.Info(
			"message received",
			slog.String("topic", m.Topic),
			slog.Int("partition", m.Partition),
			slog.Int64("offset", m.Offset),
		)

		if err = handler(ctx, m.Value); err != nil {
			c.log.Error("error handling message", slog.String("error", err.Error()))
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
