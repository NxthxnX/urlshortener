package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NxthxnX/urlshortener/internal/repository"
	"github.com/NxthxnX/urlshortener/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	mockOriginalURL = "https://www.example.com"
	mockID          = "abcd1234"
)

type mockShortener struct {
	mock.Mock
}

func (m *mockShortener) Shorten(originalURL string) (string, error) {
	args := m.Called(originalURL)
	return args.String(0), args.Error(1)
}

func (m *mockShortener) Expand(id string) (string, bool) {
	args := m.Called(id)
	return args.String(0), args.Bool(1)
}

func TestShortenHandler(t *testing.T) {
	type want struct {
		code          int
		response      string
		contentType   string
		contentLength string
	}
	tests := []struct {
		name        string
		originalURL string
		id          string
		err         error
		want        want
	}{
		{
			name:        "Full https ULR",
			originalURL: mockOriginalURL,
			id:          mockID,
			err:         nil,
			want: want{
				code:          http.StatusCreated,
				response:      "http://localhost:8080/" + mockID,
				contentType:   "text/plain; charset=utf-8",
				contentLength: "30",
			},
		},
		{
			name:        "Full http ULR",
			originalURL: "http://httpbin.org",
			id:          mockID,
			err:         nil,
			want: want{
				code:          http.StatusCreated,
				response:      "http://localhost:8080/" + mockID,
				contentType:   "text/plain; charset=utf-8",
				contentLength: "30",
			},
		},
		{
			name:        "Short ULR",
			originalURL: "example.com",
			id:          mockID,
			err:         nil,
			want: want{
				code:          http.StatusCreated,
				response:      "http://localhost:8080/" + mockID,
				contentType:   "text/plain; charset=utf-8",
				contentLength: "30",
			},
		},
		{
			name:        "ULR with queries",
			originalURL: mockOriginalURL + "/example?test=go&test=lang",
			id:          mockID,
			err:         nil,
			want: want{
				code:          http.StatusCreated,
				response:      "http://localhost:8080/" + mockID,
				contentType:   "text/plain; charset=utf-8",
				contentLength: "30",
			},
		},
		{
			name:        "Empty URL",
			originalURL: "",
			id:          mockID,
			err:         nil,
			want: want{
				code:          http.StatusBadRequest,
				response:      "Empty URL\n",
				contentType:   "text/plain; charset=utf-8",
				contentLength: "",
			},
		},
		{
			name:        "Generation has failed",
			originalURL: mockOriginalURL,
			id:          "",
			err:         io.ErrShortBuffer,
			want: want{
				code:          http.StatusBadRequest,
				response:      "short buffer\n",
				contentType:   "text/plain; charset=utf-8",
				contentLength: "",
			},
		},
		{
			name:        "Invalid URL with spaces",
			originalURL: "ex ample.com",
			id:          mockID,
			err:         nil,
			want: want{
				code:          http.StatusBadRequest,
				response:      "Invalid URL format\n",
				contentType:   "text/plain; charset=utf-8",
				contentLength: "",
			},
		},
		{
			name:        "Incomplete protocol",
			originalURL: "http:/invalid",
			id:          mockID,
			err:         nil,
			want: want{
				code:          http.StatusBadRequest,
				response:      "Invalid URL format\n",
				contentType:   "text/plain; charset=utf-8",
				contentLength: "",
			},
		},
		{
			name:        "URL only with protocol",
			originalURL: "http://",
			id:          mockID,
			err:         nil,
			want: want{
				code:          http.StatusBadRequest,
				response:      "Invalid URL format\n",
				contentType:   "text/plain; charset=utf-8",
				contentLength: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.NewReader(tt.originalURL)
			req := httptest.NewRequest(http.MethodPost, "/", body)
			w := httptest.NewRecorder()

			var expectedURLForMock string
			if tt.originalURL == "example.com" {
				expectedURLForMock = "http://example.com"
			} else {
				expectedURLForMock = tt.originalURL
			}

			mockedShortener := new(mockShortener)
			mockedShortener.On("Shorten", expectedURLForMock).Return(tt.id, tt.err)
			h := NewHandler(mockedShortener)

			h.shortenHandler(w, req)

			res := w.Result()

			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)

			assert.Equal(t, tt.want.response, string(resBody))
			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.contentLength, res.Header.Get("Content-Length"))
		})
	}
}

func TestExpandHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
		location    string
	}
	tests := []struct {
		name        string
		originalURL string
		id          string
		ok          bool
		want        want
	}{
		{
			name:        "Full https ULR",
			originalURL: mockOriginalURL,
			id:          mockID,
			ok:          true,
			want: want{
				code:        http.StatusTemporaryRedirect,
				response:    "<a href=\"" + mockOriginalURL + "\">Temporary Redirect</a>.\n\n",
				contentType: "text/html; charset=utf-8",
				location:    mockOriginalURL,
			},
		},
		{
			name:        "Full http ULR",
			originalURL: "http://httpbin.org",
			id:          mockID,
			ok:          true,
			want: want{
				code:        http.StatusTemporaryRedirect,
				response:    "<a href=\"http://httpbin.org\">Temporary Redirect</a>.\n\n",
				contentType: "text/html; charset=utf-8",
				location:    "http://httpbin.org",
			},
		},
		{
			name:        "ULR with queries",
			originalURL: mockOriginalURL + "/example?test=go&test=lang",
			id:          mockID,
			ok:          true,
			want: want{
				code:        http.StatusTemporaryRedirect,
				response:    "<a href=\"" + mockOriginalURL + "/example?test=go&amp;test=lang\">Temporary Redirect</a>.\n\n",
				contentType: "text/html; charset=utf-8",
				location:    mockOriginalURL + "/example?test=go&test=lang",
			},
		},
		{
			name:        "Failed expansion",
			originalURL: "",
			id:          mockID,
			ok:          false,
			want: want{
				code:        http.StatusBadRequest,
				response:    "Not found\n",
				contentType: "text/plain; charset=utf-8",
				location:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/"+tt.id, nil)
			w := httptest.NewRecorder()

			mockedShortener := new(mockShortener)
			mockedShortener.On("Expand", tt.id).Return(tt.originalURL, tt.ok)
			h := NewHandler(mockedShortener)

			h.expandHandler(w, req)

			res := w.Result()

			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)

			assert.Equal(t, tt.want.response, string(resBody))
			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.location, res.Header.Get("Location"))
		})
	}
}

func TestServeHTTP(t *testing.T) {
	type want struct {
		code   int
		body   string
		header map[string]string
	}
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   want
	}{
		{
			name:   "POST /",
			method: http.MethodPost,
			path:   "/",
			body:   mockOriginalURL,
			want: want{
				code: http.StatusCreated,
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
		{
			name:   "POST / empty body",
			method: http.MethodPost,
			path:   "/",
			body:   "",
			want: want{
				code: http.StatusBadRequest,
				body: "Empty URL\n",
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
		{
			name:   "POST /some-path",
			method: http.MethodPost,
			path:   "/some-path",
			body:   mockOriginalURL,
			want: want{
				code: http.StatusBadRequest,
				body: "Not found\n",
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
		{
			name:   "GET /{mockID}",
			method: http.MethodGet,
			path:   "/" + mockID,
			want: want{
				code: http.StatusTemporaryRedirect,
				header: map[string]string{
					"Location": mockOriginalURL,
				},
			},
		},
		{
			name:   "GET /",
			method: http.MethodGet,
			path:   "/",
			want: want{
				code: http.StatusBadRequest,
				body: "Missing ID\n",
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
		{
			name:   "GET /nonexistent",
			method: http.MethodGet,
			path:   "/nonexistent",
			want: want{
				code: http.StatusBadRequest,
				body: "Not found\n",
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
		{
			name:   "PUT /",
			method: http.MethodPut,
			path:   "/",
			want: want{
				code: http.StatusBadRequest,
				body: "Method not allowed\n",
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
		{
			name:   "DELETE /",
			method: http.MethodDelete,
			path:   "/",
			want: want{
				code: http.StatusBadRequest,
				body: "Method not allowed\n",
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemoryRepository()
			svc := service.NewShortenerService(repo)
			h := NewHandler(svc)

			if tt.method == http.MethodGet && tt.path == "/"+mockID {
				repo.Save(mockID, mockOriginalURL)
			}

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			res := w.Result()
			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.body != "" {
				assert.Equal(t, tt.want.body, string(resBody))
			}

			for key, expectedVal := range tt.want.header {
				assert.Equal(t, expectedVal, res.Header.Get(key))
			}
		})
	}
}
