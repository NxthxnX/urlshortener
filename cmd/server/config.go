package main

import (
	"errors"
	"flag"
	"net/url"
	"strings"
)

type WebURL string

var options struct {
	servAddr string
	baseURL  WebURL
}

func (link *WebURL) String() string {
	return string(*link)
}

func (link *WebURL) Set(value string) error {
	parsedURL, err := url.ParseRequestURI(value)

	if err != nil || parsedURL.Scheme == "" {
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			value = "http://" + value
		}

		parsedURL, err = url.ParseRequestURI(value)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return errors.New("invalid URL format")
		}
	} else {
		if parsedURL.Host == "" {
			return errors.New("invalid URL format")
		}
	}

	*link = WebURL(parsedURL.String())

	return nil
}

func parseFlags() {
	options.baseURL = WebURL("http://localhost:8080")

	flag.StringVar(&options.servAddr, "a", "localhost:8080", "server address")
	flag.Var(&options.baseURL, "b", "result URL address")

	flag.Parse()
}
