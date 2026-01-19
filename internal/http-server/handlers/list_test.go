package handlers

import (
	"bytes"
	"em_golang_rest_service_example/internal/http-server/handlers/mocks"
	"em_golang_rest_service_example/internal/model"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestListHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cases := []struct {
		name      string
		respCode  int
		respError string
		mockError error
	}{
		{
			name:     "Success",
			respCode: http.StatusOK,
		},
		{
			name:      "Any other reader error case",
			respCode:  http.StatusInternalServerError,
			respError: "failed to get subscription",
			mockError: errors.New("any error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listMock := mocks.NewListReader(t)
			listMock.On("GetSubscriptions").Return([]model.Subscription{}, tc.mockError)

			listRespCheck(t, logger, listMock, tc.respCode, &tc.respError)
		})
	}
}

// Helper for check
func listRespCheck(t *testing.T, l *slog.Logger, r ListReader, expCode int, expRespErr *string) {
	t.Helper()

	router := chi.NewRouter()
	router.Get("/subscriptions", NewListHandler(l, r))

	req, err := http.NewRequest(
		http.MethodGet,
		"/subscriptions",
		bytes.NewReader([]byte{}),
	)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, expCode, rr.Code)

	body := rr.Body.String()

	var resp ReadResponse

	assert.Nil(t, json.Unmarshal([]byte(body), &resp))
	assert.Equal(t, *expRespErr, resp.Error)
}
