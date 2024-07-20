package consumer

import (
	"context"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/weeb-vip/anime-sync/config"
	"github.com/weeb-vip/anime-sync/internal/logger"
	"go.uber.org/zap"
	"time"
)

type Consumer[T any] interface {
	Receive(ctx context.Context, process func(ctx context.Context, msg pulsar.Message) error) error
}

type ConsumerImpl[T any] struct {
	client   pulsar.Client
	consumer *pulsar.Consumer
	config   config.PulsarConfig
}

func NewConsumer[T any](ctx context.Context, cfg config.PulsarConfig) Consumer[T] {
	log := logger.FromCtx(ctx)
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: cfg.URL,
	})

	if err != nil {
		log.Fatal("Error creating pulsar client: ", zap.String("error", err.Error()))
		return nil
	}

	return &ConsumerImpl[T]{
		config: cfg,
		client: client,
	}
}

func (c *ConsumerImpl[T]) Receive(ctx context.Context, process func(ctx context.Context, msg pulsar.Message) error) error {
	log := logger.FromCtx(ctx)
	if c.consumer != nil {
		return nil
	}

	consumer, err := c.client.Subscribe(pulsar.ConsumerOptions{
		Topic:            c.config.Topic,
		SubscriptionName: c.config.SubscribtionName,
		Type:             pulsar.Shared,
	})

	if err != nil {
		log.Fatal("Error creating pulsar consumer: ", zap.String("error", err.Error()))
		return err
	}

	defer consumer.Close()

	for {
		msg, err := consumer.Receive(ctx)
		if err != nil {
			log.Fatal("Error receiving message: ", zap.String("error", err.Error()))
		}

		log.Info("Received message", zap.String("msgId", msg.ID().String()))

		err = process(ctx, msg)
		if err != nil {
			log.Warn("error processing message: ", zap.String("error", err.Error()))
			continue
		}
		consumer.Ack(msg)
		time.Sleep(50 * time.Millisecond)
	}

}
