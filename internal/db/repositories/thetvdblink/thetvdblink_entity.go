package thetvdblink

import "time"

// TheTVDBLink ties one anime to one season of one TheTVDB series.
//
// The table is owned by scraper-api, which writes it from the admin panel when
// someone links a show by hand. This service reads and adds to it: the season
// backfill derives the same fact from air dates for the thousands nobody has
// linked.
//
// anime_id is varchar here rather than uuid, matching the column as
// scraper-api created it.
type TheTVDBLink struct {
	ID           string    `gorm:"column:id;primaryKey" json:"id"`
	AnimeID      string    `gorm:"column:anime_id" json:"anime_id"`
	TheTVDBID    string    `gorm:"column:thetvdb_id" json:"thetvdb_id"`
	SeasonNumber int       `gorm:"column:season_number" json:"season_number"`
	Name         *string   `gorm:"column:name;null" json:"name"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (TheTVDBLink) TableName() string {
	return "thetvdb_link"
}
