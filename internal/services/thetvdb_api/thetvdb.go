package thetvdb_api

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/weeb-vip/thetvdb-enrichment/config"
	"net/http"
	"net/url"
)

type TheTVDBApi interface {
	FindAnimeByTitle(ctx context.Context, title string) (*[]SearchResult, error)
	GetEpisodesBySeriesID(ctx context.Context, seriesID string) (*GetSeriesEpisodesData, error)
	GetEpisodeTranslation(ctx context.Context, episodeID string, lang string) (*Translation, error)
}

type TheTVDBApiImpl struct {
	client             *http.Client
	apiKey             string
	apiPin             string
	authorizationToken *string
	baseURL            string
}

func NewTheTVDBApi(cfg config.TheTVDBConfig, client *http.Client) TheTVDBApi {

	service := &TheTVDBApiImpl{
		baseURL: "https://api4.thetvdb.com/v4",
		apiKey:  cfg.APIKey,
		apiPin:  cfg.APIPIN,
		client:  client,
	}

	_ = service.login(context.Background())

	return service
}

func (t *TheTVDBApiImpl) FindAnimeByTitle(ctx context.Context, title string) (*[]SearchResult, error) {
	var responseData Response[[]SearchResult]
	titleUrlEncoded := url.QueryEscape(title)
	// create http request with authorization token
	req, err := http.NewRequest("GET", t.baseURL+"/search?query="+titleUrlEncoded, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+*t.authorizationToken)
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&responseData)

	if err != nil {
		return nil, err
	}

	return &responseData.Data, nil
}

func (t *TheTVDBApiImpl) login(ctx context.Context) error {
	var responseData Response[LoginResponse]
	body := struct {
		ApiKey string `json:"apikey"`
		ApiPin string `json:"pin"`
	}{
		ApiKey: t.apiKey,
		ApiPin: t.apiPin,
	}
	// body to json to bytes
	bodyBytes, err := json.Marshal(body)
	req, err := http.NewRequest("POST", t.baseURL+"/login", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&responseData)
	if err != nil {
		return err
	}

	t.authorizationToken = responseData.Data.Token
	return nil
}

func (t *TheTVDBApiImpl) GetEpisodesBySeriesID(ctx context.Context, seriesID string) (*GetSeriesEpisodesData, error) {
	var responseData Response[GetSeriesEpisodesData]
	// create http request with authorization token
	req, err := http.NewRequest("GET", t.baseURL+"/series/"+seriesID+"/episodes/default", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+*t.authorizationToken)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&responseData)
	if err != nil {
		return nil, err
	}

	return &responseData.Data, nil

}

func (t *TheTVDBApiImpl) GetEpisodeTranslation(ctx context.Context, episodeID string, lang string) (*Translation, error) {
	var responseData Response[Translation]
	// create http request with authorization token
	req, err := http.NewRequest("GET", t.baseURL+"/episodes/"+episodeID+"/translations/"+lang, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+*t.authorizationToken)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&responseData)

	if err != nil {
		return nil, err
	}

	return &responseData.Data, nil
}
