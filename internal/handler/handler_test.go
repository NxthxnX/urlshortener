package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NxthxnX/urlshortener/internal/handler/mocks"
	"github.com/NxthxnX/urlshortener/internal/repository"
	"github.com/NxthxnX/urlshortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	mockResAddr     = "http://localhost:8080"
	mockOriginalURL = "https://www.example.com"
	mockID          = "abcd1234"
)

func expectedURLForMock(originalURL string) string {
	if strings.HasPrefix(originalURL, "http://") || strings.HasPrefix(originalURL, "https://") {
		return originalURL
	}
	if originalURL != "" {
		return "http://" + originalURL
	}
	return originalURL
}

type shortenScenario struct {
	name        string
	originalURL string
	id          string
	resAddr     string
	err         error
	wantCode    int
	wantBody    string
	contentType string
}

var sharedShortenScenarios = []shortenScenario{
	{
		name:        "Full https URL",
		originalURL: mockOriginalURL,
		id:          mockID,
		wantCode:    http.StatusCreated,
	},
	{
		name:        "Full http URL",
		originalURL: "http://httpbin.org",
		id:          mockID,
		wantCode:    http.StatusCreated,
	},
	{
		name:        "Short URL",
		originalURL: "example.com",
		id:          mockID,
		wantCode:    http.StatusCreated,
	},
	{
		name:        "URL with queries",
		originalURL: mockOriginalURL + "/example?test=go&test=lang",
		id:          mockID,
		wantCode:    http.StatusCreated,
	},
	{
		name:        "Empty URL",
		originalURL: "",
		wantCode:    http.StatusBadRequest,
		wantBody:    "Empty URL\n",
		contentType: "text/plain; charset=utf-8",
	},
	{
		name:        "Generation has failed",
		originalURL: mockOriginalURL,
		err:         io.ErrShortBuffer,
		wantCode:    http.StatusInternalServerError,
		wantBody:    "short buffer\n",
		contentType: "text/plain; charset=utf-8",
	},
	{
		name:        "Invalid URL with spaces",
		originalURL: "ex ample.com",
		wantCode:    http.StatusBadRequest,
		wantBody:    "Invalid URL format\n",
		contentType: "text/plain; charset=utf-8",
	},
	{
		name:        "Incomplete protocol",
		originalURL: "http:/invalid",
		wantCode:    http.StatusBadRequest,
		wantBody:    "Invalid URL format\n",
		contentType: "text/plain; charset=utf-8",
	},
	{
		name:        "URL only with protocol",
		originalURL: "http://",
		wantCode:    http.StatusBadRequest,
		wantBody:    "Invalid URL format\n",
		contentType: "text/plain; charset=utf-8",
	},
	{
		name:        "Different resAddr",
		originalURL: mockOriginalURL,
		id:          mockID,
		resAddr:     "http://different-host:9090",
		wantCode:    http.StatusCreated,
	},
}

func plainShortURL(resAddr, id string) string {
	return resAddr + "/" + id
}

func apiShortURL(resAddr, id string) string {
	return `{"result":"` + resAddr + `/` + id + `"}`
}

func setupShortenMock(ctrl *gomock.Controller, tt shortenScenario) *mocks.MockShortener {
	m := mocks.NewMockShortener(ctrl)

	if tt.wantCode == http.StatusCreated || tt.err != nil {
		m.EXPECT().Shorten(expectedURLForMock(tt.originalURL)).Return(tt.id, tt.err)
	}
	return m
}

func resolveResAddr(resAddr string) string {
	if resAddr == "" {
		return mockResAddr
	}
	return resAddr
}

func runPlainShortenScenario(t *testing.T, tt shortenScenario) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockedShortener := setupShortenMock(ctrl, tt)
	resAddr := resolveResAddr(tt.resAddr)
	h := NewHandler(mockedShortener, resAddr)

	r := chi.NewRouter()
	r.Post("/", h.shortenHandler)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.originalURL))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	wantBody := tt.wantBody
	if wantBody == "" {
		wantBody = plainShortURL(resAddr, tt.id)
	}
	wantContentType := tt.contentType
	if wantContentType == "" {
		wantContentType = "text/plain; charset=utf-8"
	}

	assert.Equal(t, wantBody, string(resBody))
	assert.Equal(t, tt.wantCode, res.StatusCode)
	assert.Equal(t, wantContentType, res.Header.Get("Content-Type"))
}

