package thetvdb_processor_kafka

type Schema struct {
	Id            string `json:"id"`
	AnimeID       string `json:"anime_id"`
	TheTVDBLinkID string `json:"thetvdb_link_id"`
	Season        int    `json:"season"`
}

type Payload struct {
	Data Schema `json:"data"`
}
