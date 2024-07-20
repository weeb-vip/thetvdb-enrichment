package thetvdb_service

import (
	"context"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_api"
	"strconv"
)

type AnimeEpisodeWithTranslation struct {
	thetvdb_api.EpisodeBaseRecord
	Translations map[string]*thetvdb_api.Translation
}

type AnimeWithEpisodes struct {
	thetvdb_api.SearchResult
	Episodes []AnimeEpisodeWithTranslation
}

type TheTVDBService interface {
	FindAnime(ctx context.Context, title string, year string) (*AnimeWithEpisodes, error)
	GetEpisodesBySeriesID(ctx context.Context, seriesID string) (*[]AnimeEpisodeWithTranslation, error)
}

type TheTVDBServiceImpl struct {
	api thetvdb_api.TheTVDBApi
}

func NewTheTVDBService(api thetvdb_api.TheTVDBApi) TheTVDBService {
	return &TheTVDBServiceImpl{
		api: api,
	}
}

func (s *TheTVDBServiceImpl) FindAnime(ctx context.Context, title string, year string) (*AnimeWithEpisodes, error) {
	results, err := s.api.FindAnimeByTitle(ctx, title)
	if err != nil {
		return nil, err
	}
	var animeResult *thetvdb_api.SearchResult
	for _, result := range *results {
		if result.Year != nil && *result.Year == year {
			animeResult = &result
		}
	}
	if animeResult != nil {
		animeEpisodes, err := s.getAnimeEpisodes(ctx, *animeResult.TVDBID)
		if err != nil {
			return nil, err
		}
		return &AnimeWithEpisodes{
			SearchResult: *animeResult,
			Episodes:     *animeEpisodes,
		}, nil

	}

	return nil, nil
}

func (s *TheTVDBServiceImpl) getAnimeEpisodes(ctx context.Context, seriesID string) (*[]AnimeEpisodeWithTranslation, error) {
	episodes, err := s.api.GetEpisodesBySeriesID(ctx, seriesID)
	if err != nil {
		return nil, err
	}

	var episodesWithTranslations []AnimeEpisodeWithTranslation
	for _, episode := range episodes.Episodes {
		translations := make(map[string]*thetvdb_api.Translation)

		episodeID := strconv.FormatInt(*episode.ID, 10)
		for _, translation := range episode.NameTranslations {
			translationRes, err := s.getEpisodeTranslation(ctx, episodeID, translation)
			if err != nil {
				return nil, err
			}
			translations[translation] = translationRes
		}

		episodesWithTranslations = append(episodesWithTranslations, AnimeEpisodeWithTranslation{
			EpisodeBaseRecord: episode,
			Translations:      translations,
		})
	}

	return &episodesWithTranslations, nil
}

func (s *TheTVDBServiceImpl) getEpisodeTranslation(ctx context.Context, episodeID string, lang string) (*thetvdb_api.Translation, error) {
	translation, err := s.api.GetEpisodeTranslation(ctx, episodeID, lang)
	if err != nil {
		return nil, err
	}

	return translation, nil
}

func (s *TheTVDBServiceImpl) GetEpisodesBySeriesID(ctx context.Context, seriesID string) (*[]AnimeEpisodeWithTranslation, error) {
	return s.getAnimeEpisodes(ctx, seriesID)
}
