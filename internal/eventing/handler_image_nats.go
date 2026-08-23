package eventing

import (
	"context"
	"github.com/ThatCatDev/ep/v2/drivers"
	epNats "github.com/ThatCatDev/ep/v2/drivers/nats"
	"github.com/ThatCatDev/ep/v2/middlewares/nats/backoffretry"
	"github.com/ThatCatDev/ep/v2/processor"
	"github.com/weeb-vip/thetvdb-enrichment/config"
	"github.com/weeb-vip/thetvdb-enrichment/internal/db"
	anime2 "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime"
	anime "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime_episode"
	"github.com/weeb-vip/thetvdb-enrichment/internal/logger"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_api"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_processor"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_service"
	"go.uber.org/zap"
	"net/http"
)

// EventingNats is the NATS counterpart of EventingKafka.
//
// A separate entry point rather than a flag, so production keeps running the
// Kafka command untouched while staging moves over. The processor, repositories
// and retry policy are shared; only the transport differs.
//
// No transform middleware, matching the Kafka handler: ep hands the raw body to
// Event.Transform and both drivers pass it the same way, so Payload is
// populated identically. Only the CDC consumers need a transform, because their
// bodies carry a Debezium envelope.
func EventingNats() error {
	cfg := config.LoadConfigOrPanic()
	ctx := context.Background()
	log := logger.Get()
	ctx = logger.WithCtx(ctx, log)

	natsConfig := &epNats.Config{
		URL:               cfg.NatsConfig.URL,
		ConsumerGroupName: cfg.NatsConfig.ConsumerGroupName,
		// Empty StreamName: thetvdb-enrichment.anime is produced by the sync
		// services, not Debezium, so the driver creates a stream from the
		// subject rather than binding to the CDC one.
		StreamName:              cfg.NatsConfig.StreamName,
		ConsumerAutoOffsetReset: &cfg.NatsConfig.Offset,
	}

	driver := epNats.NewNatsDriver(natsConfig)
	defer func(driver drivers.Driver[*epNats.Message]) {
		err := driver.Close()
		if err != nil {
			log.Error("Error closing NATS driver", zap.String("error", err.Error()))
		} else {
			log.Info("NATS driver closed successfully")
		}
	}(driver)

	httpClient := &http.Client{}
	thetvdbAPI := thetvdb_api.NewTheTVDBApi(cfg.TheTVDBConfig, httpClient)

	thetvdbService := thetvdb_service.NewTheTVDBService(thetvdbAPI)
	database := db.NewDB(cfg.DBConfig)
	episodeRepo := anime.NewAnimeEpisodeRepository(database)
	animeRepo := anime2.NewAnimeRepository(database)

	// Takes the encoded value rather than a *epNats.Message: the processor is
	// generic over the driver message now, so building the transport's message
	// is this closure's job.
	producerFunc := func(ctx context.Context, value []byte) error {
		return driver.Produce(ctx, cfg.NatsConfig.ProducerSubject, &epNats.Message{Data: value})
	}
	tvdbProcessor := thetvdb_processor.NewTheTVDBProcessor[*epNats.Message](thetvdbService, animeRepo, episodeRepo, producerFunc)

	processorInstance := processor.NewProcessor[*epNats.Message, thetvdb_processor.Payload](driver, cfg.NatsConfig.Subject, tvdbProcessor.Process)

	log.Info("initializing backoff retry middleware", zap.String("subject", cfg.NatsConfig.Subject))
	backoffRetryInstance := backoffretry.NewBackoffRetry[thetvdb_processor.Payload](driver, backoffretry.Config{
		MaxRetries: 3,
		HeaderKey:  "retry",
		RetryQueue: cfg.NatsConfig.Subject + "-retry",
	})

	log.Info("Starting NATS processor", zap.String("subject", cfg.NatsConfig.Subject))
	err := processorInstance.
		AddMiddleware(backoffRetryInstance.Process).
		Run(ctx)

	if err != nil && ctx.Err() == nil { // Ignore error if caused by context cancellation
		log.Error("Error consuming messages", zap.String("error", err.Error()))
		return err
	}

	return nil
}