func runAPIShortenScenario(t *testing.T, tt shortenScenario, body string, contentType string) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockedShortener := setupShortenMock(ctrl, tt)
	resAddr := resolveResAddr(tt.resAddr)
	h := NewHandler(mockedShortener, resAddr)

	r := chi.NewRouter()
	r.Post("/api/shorten", h.apiShortenHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	wantBody := tt.wantBody
	if wantBody == "" {
		wantBody = apiShortURL(resAddr, tt.id)
	}
	wantContentType := tt.contentType
	if wantContentType == "" {
		wantContentType = "application/json"
	}

	assert.Equal(t, wantBody, string(resBody))
	assert.Equal(t, tt.wantCode, res.StatusCode)
	assert.Equal(t, wantContentType, res.Header.Get("Content-Type"))
}

func TestShortenHandler(t *testing.T) {
	for _, tt := range sharedShortenScenarios {
		t.Run(tt.name, func(t *testing.T) {
			runPlainShortenScenario(t, tt)
		})
	}
}

func TestAPIShortenHandler(t *testing.T) {
	for _, tt := range sharedShortenScenarios {
		t.Run(tt.name, func(t *testing.T) {
			runAPIShortenScenario(t, tt, `{"url":"`+tt.originalURL+`"}`, "application/json")
		})
	}

	apiOnlyScenarios := []struct {
		name        string
		body        string
		contentType string
		wantCode    int
		wantBody    string
	}{
		{
			name:        "Invalid Content-Type",
			body:        `{"url":"` + mockOriginalURL + `"}`,
			contentType: "text/plain",
			wantCode:    http.StatusUnsupportedMediaType,
			wantBody:    "Invalid Content-Type\n",
		},
		{
			name:        "Invalid JSON body",
			body:        "{invalid json",
			contentType: "application/json",
			wantCode:    http.StatusBadRequest,
			wantBody:    "Invalid request body\n",
		},
	}

	for _, tt := range apiOnlyScenarios {
		t.Run(tt.name, func(t *testing.T) {
			runAPIShortenScenario(t, shortenScenario{
				name:        tt.name,
				wantCode:    tt.wantCode,
				wantBody:    tt.wantBody,
				contentType: "text/plain; charset=utf-8",
			}, tt.body, tt.contentType)
		})
	}
}

