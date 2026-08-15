package myjson

//go:generate easyjson -all myjson.go
// APIShortenRequest represents the request body for the /api/shorten endpoint.
type APIShortenRequest struct {
	URL string `json:"url"`
}

// APIShortenResponse represents the response body for the /api/shorten endpoint.
type APIShortenResponse struct {
	Result string `json:"result"`
}

// APIShortenBatchRequest represents one element of the batch request
// for the /api/shorten/batch endpoint.
type APIShortenBatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// APIShortenBatchResponse represents one element of the batch request
// for the /api/shorten/batch endpoint.
type APIShortenBatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
