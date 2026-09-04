package anime

import (
	"context"
	"github.com/google/uuid"
	"github.com/weeb-vip/thetvdb-enrichment/internal/db"
	"time"
)

type RECORD_TYPE string

type AnimeEpisodeRepositoryImpl interface {
	Upsert(ctx context.Context, anime *AnimeEpisode) error
	Delete(ctx context.Context, anime *AnimeEpisode) error
	DeleteByAnimeID(ctx context.Context, animeID string) error
}

type AnimeEpisodeRepository struct {
	db *db.DB
}

func NewAnimeEpisodeRepository(db *db.DB) AnimeEpisodeRepositoryImpl {
	return &AnimeEpisodeRepository{db: db}
}

func (a *AnimeEpisodeRepository) Upsert(ctx context.Context, episode *AnimeEpisode) error {
	// find episode by anime_id and episode_number
	var existing AnimeEpisode
	err := a.db.DB.Where("anime_id = ? AND episode = ?", episode.AnimeID, episode.Episode).First(&existing).Error
	if err != nil {
		// if not found, create new with uuid
		episode.ID = uuid.New().String()
		err := a.db.DB.Create(episode).Error
		if err != nil {
			return err
		}
		return nil
	}
	// if found, update
	episode.ID = existing.ID
	episode.CreatedAt = existing.CreatedAt
	episode.UpdatedAt = time.Now()

	err = a.db.DB.Save(episode).Error
	if err != nil {
		return err
	}
	return nil
}

func (a *AnimeEpisodeRepository) Delete(ctx context.Context, episode *AnimeEpisode) error {
	err := a.db.DB.Delete(episode).Error
	if err != nil {
		return err
	}
	return nil
}

func (a *AnimeEpisodeRepository) DeleteByAnimeID(ctx context.Context, animeID string) error {
	err := a.db.DB.Where("anime_id = ?", animeID).Delete(&AnimeEpisode{}).Error
	if err != nil {
		return err
	}
	return nil
}
