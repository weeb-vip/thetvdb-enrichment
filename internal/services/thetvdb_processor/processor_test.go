package thetvdb_processor_test

import (
	"context"
	"encoding/json"
	"github.com/weeb-vip/thetvdb-enrichment/config"
	db2 "github.com/weeb-vip/thetvdb-enrichment/internal/db"
	anime2 "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime"
	anime "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime_episode"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_api"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_processor"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_service"
	"net/http"
	"testing"
)

func TestNewTheTVDBProcessor(t *testing.T) {
	t.Run("Test NewTheTVDBProcessor", func(t *testing.T) {
		ctx := context.Background()
		tvdbConfig := config.TheTVDBConfig{
			APIKey: "225f48ff-05e0-41f6-a479-42643a545c7a",
			APIPIN: "HMXIFHN5",
		}
		dbConfig := config.DBConfig{
			Host:     "aws.connect.psdb.cloud",
			User:     "wya0iyqypumz076br8bq",
			Password: "pscale_pw_HwgnMZNz7o2vSSMLREcREC7F3Cgr7YM8hiCgBQIxVKn",
			Port:     3306,
			DataBase: "test",
			SSLMode:  "true",
		}

		httpClient := &http.Client{}
		thetvdbAPI := thetvdb_api.NewTheTVDBApi(tvdbConfig, httpClient)

		thetvdbService := thetvdb_service.NewTheTVDBService(thetvdbAPI)
		db := db2.NewDB(dbConfig)
		episodeRepo := anime.NewAnimeEpisodeRepository(db)
		animeRepo := anime2.NewAnimeRepository(db)
		processor := thetvdb_processor.NewTheTVDBProcessor(thetvdbService, animeRepo, episodeRepo)
		if processor == nil {
			t.Errorf("Expected processor to be not nil")
		}

		jsonString := `{
				"id": "94a36163-79b9-4fc2-9bbd-f258da832111",
				"anime_id": "33c70baf-89bd-4e11-90c6-f56f25a71678",
				"thetvdb_link_id": "424435",
				"season": 2
}`
		var payload thetvdb_processor.Schema
		err := json.Unmarshal([]byte(jsonString), &payload)
		processor.Process(ctx, thetvdb_processor.Payload{
			Data: payload,
		})
		if err != nil {
			t.Errorf("Error: %s", err)
		}
	})
	//t.Run("Test NewTheTVDBProcessor2", func(t *testing.T) {
	//	ctx := context.Background()
	//	tvdbConfig := config.TheTVDBConfig{
	//		APIKey: "225f48ff-05e0-41f6-a479-42643a545c7a",
	//		APIPIN: "HMXIFHN5",
	//	}
	//	dbConfig := config.DBConfig{
	//		Host:     "aws.connect.psdb.cloud",
	//		User:     "wya0iyqypumz076br8bq",
	//		Password: "pscale_pw_HwgnMZNz7o2vSSMLREcREC7F3Cgr7YM8hiCgBQIxVKn",
	//		Port:     3306,
	//		DataBase: "test",
	//		SSLMode:  "true",
	//	}
	//
	//	httpClient := &http.Client{}
	//	thetvdbAPI := thetvdb_api.NewTheTVDBApi(tvdbConfig, httpClient)
	//
	//	thetvdbService := thetvdb_service.NewTheTVDBService(thetvdbAPI)
	//	db := db2.NewDB(dbConfig)
	//	episodeRepo := anime.NewAnimeEpisodeRepository(db)
	//	processor := thetvdb_processor.NewTheTVDBProcessor(thetvdbService, episodeRepo)
	//	if processor == nil {
	//		t.Errorf("Expected processor to be not nil")
	//	}
	//
	//	jsonString := `
	//{
	//  "id": "05950632-9b9d-4d47-8f02-65246081baaa",
	//  "title_en": "Spice and Wolf II",
	//  "title_jp": "狼と香辛料II",
	//  "title_romaji": null,
	//  "title_kanji": null,
	//  "type": "TV",
	//  "image_url": "https://cdn.myanimelist.net/images/anime/6/59399.jpg",
	//  "synopsis": "Traveling merchant Kraft Lawrence continues his northward journey with wolf goddess Holo, in search of her lost home of Yoitsu. Lawrence and his sharp-witted partner continue to make some small profits along the way, while slowly uncovering more information about Holo's hometown. However, the road to Yoitsu is a bumpy one filled with many troubles—Lawrence runs into a charming young fellow merchant who has his eyes set on the female wolf companion, and he begins to doubt if Holo will remain by his side; he and the goddess will also have to consider precarious and risky business deals as Lawrence strives to achieve his dream of becoming a shopowner. All the while, with his determination tested at every turn during his journey, Lawrence must question his relationship with Holo, take on business ventures, and ask himself whether it is time for him and Holo to go their separate ways.\n\n[Written by MAL Rewrite]",
	//  "episodes": 12,
	//  "status": "Finished Airing",
	//  "duration": "23 min. per ep.",
	//  "broadcast": "Thursdays at 01:15 (JST)",
	//  "source": "Light novel",
	//  "rating": "8.31",
	//  "start_date": "2009-07-09 04:00:00.000000 +00:00",
	//  "end_date": "2009-09-24 04:00:00.000000 +00:00",
	//  "title_synonyms": "[\"Ookami to Koushinryou 2nd Season\",\" Spice and Wolf 2nd Season\"]",
	//  "genres": "[\"Adventure\",\"Drama\",\"Fantasy\",\"Romance\"]",
	//  "licensors": "[\"Funimation\"]",
	//  "studios": "[\"Brain's Base\",\"       Marvy Jack\"]",
	//  "anidbid": "6169",
	//  "ranking": 256
	//}`
	//	var payload thetvdb_processor.Schema
	//	err := json.Unmarshal([]byte(jsonString), &payload)
	//	processor.Process(ctx, thetvdb_processor.Payload{
	//		After: &payload,
	//	})
	//	if err != nil {
	//		t.Errorf("Error: %s", err)
	//	}
	//})
}
