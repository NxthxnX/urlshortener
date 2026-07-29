package config

import (
	"errors"
	"flag"
	"net/url"
	"os"
	"path/filepath"

	"github.com/NxthxnX/urlshortener/internal/urlutils"
)

type webURL string

type Config struct {
	ServAddr        string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
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

func ParseFlags() Config {
	var options struct {
		servAddr        string
		baseURL         webURL
		fileStoragePath string
		databaseDSN     string
	}

	options.baseURL = webURL("http://localhost:8080")

	flag.StringVar(&options.servAddr, "a", "localhost:8080", "server address")
	flag.Var(&options.baseURL, "b", "result URL address")
	flag.StringVar(&options.fileStoragePath, "f", filepath.Join("tmp", "short-url-db.json"), "file storage path")
	flag.StringVar(&options.databaseDSN, "d", "", "database connection string")

	flag.Parse()

	if envServAddr := os.Getenv("SERVER_ADDRESS"); envServAddr != "" {
		options.servAddr = envServAddr
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		options.baseURL = webURL(envBaseURL)
	}
	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		options.fileStoragePath = envFileStoragePath
	}
	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		options.databaseDSN = envDatabaseDSN
	}

	return Config{
		ServAddr:        options.servAddr,
		BaseURL:         string(options.baseURL),
		FileStoragePath: options.fileStoragePath,
		DatabaseDSN:     options.databaseDSN,
	}
}
