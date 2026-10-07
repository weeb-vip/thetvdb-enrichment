package thetvdb_processor

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
	// ID is what image-sync keys the object by. Name is still sent so a
	// consumer that has not picked up the id yet keeps working.
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
	// Force asks image-sync to store and announce the image even when it
	// already holds this source: the artwork sync's --force.
	Force bool `json:"force,omitempty"`
}

// ImagePayload is the envelope the image-sync kafka consumer expects
type ImagePayload struct {
	Data ImageSchema `json:"data"`
}
