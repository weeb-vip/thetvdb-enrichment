package thetvdb_api

type TranslationSimple map[string]interface{}

type RemoteID struct {
	ID         *string `json:"id,omitempty"`
	Type       *int64  `json:"type,omitempty"`
	SourceName *string `json:"sourceName,omitempty"`
}

type SearchResult struct {
	Aliases              []string          `json:"aliases,omitempty"`
	Companies            []string          `json:"companies,omitempty"`
	CompanyType          *string           `json:"companyType,omitempty"`
	Country              *string           `json:"country,omitempty"`
	Director             *string           `json:"director,omitempty"`
	FirstAirTime         *string           `json:"first_air_time,omitempty"`
	Genres               []string          `json:"genres,omitempty"`
	ID                   *string           `json:"id,omitempty"`
	ImageURL             *string           `json:"image_url,omitempty"`
	Name                 *string           `json:"name,omitempty"`
	IsOfficial           *bool             `json:"is_official,omitempty"`
	NameTranslated       *string           `json:"name_translated,omitempty"`
	Network              *string           `json:"network,omitempty"`
	ObjectID             *string           `json:"objectID,omitempty"`
	OfficialList         *string           `json:"officialList,omitempty"`
	Overview             *string           `json:"overview,omitempty"`
	Overviews            TranslationSimple `json:"overviews,omitempty"`
	OverviewTranslated   []string          `json:"overview_translated,omitempty"`
	Poster               *string           `json:"poster,omitempty"`
	Posters              []string          `json:"posters,omitempty"`
	PrimaryLanguage      *string           `json:"primary_language,omitempty"`
	RemoteIDs            []RemoteID        `json:"remote_ids,omitempty"`
	Status               *string           `json:"status,omitempty"`
	Slug                 *string           `json:"slug,omitempty"`
	Studios              []string          `json:"studios,omitempty"`
	Title                *string           `json:"title,omitempty"`
	Thumbnail            *string           `json:"thumbnail,omitempty"`
	Translations         TranslationSimple `json:"translations,omitempty"`
	TranslationsWithLang []string          `json:"translationsWithLang,omitempty"`
	TVDBID               *string           `json:"tvdb_id,omitempty"`
	Type                 *string           `json:"type,omitempty"`
	Year                 *string           `json:"year,omitempty"`
}

type Alias struct {
	Language string `json:"language,omitempty"` // A 3-4 character string indicating the language of the alias, as defined in Language.
	Name     string `json:"name,omitempty"`     // A string containing the alias itself.
}

type EpisodeBaseRecord struct {
	AbsoluteNumber       *int               `json:"absoluteNumber,omitempty"`
	Aired                *string            `json:"aired,omitempty"`
	AirsAfterSeason      *int               `json:"airsAfterSeason,omitempty"`
	AirsBeforeEpisode    *int               `json:"airsBeforeEpisode,omitempty"`
	AirsBeforeSeason     *int               `json:"airsBeforeSeason,omitempty"`
	FinaleType           *string            `json:"finaleType,omitempty"` // season, midseason, or series
	ID                   *int64             `json:"id,omitempty"`
	Image                *string            `json:"image,omitempty"`
	ImageType            *int               `json:"imageType,omitempty"`
	IsMovie              *int64             `json:"isMovie,omitempty"`
	LastUpdated          *string            `json:"lastUpdated,omitempty"`
	LinkedMovie          *int               `json:"linkedMovie,omitempty"`
	Name                 *string            `json:"name,omitempty"`
	NameTranslations     []string           `json:"nameTranslations,omitempty"`
	Number               *int               `json:"number,omitempty"`
	Overview             *string            `json:"overview,omitempty"`
	OverviewTranslations []string           `json:"overviewTranslations,omitempty"`
	Runtime              *int               `json:"runtime,omitempty"`
	SeasonNumber         *int               `json:"seasonNumber,omitempty"`
	Seasons              []SeasonBaseRecord `json:"seasons,omitempty"`
	SeriesID             *int64             `json:"seriesId,omitempty"`
	SeasonName           *string            `json:"seasonName,omitempty"`
	Year                 *string            `json:"year,omitempty"`
}

type CompanyRelationShip struct {
	ID       *int    `json:"id,omitempty"`
	TypeName *string `json:"typeName,omitempty"`
}

type ParentCompany struct {
	ID       *int                 `json:"id,omitempty"`
	Name     *string              `json:"name,omitempty"`
	Relation *CompanyRelationShip `json:"relation,omitempty"`
}

type TagOption struct {
	HelpText *string `json:"helpText,omitempty"`
	ID       *int64  `json:"id,omitempty"`
	Name     *string `json:"name,omitempty"`
	Tag      *int64  `json:"tag,omitempty"`
	TagName  *string `json:"tagName,omitempty"`
}

