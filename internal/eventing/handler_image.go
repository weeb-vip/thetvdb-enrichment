package eventing

import (
	"context"
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/weeb-vip/thetvdb-enrichment/config"
	"github.com/weeb-vip/thetvdb-enrichment/internal/db"
	anime2 "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime"
	anime "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime_episode"
	"github.com/weeb-vip/thetvdb-enrichment/internal/logger"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/consumer"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/processor"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_api"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_processor"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_service"
	"net/http"
)

func Eventing() error {
	cfg := config.LoadConfigOrPanic()
	ctx := context.Background()
	log := logger.Get()
	ctx = logger.WithCtx(ctx, log)

	httpClient := &http.Client{}
	thetvdbAPI := thetvdb_api.NewTheTVDBApi(cfg.TheTVDBConfig, httpClient)

	thetvdbService := thetvdb_service.NewTheTVDBService(thetvdbAPI)
	database := db.NewDB(cfg.DBConfig)
	episodeRepo := anime.NewAnimeEpisodeRepository(database)
	animeRepo := anime2.NewAnimeRepository(database)
	tvdbProcessor := thetvdb_processor.NewTheTVDBProcessor(thetvdbService, animeRepo, episodeRepo)
	messageProcessor := processor.NewProcessor[thetvdb_processor.Payload]()

	animeConsumer := consumer.NewConsumer[thetvdb_processor.Payload](ctx, cfg.PulsarConfig)

	log.Info("Starting anime eventing")
	err := animeConsumer.Receive(ctx, func(ctx context.Context, msg pulsar.Message) error {
		return messageProcessor.Process(ctx, string(msg.Payload()), tvdbProcessor.Process)
	})
	if err != nil {
		log.Error(fmt.Sprintf("Error receiving message: %v", err))
		return err
	}

	return err
}
