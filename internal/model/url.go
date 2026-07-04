package model

// URLRecord represents a persisted shortened URL entry.
type URLRecord struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