func TestAPIShortenBatchHandler(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		setup       func(m *mocks.MockShortener)
		wantCode    int
		wantBody    string
	}{
		{
			name:        "Success",
			body:        `[{"correlation_id":"1","original_url":"https://example.com"},{"correlation_id":"2","original_url":"yandex.ru"}]`,
			contentType: "application/json",
			setup: func(m *mocks.MockShortener) {
				m.EXPECT().
					ShortenBatch([]string{"https://example.com", "http://yandex.ru"}).
					Return([]string{"abcd1234", "efgh5678"}, nil)
			},
			wantCode: http.StatusCreated,
			wantBody: `[{"correlation_id":"1","short_url":"http://localhost:8080/abcd1234"},{"correlation_id":"2","short_url":"http://localhost:8080/efgh5678"}]`,
		},
		{
			name:        "Invalid Content-Type",
			body:        `[{"correlation_id":"1","original_url":"https://example.com"}]`,
			contentType: "text/plain",
			wantCode:    http.StatusUnsupportedMediaType,
			wantBody:    "Invalid Content-Type\n",
		},
		{
			name:        "Invalid JSON body",
			body:        "{invalid json",
			contentType: "application/json",
			wantCode:    http.StatusBadRequest,
			wantBody:    "Invalid request body\n",
		},
		{
			name:        "Empty batch",
			body:        `[]`,
			contentType: "application/json",
			wantCode:    http.StatusBadRequest,
			wantBody:    "Empty batch\n",
		},
		{
			name:        "Duplicate correlation_id",
			body:        `[{"correlation_id":"1","original_url":"https://example.com"},{"correlation_id":"1","original_url":"https://other.com"}]`,
			contentType: "application/json",
			wantCode:    http.StatusBadRequest,
			wantBody:    "Duplicate correlation_id\n",
		},
		{
			name:        "Empty URL",
			body:        `[{"correlation_id":"1","original_url":""}]`,
			contentType: "application/json",
			wantCode:    http.StatusBadRequest,
			wantBody:    "Empty URL\n",
		},
		{
			name:        "Invalid URL format",
			body:        `[{"correlation_id":"1","original_url":"ex ample.com"}]`,
			contentType: "application/json",
			wantCode:    http.StatusBadRequest,
			wantBody:    "Invalid URL format\n",
		},
		{
			name:        "Shortener failure",
			body:        `[{"correlation_id":"1","original_url":"https://example.com"}]`,
			contentType: "application/json",
			setup: func(m *mocks.MockShortener) {
				m.EXPECT().
					ShortenBatch([]string{"https://example.com"}).
					Return(nil, io.ErrShortBuffer)
			},
			wantCode: http.StatusInternalServerError,
			wantBody: "short buffer\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks.NewMockShortener(ctrl)
			if tt.setup != nil {
				tt.setup(m)
			}

			h := NewHandler(m, mockResAddr)

			r := chi.NewRouter()
			r.Post("/api/shorten/batch", h.apiShortenBatchHandler)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Equal(t, tt.wantBody, string(resBody))
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
				code:        http.StatusNotFound,
				response:    "Not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockedShortener := mocks.NewMockShortener(ctrl)
			mockedShortener.EXPECT().Expand(tt.id).Return(tt.originalURL, tt.ok)
			h := NewHandler(mockedShortener, mockResAddr)

			r := chi.NewRouter()
			r.Get("/{id}", h.expandHandler)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.id, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

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
				body: `http://localhost:8080/[a-zA-Z0-9]+`,
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
				code: http.StatusMethodNotAllowed,
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
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "GET /nonexistent",
			method: http.MethodGet,
			path:   "/nonexistent",
			want: want{
				code: http.StatusNotFound,
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
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "DELETE /",
			method: http.MethodDelete,
			path:   "/",
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "POST /api/shorten",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   `{"url":"` + mockOriginalURL + `"}`,
			want: want{
				code: http.StatusCreated,
				body: `{"result":"http://localhost:8080/[a-zA-Z0-9]+"}`,
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
				code: http.StatusUnsupportedMediaType,
				body: "Invalid Content-Type\n",
				header: map[string]string{
					"Content-Type": "text/plain; charset=utf-8",
				},
			},
		},
		{
			name:   "POST /api/shorten/batch",
			method: http.MethodPost,
			path:   "/api/shorten/batch",
			body:   `[{"correlation_id":"1","original_url":"https://example.com"}]`,
			want: want{
				code: http.StatusCreated,
				body: `\[{"correlation_id":"1","short_url":"http://localhost:8080/[a-zA-Z0-9]+"}\]`,
				header: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		{
			name:   "POST /api/shorten/batch empty batch",
			method: http.MethodPost,
			path:   "/api/shorten/batch",
			body:   `[]`,
			want: want{
				code: http.StatusBadRequest,
				body: "Empty batch\n",
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
				err := repo.Save(mockID, mockOriginalURL)
				require.NoError(t, err)
			}

			targetURL := ts.URL + tt.path

			req, err := http.NewRequest(tt.method, targetURL, strings.NewReader(tt.body))
			require.NoError(t, err)

			if (tt.path == "/api/shorten" || tt.path == "/api/shorten/batch") && tt.method == http.MethodPost && tt.name != "POST /api/shorten wrong Content-Type" {
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
				assert.Regexp(t, tt.want.body, string(resBody))
			}

			for key, expectedVal := range tt.want.header {
				assert.Equal(t, expectedVal, res.Header.Get(key))
			}
		})
	}
}
