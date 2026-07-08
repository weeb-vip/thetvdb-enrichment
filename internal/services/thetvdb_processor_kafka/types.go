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

type ImageSchema struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

// ImagePayload is the envelope the image-sync kafka consumer expects
type ImagePayload struct {
	Data ImageSchema `json:"data"`
}
