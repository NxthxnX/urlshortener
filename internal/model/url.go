package model

// URLRecord represents a persisted shortened URL entry.
type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// URLPair represents a pair of short and original URLs.
type URLPair struct {
	ShortURL    string
	OriginalURL string
}
