package anime

import (
	"context"

	"github.com/weeb-vip/thetvdb-enrichment/internal/db"
)

type RECORD_TYPE string

type AnimeRepositoryImpl interface {
	Upsert(anime *Anime) error
	Delete(anime *Anime) error
	FindById(id string) (*Anime, error)
	// FindWithTheTVDBID pages through anime that carry a thetvdbid, ordered by
	// id so a paged walk is stable while the table is being written to.
	FindWithTheTVDBID(ctx context.Context, afterID string, limit int) ([]*Anime, error)
}

type AnimeRepository struct {
	db *db.DB
}

func NewAnimeRepository(db *db.DB) AnimeRepositoryImpl {
	return &AnimeRepository{db: db}
}

func (a *AnimeRepository) Upsert(anime *Anime) error {
	err := a.db.DB.Save(anime).Error
	if err != nil {
		return err
	}
	return nil
}

func (a *AnimeRepository) Delete(anime *Anime) error {
	err := a.db.DB.Delete(anime).Error
	if err != nil {
		return err
	}
	return nil
}

func (a *AnimeRepository) FindById(id string) (*Anime, error) {
	var anime Anime
	err := a.db.DB.Where("id = ?", id).First(&anime).Error
	if err != nil {
		return nil, err
	}
	return &anime, nil
}

// FindWithTheTVDBID returns anime with a non-empty thetvdbid, id-ordered and
// keyset-paged. Keyset rather than OFFSET so a long walk does not degrade and
// does not skip rows if the table shifts underneath it.
func (a *AnimeRepository) FindWithTheTVDBID(ctx context.Context, afterID string, limit int) ([]*Anime, error) {
	var records []*Anime
	q := a.db.DB.WithContext(ctx).
		Where("thetvdbid IS NOT NULL AND thetvdbid <> ''")
	if afterID != "" {
		q = q.Where("id > ?", afterID)
	}
	err := q.Order("id ASC").Limit(limit).Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}
