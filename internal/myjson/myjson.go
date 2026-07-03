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
