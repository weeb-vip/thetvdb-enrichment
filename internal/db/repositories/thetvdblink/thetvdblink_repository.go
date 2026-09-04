package thetvdblink

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/weeb-vip/thetvdb-enrichment/internal/db"
)

type TheTVDBLinkRepositoryImpl interface {
	// FindByAnimeID returns the link for an anime, or nil when it has none.
	FindByAnimeID(ctx context.Context, animeID string) (*TheTVDBLink, error)
	// Upsert creates the link or updates its season, keyed on the anime.
	Upsert(ctx context.Context, link *TheTVDBLink) error
}

type TheTVDBLinkRepository struct {
	db *db.DB
}

func NewTheTVDBLinkRepository(db *db.DB) TheTVDBLinkRepositoryImpl {
	return &TheTVDBLinkRepository{db: db}
}

func (r *TheTVDBLinkRepository) FindByAnimeID(ctx context.Context, animeID string) (*TheTVDBLink, error) {
	var found TheTVDBLink
	err := r.db.DB.WithContext(ctx).Where("anime_id = ?", animeID).First(&found).Error
	if err != nil {
		// No link is the normal case for most of the catalogue, not an error.
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return &found, nil
}

// Upsert keys on the anime rather than on the row id.
//
// One anime belongs to one season of one series, so the anime is the natural
// key; writing a second row for the same anime would leave the mirror trigger
// picking arbitrarily between them.
func (r *TheTVDBLinkRepository) Upsert(ctx context.Context, link *TheTVDBLink) error {
	existing, err := r.FindByAnimeID(ctx, link.AnimeID)
	if err != nil {
		return err
	}

	if existing == nil {
		if link.ID == "" {
			link.ID = uuid.NewString()
		}

		return r.db.DB.WithContext(ctx).Create(link).Error
	}

	return r.db.DB.WithContext(ctx).
		Model(&TheTVDBLink{}).
		Where("id = ?", existing.ID).
		Updates(map[string]interface{}{
			"thetvdb_id":    link.TheTVDBID,
			"season_number": link.SeasonNumber,
			"name":          link.Name,
		}).Error
}
