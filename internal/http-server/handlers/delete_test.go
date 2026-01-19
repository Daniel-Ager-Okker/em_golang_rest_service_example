package handlers

import (
	"bytes"
	"em_golang_rest_service_example/internal/http-server/handlers/mocks"
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

func TestDeleteHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cases := []struct {
		name      string
		id        string
		respCode  int
		respError string
		mockError error
	}{
		// {
		// 	name:     "Success",
		// 	id:       "1",
		// 	respCode: http.StatusOK,
		// },
		{
			name:      "Invalid id",
			id:        "trash",
			respCode:  http.StatusBadRequest,
			respError: "invalid subscription id format",
		},
		{
			name:      "Not found subscription",
			id:        "532",
			respCode:  http.StatusNotFound,
			respError: "subscription not found",
			mockError: storage.ErrSubscribtionNotFound,
		},
		{
			name:      "Any other reader error case",
			id:        "1",
			respCode:  http.StatusInternalServerError,
			respError: "failed to delete subscription",
			mockError: errors.New("any error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deleterMock := mocks.NewDeleter(t)

			id, err := strconv.Atoi(tc.id)
			if err == nil {
				deleterMock.On("DeleteSubscription", int64(id)).Return(tc.mockError)
			}

			deleteRespCheck(t, logger, deleterMock, tc.id, tc.respCode, &tc.respError)
		})
	}
}

// Helper for check
func deleteRespCheck(t *testing.T, l *slog.Logger, d Deleter, id string, expCode int, expRespErr *string) {
	t.Helper()

	router := chi.NewRouter()
	router.Delete("/subscription/{id}", NewDeleteHandler(l, d))

	req, err := http.NewRequest(
		http.MethodDelete,
		"/subscription/"+id,
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
