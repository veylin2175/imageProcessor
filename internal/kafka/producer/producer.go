package producer

import (
	"context"
	"imageProcessor/internal/config"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	log    *slog.Logger
}

func NewProducer(kafkaCfg *config.Kafka, log *slog.Logger) (*Producer, error) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaCfg.Brokers...),
		Topic:    kafkaCfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{
		writer: writer,
		log:    log,
	}, nil
}

func (p *Producer) SendMessage(ctx context.Context, message []byte) error {
	msg := kafka.Message{
		Value: message,
	}

	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		p.log.Error("failed to send message to kafka", slog.String("topic", p.writer.Topic), slog.String("error", err.Error()))
		return err
	}

	p.log.Info("message sent to kafka", slog.String("topic", p.writer.Topic))
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
