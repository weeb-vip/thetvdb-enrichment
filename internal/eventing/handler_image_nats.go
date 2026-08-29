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
	"golang.org/x/sync/errgroup"
	"net/http"
)

const (
	maxRetries     = 3
	retryHeaderKey = "retry"
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

	retrySubject := cfg.NatsConfig.Subject + "-retry"
	dlqSubject := cfg.NatsConfig.Subject + "-dlq"

	// The retry consumer runs in this same process rather than a second
	// deployment. It needs its own driver because the durable consumer name is
	// driver-level configuration, not per-subject: two Consume calls on one
	// driver would call CreateOrUpdateConsumer with the same durable name and
	// different filter subjects, and the second would reconfigure the first.
	retryDriver := epNats.NewNatsDriver(&epNats.Config{
		URL:                     cfg.NatsConfig.URL,
		ConsumerGroupName:       cfg.NatsConfig.ConsumerGroupName + "-retry",
		StreamName:              cfg.NatsConfig.StreamName,
		ConsumerAutoOffsetReset: &cfg.NatsConfig.Offset,
	})
	defer func(d drivers.Driver[*epNats.Message]) {
		if err := d.Close(); err != nil {
			log.Error("Error closing NATS retry driver", zap.String("error", err.Error()))
		}
	}(retryDriver)

	processorInstance := processor.NewProcessor[*epNats.Message, thetvdb_processor.Payload](driver, cfg.NatsConfig.Subject, tvdbProcessor.Process).
		AddMiddleware(backoffretry.NewBackoffRetry[thetvdb_processor.Payload](driver, backoffretry.Config{
			MaxRetries: maxRetries,
			HeaderKey:  retryHeaderKey,
			RetryQueue: retrySubject,
		}).Process)

	// Exhausted retries go to a dead-letter subject rather than back onto the
	// retry subject. ep acks and drops a message once the counter reaches
	// MaxRetries, so cycling it here would make a permanently failing message
	// disappear with no record.
	retryProcessorInstance := processor.NewProcessor[*epNats.Message, thetvdb_processor.Payload](retryDriver, retrySubject, tvdbProcessor.Process).
		AddMiddleware(backoffretry.NewBackoffRetry[thetvdb_processor.Payload](retryDriver, backoffretry.Config{
			MaxRetries: maxRetries,
			HeaderKey:  retryHeaderKey,
			RetryQueue: dlqSubject,
		}).Process)

	log.Info("Starting NATS processors",
		zap.String("subject", cfg.NatsConfig.Subject),
		zap.String("retry_subject", retrySubject),
		zap.String("dlq_subject", dlqSubject))

	// One consumer returning must stop the other: Consume blocks until its
	// iterator is stopped, so without cancelling here a dead main consumer
	// would leave the process alive and apparently healthy.
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error { return processorInstance.Run(groupCtx) })
	group.Go(func() error { return retryProcessorInstance.Run(groupCtx) })

	if err := group.Wait(); err != nil && ctx.Err() == nil { // Ignore error if caused by context cancellation
		log.Error("Error consuming messages", zap.String("error", err.Error()))
		return err
	}

	return nil
}
