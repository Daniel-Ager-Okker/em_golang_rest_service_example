package handlers

import (
	"bytes"
	"em_golang_rest_service_example/internal/http-server/handlers/mocks"
	"em_golang_rest_service_example/internal/model"
	"em_golang_rest_service_example/internal/storage"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestReadHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cases := []struct {
		name      string
		id        string
		respError string
		mockError error
	}{
		{
			name: "Success",
			id:   "1",
		},
		{
			name:      "Invalid id",
			id:        "trash",
			respError: "invalid subscription id format",
		},
		{
			name:      "Not found subscription",
			id:        "532",
			respError: "subscription not found",
			mockError: storage.ErrSubscribtionNotFound,
		},
		{
			name:      "Any other reader error case",
			id:        "1",
			respError: "failed to get subscription",
			mockError: errors.New("any error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			readerMock := mocks.NewReader(t)

			id, err := strconv.Atoi(tc.id)
			if err == nil {
				readerMock.On("GetSubscription", int64(id)).Return(model.Subscription{}, tc.mockError)
			}

			readRespCheck(t, logger, readerMock, tc.id, &tc.respError)
		})
	}
}

// Helper for check
func readRespCheck(t *testing.T, l *slog.Logger, r Reader, id string, expRespErr *string) {
	t.Helper()

	router := chi.NewRouter()
	router.Get("/subscription/{id}", NewReadHandler(l, r))

	req, err := http.NewRequest(
		http.MethodGet,
		"/subscription/"+id,
		bytes.NewReader([]byte{}),
	)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)

	body := rr.Body.String()

	var resp ReadResponse

	assert.Nil(t, json.Unmarshal([]byte(body), &resp))
	assert.Equal(t, *expRespErr, resp.Error)
}
