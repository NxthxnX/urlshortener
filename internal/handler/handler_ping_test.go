package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NxthxnX/urlshortener/internal/handler/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPingHandler(t *testing.T) {
	pingErr := errors.New("connection refused")

	tests := []struct {
		name     string
		pinger   func(ctrl *gomock.Controller) Pinger
		wantCode int
		wantBody string
	}{
		{
			name: "nil pinger returns 500",
			pinger: func(_ *gomock.Controller) Pinger {
				return nil
			},
			wantCode: http.StatusInternalServerError,
			wantBody: "database is not configured\n",
		},
		{
			name: "ping error returns 500",
			pinger: func(ctrl *gomock.Controller) Pinger {
				m := mocks.NewMockPinger(ctrl)
				m.EXPECT().Ping(gomock.Any()).Return(pingErr)
				return m
			},
			wantCode: http.StatusInternalServerError,
			wantBody: "connection refused\n",
		},
		{
			name: "successful ping returns 200",
			pinger: func(ctrl *gomock.Controller) Pinger {
				m := mocks.NewMockPinger(ctrl)
				m.EXPECT().Ping(gomock.Any()).Return(nil)
				return m
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			rec := httptest.NewRecorder()

			PingHandler(tt.pinger(ctrl))(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Equal(t, tt.wantBody, string(body))
		})
	}
}
