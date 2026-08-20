package eventing

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ThatCatDev/ep/v2/drivers"
	epKafka "github.com/ThatCatDev/ep/v2/drivers/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/weeb-vip/thetvdb-enrichment/config"
	"github.com/weeb-vip/thetvdb-enrichment/internal/db"
	anime2 "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime"
	"github.com/weeb-vip/thetvdb-enrichment/internal/logger"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_api"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_processor_kafka"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_service"
	"go.uber.org/zap"
	"net/http"
)

type SyncBannersOptions struct {
	DryRun  bool
	Limit   int
	DelayMs int
	After   string
}

// pageSize is the DB read size; it is unrelated to pacing, which is per-anime.
const pageSize = 200

// SyncBanners publishes series artwork for every anime carrying a thetvdbid.
//
// Deliberately separate from the link-driven enrichment path: that one deletes
// and re-imports episodes for a given season, which is destructive if the season
// is wrong and is unnecessary for artwork. This needs only the series id.
//
// Paced per anime because every iteration is a TheTVDB API call, and every
// publish becomes an image-sync download.
func SyncBanners(opts SyncBannersOptions) error {
	cfg := config.LoadConfigOrPanic()
	ctx := context.Background()
	log := logger.Get()
	ctx = logger.WithCtx(ctx, log)

	kafkaConfig := &epKafka.KafkaConfig{
		ConsumerGroupName:       cfg.KafkaConfig.ConsumerGroupName,
		BootstrapServers:        cfg.KafkaConfig.BootstrapServers,
		ConsumerAutoOffsetReset: &cfg.KafkaConfig.Offset,
	}
	driver := epKafka.NewKafkaDriver(kafkaConfig)
	defer func(d drivers.Driver[*kafka.Message]) {
		if err := d.Close(); err != nil {
			log.Error("Error closing Kafka driver", zap.String("error", err.Error()))
		}
	}(driver)

	httpClient := &http.Client{}
	thetvdbService := thetvdb_service.NewTheTVDBService(thetvdb_api.NewTheTVDBApi(cfg.TheTVDBConfig, httpClient))
	animeRepo := anime2.NewAnimeRepository(db.NewDB(cfg.DBConfig))

	var (
		after     = opts.After
		processed int
		published int
		noArtwork int
		failed    int
	)

	log.Info("Starting banner sync",
		zap.Bool("dryRun", opts.DryRun),
		zap.Int("limit", opts.Limit),
		zap.Int("delayMs", opts.DelayMs),
		zap.String("producerTopic", cfg.KafkaConfig.ProducerTopic))

	for {
		records, err := animeRepo.FindWithTheTVDBID(ctx, after, pageSize)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			break
		}

		for _, record := range records {
			after = record.ID
			if opts.Limit > 0 && processed >= opts.Limit {
				break
			}
			processed++

			if record.TheTVDBID == nil || *record.TheTVDBID == "" {
				continue
			}

			bannerURL, err := thetvdbService.GetSeriesBannerURL(ctx, *record.TheTVDBID)
			if err != nil {
				failed++
				log.Warn("Failed to fetch artwork",
					zap.String("anime_id", record.ID),
					zap.String("thetvdb_id", *record.TheTVDBID),
					zap.Error(err))
				continue
			}
			if bannerURL == "" {
				noArtwork++
				continue
			}

			if opts.DryRun {
				published++
				log.Info("Would publish banner",
					zap.String("anime_id", record.ID),
					zap.String("url", bannerURL))
			} else {
				payload, err := json.Marshal(thetvdb_processor_kafka.ImagePayload{
					Data: thetvdb_processor_kafka.ImageSchema{
						ID:   record.ID,
						Name: record.ID,
						URL:  bannerURL,
						Type: "Banner",
					},
				})
				if err != nil {
					failed++
					log.Warn("Failed to marshal banner payload", zap.Error(err))
					continue
				}
				if err := driver.Produce(ctx, cfg.KafkaConfig.ProducerTopic, &kafka.Message{Value: payload}); err != nil {
					failed++
					log.Warn("Failed to publish banner", zap.String("anime_id", record.ID), zap.Error(err))
					continue
				}
				published++
			}

			if processed%100 == 0 {
				log.Info("Progress",
					zap.Int("processed", processed),
					zap.Int("published", published),
					zap.Int("noArtwork", noArtwork),
					zap.Int("failed", failed))
			}

			if opts.DelayMs > 0 {
				time.Sleep(time.Duration(opts.DelayMs) * time.Millisecond)
			}
		}

		if opts.Limit > 0 && processed >= opts.Limit {
			break
		}
		if len(records) < pageSize {
			break
		}
	}

	log.Info("Banner sync complete",
		zap.Int("processed", processed),
		zap.Int("published", published),
		zap.Int("noArtwork", noArtwork),
		zap.Int("failed", failed),
		zap.String("lastID", after))
	return nil
}
