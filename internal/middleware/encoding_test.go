package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(data)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	return buf.Bytes()
}

func gunzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()

	zr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer zr.Close()

	out, err := io.ReadAll(zr)
	require.NoError(t, err)

	return out
}

func TestWithEncoding(t *testing.T) {
	const responseBody = `{"ok":true}`

	echoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if ct := r.Header.Get("Content-Type"); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	fixedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	})

	tests := []struct {
		name                 string
		handler              http.Handler
		contentEncoding      string
		contentType          string
		acceptEncoding       string
		body                 []byte
		wantStatus           int
		wantResponseEncoding string
		wantBody             string
		checkRequestBody     string
	}{
		{
			name:             "decompresses gzip request body",
			handler:          echoHandler,
			contentEncoding:  "gzip",
			contentType:      "text/plain",
			acceptEncoding:   "identity",
			body:             gzipBytes(t, []byte("hello")),
			wantStatus:       http.StatusOK,
			wantBody:         "hello",
			checkRequestBody: "hello",
		},
		{
			name:            "invalid gzip request body",
			handler:         echoHandler,
			contentEncoding: "gzip",
			contentType:     "text/plain",
			acceptEncoding:  "identity",
			body:            []byte("not gzip"),
			wantStatus:      http.StatusInternalServerError,
		},
		{
			name:            "unsupported request content encoding",
			handler:         echoHandler,
			contentEncoding: "deflate",
			contentType:     "text/plain",
			acceptEncoding:  "identity",
			body:            []byte("hello"),
			wantStatus:      http.StatusNotAcceptable,
		},
		{
			name:                 "compresses response when accept gzip and compressible content type",
			handler:              fixedHandler,
			contentType:          "application/json",
			acceptEncoding:       "gzip",
			wantStatus:           http.StatusOK,
			wantResponseEncoding: "gzip",
			wantBody:             responseBody,
		},
		{
			name:           "plain response when accept gzip but content type not compressible",
			handler:        fixedHandler,
			contentType:    "image/png",
			acceptEncoding: "gzip",
			wantStatus:     http.StatusOK,
			wantBody:       responseBody,
		},
		{
			name:           "plain response for identity accept encoding",
			handler:        fixedHandler,
			contentType:    "application/json",
			acceptEncoding: "identity",
			wantStatus:     http.StatusOK,
			wantBody:       responseBody,
		},
		{
			name:           "not acceptable when accept encoding rejected",
			handler:        fixedHandler,
			contentType:    "application/json",
			acceptEncoding: "*;q=0",
			wantStatus:     http.StatusNotAcceptable,
		},
		{
			name:                 "compressible content type with charset",
			handler:              fixedHandler,
			contentType:          "application/json; charset=utf-8",
			acceptEncoding:       "gzip",
			wantStatus:           http.StatusOK,
			wantResponseEncoding: "gzip",
			wantBody:             responseBody,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.handler

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(tt.body))
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			rec := httptest.NewRecorder()
			WithEncoding(handler).ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantResponseEncoding, rec.Header().Get("Content-Encoding"))

			if tt.checkRequestBody != "" {
				assert.Equal(t, tt.checkRequestBody, rec.Body.String())
			}

			if tt.wantBody == "" {
				return
			}

			body := rec.Body.Bytes()
			if rec.Header().Get("Content-Encoding") == "gzip" {
				body = gunzipBytes(t, body)
			}
			assert.Equal(t, tt.wantBody, string(body))
		})

	}
}
