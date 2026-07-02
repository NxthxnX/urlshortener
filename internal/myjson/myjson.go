package myjson

//go:generate easyjson -all myjson.go
// apiShortenRequest represents the request body for the /api/shorten endpoint.
type ApiShortenRequest struct {
	URL string `json:"url"`
}

// apiShortenResponse represents the response body for the /api/shorten endpoint.
type ApiShortenResponse struct {
	Result string `json:"result"`
}