type Company struct {
	ActiveDate           *string        `json:"activeDate,omitempty"`
	Aliases              []Alias        `json:"aliases,omitempty"`
	Country              *string        `json:"country,omitempty"`
	ID                   *int64         `json:"id,omitempty"`
	InactiveDate         *string        `json:"inactiveDate,omitempty"`
	Name                 *string        `json:"name,omitempty"`
	NameTranslations     []string       `json:"nameTranslations,omitempty"`
	OverviewTranslations []string       `json:"overviewTranslations,omitempty"`
	PrimaryCompanyType   *int64         `json:"primaryCompanyType,omitempty"`
	Slug                 *string        `json:"slug,omitempty"`
	ParentCompany        *ParentCompany `json:"parentCompany,omitempty"`
	TagOptions           []TagOption    `json:"tagOptions,omitempty"`
}

type Companies struct {
	Studio         []Company `json:"studio,omitempty"`
	Network        []Company `json:"network,omitempty"`
	Production     []Company `json:"production,omitempty"`
	Distributor    []Company `json:"distributor,omitempty"`
	SpecialEffects []Company `json:"special_effects,omitempty"`
}

type SeasonType struct {
	AlternateName *string `json:"alternateName,omitempty"`
	ID            *int64  `json:"id,omitempty"`
	Name          *string `json:"name,omitempty"`
	Type          *string `json:"type,omitempty"`
}

type SeasonBaseRecord struct {
	ID                   *int64      `json:"id,omitempty"`
	Image                *string     `json:"image,omitempty"`
	ImageType            *int        `json:"imageType,omitempty"`
	LastUpdated          *string     `json:"lastUpdated,omitempty"`
	Name                 *string     `json:"name,omitempty"`
	NameTranslations     []string    `json:"nameTranslations,omitempty"`
	Number               *int64      `json:"number,omitempty"`
	OverviewTranslations []string    `json:"overviewTranslations,omitempty"`
	Companies            *Companies  `json:"companies,omitempty"`
	SeriesID             *int64      `json:"seriesId,omitempty"`
	Type                 *SeasonType `json:"type,omitempty"`
	Year                 *string     `json:"year,omitempty"`
}

type Status struct {
	ID          *int64  `json:"id,omitempty"`
	KeepUpdated *bool   `json:"keepUpdated,omitempty"`
	Name        *string `json:"name,omitempty"`
	RecordType  *string `json:"recordType,omitempty"`
}

type SeriesBaseRecord struct {
	Aliases              []Alias             `json:"aliases,omitempty"`
	AverageRuntime       *int                `json:"averageRuntime,omitempty"`
	Country              *string             `json:"country,omitempty"`
	DefaultSeasonType    *int64              `json:"defaultSeasonType,omitempty"`
	Episodes             []EpisodeBaseRecord `json:"episodes,omitempty"`
	FirstAired           *string             `json:"firstAired,omitempty"`
	ID                   *int                `json:"id,omitempty"`
	Image                *string             `json:"image,omitempty"`
	IsOrderRandomized    *bool               `json:"isOrderRandomized,omitempty"`
	LastAired            *string             `json:"lastAired,omitempty"`
	LastUpdated          *string             `json:"lastUpdated,omitempty"`
	Name                 *string             `json:"name,omitempty"`
	NameTranslations     []string            `json:"nameTranslations,omitempty"`
	NextAired            *string             `json:"nextAired,omitempty"`
	OriginalCountry      *string             `json:"originalCountry,omitempty"`
	OriginalLanguage     *string             `json:"originalLanguage,omitempty"`
	OverviewTranslations []string            `json:"overviewTranslations,omitempty"`
	Score                *float64            `json:"score,omitempty"`
	Slug                 *string             `json:"slug,omitempty"`
	Status               *Status             `json:"status,omitempty"`
	Year                 *string             `json:"year,omitempty"`
}

type Translation struct {
	Aliases   []string `json:"aliases,omitempty"`
	IsAlias   *bool    `json:"isAlias,omitempty"`
	IsPrimary *bool    `json:"isPrimary,omitempty"`
	Language  *string  `json:"language,omitempty"`
	Name      *string  `json:"name,omitempty"`
	Overview  *string  `json:"overview,omitempty"`
	// Only populated for movie translations. We disallow taglines without a title.
	Tagline *string `json:"tagline,omitempty"`
}

type GetSeriesEpisodesData struct {
	Episodes []EpisodeBaseRecord `json:"episodes,omitempty"`
	Series   SeriesBaseRecord    `json:"series,omitempty"`
}

type Response[T any] struct {
	Status *string `json:"status"`
	Data   T       `json:"data"`
}

type LoginResponse struct {
	Token *string `json:"token"`
}
