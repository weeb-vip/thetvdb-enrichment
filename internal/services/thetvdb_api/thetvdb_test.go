package thetvdb_api_test

import (
	"context"
	"github.com/weeb-vip/thetvdb-enrichment/config"
	"github.com/weeb-vip/thetvdb-enrichment/internal/services/thetvdb_api"
	"log"
	"net/http"
	"testing"
)

func TestTheTVDBApiImpl_FindAnimeByTitle(t *testing.T) {
	t.Run("Test FindAnimeByTitle", func(t *testing.T) {
		conf := config.TheTVDBConfig{
			APIKey: "225f48ff-05e0-41f6-a479-42643a545c7a",
			APIPIN: "HMXIFHN5",
		}

		httpClient := &http.Client{}
		api := thetvdb_api.NewTheTVDBApi(conf, httpClient)
		ctx := context.Background()
		results, err := api.FindAnimeByTitle(ctx, "Spice and Wolf: Merchant Meets the Wise Wolf")
		if err != nil {
			t.Errorf("Error: %s", err)
		}

		log.Println(results)
	})
}
