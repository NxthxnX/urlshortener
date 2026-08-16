package handler

import (
	"fmt"
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

func TestShortenHandler_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockShortener(ctrl)

	m.EXPECT().Shorten(expectedURLForMock(mockOriginalURL)).
		Return(mockID, fmt.Errorf("%w: original URL already exists", repository.ErrOriginalURLConflict))

	h := NewHandler(m, mockResAddr)

	r := chi.NewRouter()
	r.Post("/", h.shortenHandler)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(mockOriginalURL))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusConflict, res.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
	assert.Equal(t, plainShortURL(mockResAddr, mockID), string(resBody))
}

func TestAPIShortenHandler_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockShortener(ctrl)

	m.EXPECT().Shorten(expectedURLForMock(mockOriginalURL)).
		Return(mockID, fmt.Errorf("%w: original URL already exists", repository.ErrOriginalURLConflict))

	h := NewHandler(m, mockResAddr)

	r := chi.NewRouter()
	r.Post("/api/shorten", h.apiShortenHandler)

	body := `{"url":"` + mockOriginalURL + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusConflict, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "application/json")
	assert.Equal(t, apiShortURL(mockResAddr, mockID), string(resBody))
}

func TestAPIShortenBatchHandler_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockShortener(ctrl)

	m.EXPECT().
		ShortenBatch([]string{"https://example.com", "http://yandex.ru"}).
		Return([]string{"abcd1234", "existingID"},
			fmt.Errorf("%w: some original URLs already exist", repository.ErrOriginalURLConflict))

	h := NewHandler(m, mockResAddr)

	r := chi.NewRouter()
	r.Post("/api/shorten/batch", h.apiShortenBatchHandler)

	body := `[{"correlation_id":"1","original_url":"https://example.com"},{"correlation_id":"2","original_url":"yandex.ru"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusConflict, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "application/json")
	assert.Equal(t,
		`[{"correlation_id":"1","short_url":"`+mockResAddr+`/abcd1234"},{"correlation_id":"2","short_url":"`+mockResAddr+`/existingID"}]`,
		string(resBody))
}

func TestShortenHandler_ShortURLConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockShortener(ctrl)

	m.EXPECT().Shorten(expectedURLForMock(mockOriginalURL)).
		Return("", fmt.Errorf("%w: short URL already exists", repository.ErrShortURLConflict))

	h := NewHandler(m, mockResAddr)

	r := chi.NewRouter()
	r.Post("/", h.shortenHandler)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(mockOriginalURL))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}

func TestShortenHandler_Conflict_Integration(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortenerService(repo)
	h := NewHandler(svc, mockResAddr)

	r := chi.NewRouter()
	r.Post("/", h.shortenHandler)

	req1 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(mockOriginalURL))
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	res1 := w1.Result()
	defer res1.Body.Close()
	assert.Equal(t, http.StatusCreated, res1.StatusCode)
	firstBody, err := io.ReadAll(res1.Body)
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(mockOriginalURL))
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	res2 := w2.Result()
	defer res2.Body.Close()
	assert.Equal(t, http.StatusConflict, res2.StatusCode)
	secondBody, err := io.ReadAll(res2.Body)
	require.NoError(t, err)
	assert.Equal(t, string(firstBody), string(secondBody))
}

func TestAPIShortenHandler_Conflict_Integration(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortenerService(repo)
	h := NewHandler(svc, mockResAddr)

	r := chi.NewRouter()
	r.Post("/api/shorten", h.apiShortenHandler)

	body := `{"url":"` + mockOriginalURL + `"}`

	req1 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	res1 := w1.Result()
	defer res1.Body.Close()
	assert.Equal(t, http.StatusCreated, res1.StatusCode)
	firstBody, err := io.ReadAll(res1.Body)
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	res2 := w2.Result()
	defer res2.Body.Close()
	assert.Equal(t, http.StatusConflict, res2.StatusCode)
	secondBody, err := io.ReadAll(res2.Body)
	require.NoError(t, err)
	assert.Equal(t, string(firstBody), string(secondBody))
}

func TestAPIShortenBatchHandler_Conflict_Integration(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortenerService(repo)
	h := NewHandler(svc, mockResAddr)

	r := chi.NewRouter()
	r.Post("/api/shorten/batch", h.apiShortenBatchHandler)

	_, err := repo.Save("existingID", "http://yandex.ru")
	require.NoError(t, err)

	body := `[{"correlation_id":"1","original_url":"https://example.com"},{"correlation_id":"2","original_url":"yandex.ru"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusConflict, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "application/json")

	assert.Contains(t, string(resBody), `"correlation_id":"1"`)
	assert.Contains(t, string(resBody), `"correlation_id":"2"`)
	assert.Contains(t, string(resBody), "existingID")
}
