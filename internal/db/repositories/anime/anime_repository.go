package anime

import "github.com/weeb-vip/thetvdb-enrichment/internal/db"

type RECORD_TYPE string

type AnimeRepositoryImpl interface {
	Upsert(anime *Anime) error
	Delete(anime *Anime) error
	FindById(id string) (*Anime, error)
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
