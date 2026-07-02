package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NxthxnX/urlshortener/internal/repository"
	"github.com/NxthxnX/urlshortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	mockResAddr     = "http://localhost:8080"
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
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name        string
		originalURL string
		id          string
		resAddr     string
		err         error
		want        want
	}{
		{
			name:        "Full https URL",
			originalURL: mockOriginalURL,
			id:          mockID,
			want: want{
				code:        http.StatusCreated,
				response:    "http://localhost:8080/" + mockID,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Full http URL",
			originalURL: "http://httpbin.org",
			id:          mockID,
			want: want{
				code:        http.StatusCreated,
				response:    "http://localhost:8080/" + mockID,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Short URL",
			originalURL: "example.com",
			id:          mockID,
			want: want{
				code:        http.StatusCreated,
				response:    "http://localhost:8080/" + mockID,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "URL with queries",
			originalURL: mockOriginalURL + "/example?test=go&test=lang",
			id:          mockID,
			want: want{
				code:        http.StatusCreated,
				response:    "http://localhost:8080/" + mockID,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Empty URL",
			originalURL: "",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Empty URL\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Generation has failed",
			originalURL: mockOriginalURL,
			err:         io.ErrShortBuffer,
			want: want{
				code:        http.StatusBadRequest,
				response:    "short buffer\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Invalid URL with spaces",
			originalURL: "ex ample.com",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Invalid URL format\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Incomplete protocol",
			originalURL: "http:/invalid",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Invalid URL format\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "URL only with protocol",
			originalURL: "http://",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Invalid URL format\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Different resAddr",
			originalURL: mockOriginalURL,
			id:          mockID,
			resAddr:     "http://different-host:9090",
			want: want{
				code:        http.StatusCreated,
				response:    "http://different-host:9090/" + mockID,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var expectedURLForMock string
			if strings.HasPrefix(tt.originalURL, "http://") || strings.HasPrefix(tt.originalURL, "https://") {
				expectedURLForMock = tt.originalURL
			} else if tt.originalURL != "" && tt.originalURL != "{invalid json" {
				expectedURLForMock = "http://" + tt.originalURL
			} else {
				expectedURLForMock = tt.originalURL
			}

			mockedShortener := new(mockShortener)
			if expectedURLForMock != "" {
				mockedShortener.On("Shorten", expectedURLForMock).Return(tt.id, tt.err)
			}

			resAddr := tt.resAddr
			if resAddr == "" {
				resAddr = mockResAddr
			}
			h := NewHandler(mockedShortener, resAddr)

			r := chi.NewRouter()
			r.Post("/", h.shortenHandler)

			body := strings.NewReader(tt.originalURL)
			req := httptest.NewRequest(http.MethodPost, "/", body)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			res := w.Result()

			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)

			assert.Equal(t, tt.want.response, string(resBody))
			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestAPIShortenHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name        string
		originalURL string
		id          string
		resAddr     string
		err         error
		want        want
	}{
		{
			name:        "Full https URL",
			originalURL: mockOriginalURL,
			id:          mockID,
			want: want{
				code:        http.StatusCreated,
				response:    `{"result":"http://localhost:8080/` + mockID + `"}`,
				contentType: "application/json",
			},
		},
		{
			name:        "Full http URL",
			originalURL: "http://httpbin.org",
			id:          mockID,
			want: want{
				code:        http.StatusCreated,
				response:    `{"result":"http://localhost:8080/` + mockID + `"}`,
				contentType: "application/json",
			},
		},
		{
			name:        "Short URL",
			originalURL: "example.com",
			id:          mockID,
			want: want{
				code:        http.StatusCreated,
				response:    `{"result":"http://localhost:8080/` + mockID + `"}`,
				contentType: "application/json",
			},
		},
		{
			name:        "URL with queries",
			originalURL: mockOriginalURL + "/example?test=go&test=lang",
			id:          mockID,
			want: want{
				code:        http.StatusCreated,
				response:    `{"result":"http://localhost:8080/` + mockID + `"}`,
				contentType: "application/json",
			},
		},
		{
			name:        "Empty URL",
			originalURL: "",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Empty URL\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Generation has failed",
			originalURL: mockOriginalURL,
			err:         io.ErrShortBuffer,
			want: want{
				code:        http.StatusBadRequest,
				response:    "short buffer\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Invalid URL with spaces",
			originalURL: "ex ample.com",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Invalid URL format\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Incomplete protocol",
			originalURL: "http:/invalid",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Invalid URL format\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "URL only with protocol",
			originalURL: "http://",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Invalid URL format\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Different resAddr",
			originalURL: mockOriginalURL,
			id:          mockID,
			resAddr:     "http://different-host:9090",
			want: want{
				code:        http.StatusCreated,
				response:    `{"result":"http://different-host:9090/` + mockID + `"}`,
				contentType: "application/json",
			},
		},
		{
			name:        "Invalid Content-Type",
			originalURL: mockOriginalURL,
			want: want{
				code:        http.StatusBadRequest,
				response:    "Invalid Content-Type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Invalid JSON body",
			originalURL: "{invalid json",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Invalid request body\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var expectedURLForMock string
			if strings.HasPrefix(tt.originalURL, "http://") || strings.HasPrefix(tt.originalURL, "https://") {
				expectedURLForMock = tt.originalURL
			} else if tt.originalURL != "" && tt.originalURL != "{invalid json" {
				expectedURLForMock = "http://" + tt.originalURL
			} else {
				expectedURLForMock = tt.originalURL
			}

			mockedShortener := new(mockShortener)
			if expectedURLForMock != "" && tt.name != "Invalid Content-Type" && tt.name != "Invalid JSON body" {
				mockedShortener.On("Shorten", expectedURLForMock).Return(tt.id, tt.err)
			}

			resAddr := tt.resAddr
			if resAddr == "" {
				resAddr = mockResAddr
			}
			h := NewHandler(mockedShortener, resAddr)

			r := chi.NewRouter()
			r.Post("/api/shorten", h.apiShortenHandler)

			var body *strings.Reader
			switch tt.name {
			case "Invalid JSON body":
				body = strings.NewReader(tt.originalURL)
			default:
				body = strings.NewReader(`{"url":"` + tt.originalURL + `"}`)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", body)

			switch tt.name {
			case "Invalid Content-Type":
				req.Header.Set("Content-Type", "text/plain")
			default:
				req.Header.Set("Content-Type", "application/json")
			}

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			res := w.Result()

			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)

			assert.Equal(t, tt.want.response, string(resBody))
			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
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
			name:        "Full https URL",
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
			name:        "Full http URL",
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
			name:        "URL with queries",
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockedShortener := new(mockShortener)
			mockedShortener.On("Expand", tt.id).Return(tt.originalURL, tt.ok)
			h := NewHandler(mockedShortener, mockResAddr)

			r := chi.NewRouter()
			r.Get("/{id}", h.expandHandler)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.id, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

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
				body: "Method not allowed\n",
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
				body: "Method not allowed\n",
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
		{
			name:   "POST /api/shorten",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   `{"url":"` + mockOriginalURL + `"}`,
			want: want{
				code: http.StatusCreated,
				header: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		{
			name:   "POST /api/shorten empty URL",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   `{"url":""}`,
			want: want{
				code: http.StatusBadRequest,
				body: "Empty URL\n",
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
		{
			name:   "POST /api/shorten invalid JSON",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   "{invalid",
			want: want{
				code: http.StatusBadRequest,
				body: "Invalid request body\n",
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
		{
			name:   "POST /api/shorten wrong Content-Type",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   `{"url":"` + mockOriginalURL + `"}`,
			want: want{
				code: http.StatusBadRequest,
				body: "Invalid Content-Type\n",
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
	}

	repo := repository.NewMemoryRepository()
	svc := service.NewShortenerService(repo)
	h := NewHandler(svc, mockResAddr)

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	ts := httptest.NewServer(r)
	defer ts.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.method == http.MethodGet && tt.path == "/"+mockID {
				repo.Save(mockID, mockOriginalURL)
			}

			targetURL := ts.URL + tt.path

			req, err := http.NewRequest(tt.method, targetURL, strings.NewReader(tt.body))
			require.NoError(t, err)

			if tt.path == "/api/shorten" && tt.method == http.MethodPost && tt.name != "POST /api/shorten wrong Content-Type" {
				req.Header.Set("Content-Type", "application/json")
			}

			client := &http.Client{
				CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
			res, err := client.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

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
