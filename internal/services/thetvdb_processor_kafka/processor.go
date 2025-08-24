package thetvdb_processor_kafka

import (
	"github.com/ThatCatDev/ep/v2/event"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	anime2 "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime"
	anime "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime_episode"
	"github.com/weeb-vip/thetvdb-enrichment/internal/logger"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_service"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"time"
)

type TheTVDBProcessor interface {
	Process(ctx context.Context, data event.Event[*kafka.Message, Payload]) (event.Event[*kafka.Message, Payload], error)
}

type TheTVDBProcessorImpl struct {
	theTVDBService thetvdb_service.TheTVDBService
	episodeRepo    anime.AnimeEpisodeRepositoryImpl
	animeRepo      anime2.AnimeRepositoryImpl
}

func NewTheTVDBProcessor(theTVDBService thetvdb_service.TheTVDBService, animeRepo anime2.AnimeRepositoryImpl, episodeRepo anime.AnimeEpisodeRepositoryImpl) TheTVDBProcessor {
	return &TheTVDBProcessorImpl{
		theTVDBService: theTVDBService,
		animeRepo:      animeRepo,
		episodeRepo:    episodeRepo,
	}
}

func (p *TheTVDBProcessorImpl) Process(ctx context.Context, data event.Event[*kafka.Message, Payload]) (event.Event[*kafka.Message, Payload], error) {
	log := logger.FromCtx(ctx)

	payload := data.Payload

	animeName := "unknown"

	animeData, err := p.animeRepo.FindById(payload.Data.AnimeID)
	if err != nil {
		return data, err
	}
	if animeData != nil {
		if animeData.TitleEn != nil {
			animeName = *animeData.TitleEn
		}
		if animeData.TitleJp != nil {
			animeName = *animeData.TitleJp
		}
	}

	log.Info("Processing record", zap.String("id", payload.Data.AnimeID), zap.String("animeName", animeName), zap.String("thetvdb_link_id", payload.Data.TheTVDBLinkID), zap.Int("season", payload.Data.Season))

	season := payload.Data.Season

	animeWithEpisodes, err := p.theTVDBService.GetEpisodesBySeriesID(ctx, payload.Data.TheTVDBLinkID)
	if err != nil {
		return data, err
	}

	// clear out episodes
	err = p.episodeRepo.DeleteByAnimeID(ctx, payload.Data.AnimeID)
	if err != nil {
		log.Error("Failed to delete episodes", zap.Error(err))
		return data, err
	}

	if animeWithEpisodes != nil {
		log.Info("Episodes found", zap.Int("count", len(*animeWithEpisodes)))

		// update record
		for _, episode := range *animeWithEpisodes {
			// if season is 0 then its a special so we skip it
			//if episode.SeasonNumber != nil && *episode.SeasonNumber == 0 {
			//	continue
			//}
			// convert int to string

			if episode.SeasonNumber != nil && *episode.SeasonNumber != season {
				continue
			}
			var titleEN *string
			if episode.Translations["eng"] != nil {
				titleEN = episode.Translations["eng"].Name
			}
			var titleJP *string
			if episode.Translations["jpn"] != nil {
				titleJP = episode.Translations["jpn"].Name
			}

			var episodeAired *time.Time

			if episode.Aired != nil {
				aired, err := time.Parse("2006-01-02", *episode.Aired)
				if err != nil {
					return data, err
				}

				episodeAired = &aired
			}

			var Synopsis *string
			if episode.Translations["eng"] != nil {
				Synopsis = episode.Translations["eng"].Overview
			} else {
				Synopsis = episode.Translations["jpn"].Overview
			}
			episodeNumber := *episode.Number
			episodeRecord := &anime.AnimeEpisode{
				AnimeID:  &payload.Data.AnimeID,
				Episode:  &episodeNumber,
				TitleEn:  titleEN,
				TitleJp:  titleJP,
				Aired:    episodeAired,
				Synopsis: Synopsis,
			}
			err := p.episodeRepo.Upsert(ctx, episodeRecord)
			if err != nil {
				return data, err
			}
		}

	}

	//if payload.Before != nil && payload.After == nil {
	//	// new record
	//	if payload.Before.ImageUrl == nil {
	//		return nil
	//	}
	//	// download image
	//	resp, err := http.Get(*payload.Before.ImageUrl)
	//	if err != nil {
	//		return err
	//	}
	//	defer resp.Body.Close()
	//	imageData, err := io.ReadAll(resp.Body)
	//	if err != nil {
	//		return err
	//	}
	//
	//	// save to storage
	//	err = p.Storage.Put(ctx, imageData, payload.Before.Id)
	//	if err != nil {
	//		return err
	//	}
	//
	//	return nil
	//}

	return data, nil
}
