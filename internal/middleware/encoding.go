package middleware

import (
	"mime"
	"net/http"
	"strings"

	"github.com/NxthxnX/urlshortener/internal/parser"
)

var defaultSupportedEncodings = []string{"gzip"}

var compressibleContentTypes = map[string]struct{}{
	"application/javascript": {},
	"application/json":       {},
	"text/css":               {},
	"text/html":              {},
	"text/plain":             {},
	"text/xml":               {},
}

func isCompressibleContentType(contentType string) bool {
	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}

	_, ok := compressibleContentTypes[strings.ToLower(mediaType)]
	return ok
}

// WithEncoding negotiates Content-Encoding on requests and Accept-Encoding on responses.
func WithEncoding(h http.Handler) http.Handler {
	return withEncoding(h, defaultSupportedEncodings)
}

func withEncoding(next http.Handler, supported []string) http.Handler {
	encodeFn := func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err := newGzipRequestReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = reader
			defer reader.Close()
		case "":
		default:
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		compressible := isCompressibleContentType(r.Header.Get("Content-Type"))

		switch parser.SelectAcceptEncoding(r.Header.Values("Accept-Encoding"), supported) {
		case "gzip":
			if compressible {
				gzipCompress(next).ServeHTTP(w, r)
			} else {
				next.ServeHTTP(w, r)
			}
		case "identity":
			next.ServeHTTP(w, r)
		case "":
			w.WriteHeader(http.StatusNotAcceptable)
			return
		default:
			next.ServeHTTP(w, r)
		}
	}
	return http.HandlerFunc(encodeFn)
}

func gzipCompress(next http.Handler) http.Handler {
	gzipFn := func(w http.ResponseWriter, r *http.Request) {
		gw := newGzipResponseWriter(w)
		defer gw.Close()
		next.ServeHTTP(gw, r)
	}
	return http.HandlerFunc(gzipFn)
}
