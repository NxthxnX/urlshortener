package urlutils

import (
	"errors"
	"net/url"
	"strings"
)

// ErrInvalidURL is returned when a URL cannot be parsed or is missing required components.
var ErrInvalidURL = errors.New("invalid URL")

// Normalize validates a URL and prepends http:// when no scheme is present.
func Normalize(rawURL string) (string, error) {
	candidate := rawURL

	parsedURL, err := url.ParseRequestURI(candidate)
	if err != nil || parsedURL.Scheme == "" {
		if !strings.HasPrefix(candidate, "http://") && !strings.HasPrefix(candidate, "https://") {
			candidate = "http://" + candidate
		}

		parsedURL, err = url.ParseRequestURI(candidate)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return "", ErrInvalidURL
		}
	} else if parsedURL.Host == "" {
		return "", ErrInvalidURL
	}

	return candidate, nil
}
