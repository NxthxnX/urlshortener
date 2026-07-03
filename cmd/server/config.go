package main

import (
	"errors"
	"flag"
	"net/url"
	"os"

	"github.com/NxthxnX/urlshortener/internal/urlutils"
)

type webURL string

var options struct {
	servAddr string
	baseURL  webURL
}

func (link *webURL) String() string {
	return string(*link)
}

func (link *webURL) Set(value string) error {
	normalized, err := urlutils.Normalize(value)
	if err != nil {
		return errors.New("invalid URL format")
	}

	parsedURL, err := url.ParseRequestURI(normalized)
	if err != nil {
		return errors.New("invalid URL format")
	}

	*link = webURL(parsedURL.String())

	return nil
}

func parseFlags() {
	options.baseURL = webURL("http://localhost:8080")

	flag.StringVar(&options.servAddr, "a", "localhost:8080", "server address")
	flag.Var(&options.baseURL, "b", "result URL address")

	flag.Parse()

	if envServAddr := os.Getenv("SERVER_ADDRESS"); envServAddr != "" {
		options.servAddr = envServAddr
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		options.baseURL = webURL(envBaseURL)
	}
}
