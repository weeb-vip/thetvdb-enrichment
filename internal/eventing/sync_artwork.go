package eventing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ThatCatDev/ep/v2/drivers"
	epKafka "github.com/ThatCatDev/ep/v2/drivers/kafka"
	epNats "github.com/ThatCatDev/ep/v2/drivers/nats"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/weeb-vip/thetvdb-enrichment/config"
	"github.com/weeb-vip/thetvdb-enrichment/internal/db"
	anime2 "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime"
	"github.com/weeb-vip/thetvdb-enrichment/internal/logger"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_api"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_processor"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_service"
	"go.uber.org/zap"
	"net/http"
)

// ArtworkKind is which TheTVDB artwork a run publishes. The walk, the pacing,
// the keyset paging and the resume are identical for both; only the resolver and
// the image-sync data type differ, so they share one implementation rather than
// two files that drift.
type ArtworkKind string

const (
	ArtworkBanner ArtworkKind = "Banner"
	ArtworkPoster ArtworkKind = "Poster"
)

type SyncBannersOptions struct {
	// Kind defaults to ArtworkBanner when empty, so existing callers are
	// unchanged.
	Kind    ArtworkKind
	DryRun  bool
	Limit   int
	DelayMs int
	After   string
	// Season narrows the walk to one season's anime (anime_seasons, e.g.
	// FALL_2026); IDs to the given anime. Either replaces the keyset walk.
	Season string
	IDs    []string
	// Force marks each message so image-sync re-pulls the artwork even when
	// it already holds that source; without it only new or changed artwork
	// is stored.
	Force bool
	// Transport is "nats" (the default: image-sync consumes NATS everywhere
	// now) or "kafka", kept for a cluster that still has one.
	Transport string
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
	if opts.Kind == "" {
		opts.Kind = ArtworkBanner
	}
	cfg := config.LoadConfigOrPanic()
	ctx := context.Background()
	log := logger.Get()
	ctx = logger.WithCtx(ctx, log)

	// One publish function whichever transport carries it.
	var publish func(ctx context.Context, value []byte) error
	switch opts.Transport {
	case "nats", "":
		natsDriver := epNats.NewNatsDriver(&epNats.Config{URL: cfg.NatsConfig.URL})
		defer func(d drivers.Driver[*epNats.Message]) {
			if err := d.Close(); err != nil {
				log.Error("Error closing NATS driver", zap.String("error", err.Error()))
			}
		}(natsDriver)
		publish = func(ctx context.Context, value []byte) error {
			return natsDriver.Produce(ctx, cfg.NatsConfig.ProducerSubject, &epNats.Message{Data: value})
		}
	case "kafka":
		kafkaConfig := &epKafka.KafkaConfig{
			ConsumerGroupName:       cfg.KafkaConfig.ConsumerGroupName,
			BootstrapServers:        cfg.KafkaConfig.BootstrapServers,
			ConsumerAutoOffsetReset: &cfg.KafkaConfig.Offset,
		}
		kafkaDriver := epKafka.NewKafkaDriver(kafkaConfig)
		defer func(d drivers.Driver[*kafka.Message]) {
			if err := d.Close(); err != nil {
				log.Error("Error closing Kafka driver", zap.String("error", err.Error()))
			}
		}(kafkaDriver)
		publish = func(ctx context.Context, value []byte) error {
			return kafkaDriver.Produce(ctx, cfg.KafkaConfig.ProducerTopic, &kafka.Message{Value: value})
		}
	default:
		return fmt.Errorf("unknown transport %q: nats or kafka", opts.Transport)
	}

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

	log.Info("Starting artwork sync",
		zap.String("artwork", string(opts.Kind)),
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

			var artworkURL string
			if opts.Kind == ArtworkPoster {
				artworkURL, err = thetvdbService.GetSeriesPosterURL(ctx, *record.TheTVDBID)
			} else {
				artworkURL, err = thetvdbService.GetSeriesBannerURL(ctx, *record.TheTVDBID)
			}
			if err != nil {
				failed++
				log.Warn("Failed to fetch artwork",
					zap.String("anime_id", record.ID),
					zap.String("thetvdb_id", *record.TheTVDBID),
					zap.Error(err))
				continue
			}
			// Nothing of the right shape exists for this show. Publishing a
			// fallback of the wrong aspect would be worse than publishing
			// nothing: the frontend falls back per-anime on a missing object.
			if artworkURL == "" {
				noArtwork++
				continue
			}

			if opts.DryRun {
				published++
				log.Info("Would publish artwork",
					zap.String("artwork", string(opts.Kind)),
					zap.String("anime_id", record.ID),
					zap.String("url", artworkURL))
			} else {
				payload, err := json.Marshal(thetvdb_processor.ImagePayload{
					Data: thetvdb_processor.ImageSchema{
						ID:    record.ID,
						Name:  record.ID,
						URL:   artworkURL,
						Type:  string(opts.Kind),
						Force: opts.Force,
					},
				})
				if err != nil {
					failed++
					log.Warn("Failed to marshal artwork payload", zap.Error(err))
					continue
				}
				if err := publish(ctx, payload); err != nil {
					failed++
					log.Warn("Failed to publish artwork", zap.String("anime_id", record.ID), zap.Error(err))
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

	log.Info("Artwork sync complete",
		zap.String("artwork", string(opts.Kind)),
		zap.Int("processed", processed),
		zap.Int("published", published),
		zap.Int("noArtwork", noArtwork),
		zap.Int("failed", failed),
		zap.String("lastID", after))
	return nil
}
