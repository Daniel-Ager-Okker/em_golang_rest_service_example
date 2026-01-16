package handlers

import (
	"bytes"
	"em_golang_rest_service_example/internal/http-server/handlers/mocks"
	"em_golang_rest_service_example/internal/model"
	"em_golang_rest_service_example/internal/storage"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type updateTCase struct {
	name       string
	id         string
	newPrice   int
	newEndDate string
	respError  string
	mockError  error
}

func TestUpdateHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// 1.Common cases
	commonCases := []updateTCase{
		{
			name:       "Success",
			id:         "1",
			newPrice:   350,
			newEndDate: "04-2026",
		},
		{
			name:      "Invalid id",
			id:        "trash",
			respError: "invalid subscription id format",
		},
		{
			name:      "Validation error on price",
			id:        "2",
			newPrice:  -5,
			respError: "request price is invalid",
		},
		{
			name:       "Validation error on new end date",
			id:         "2",
			newPrice:   155,
			newEndDate: "trash-garbage",
			respError:  "request end date is invalid",
		},
		{
			name:       "Not found subscription",
			id:         "3",
			newPrice:   155,
			newEndDate: "05-2025",
			respError:  "subscription not found",
			mockError:  storage.ErrSubscribtionNotFound,
		},
		{
			name:       "Any other storage error",
			id:         "3",
			newPrice:   155,
			newEndDate: "05-2025",
			respError:  "failed to get subscription",
			mockError:  errors.New("some error"),
		},
	}

	for _, tc := range commonCases {
		t.Run(tc.name, func(t *testing.T) {
			updaterMock := mocks.NewUpdater(t)

			if tc.respError == "" || tc.mockError != nil {
				id, err := strconv.Atoi(tc.id)
				if err == nil {
					newEndDate, err := model.DateFromString(tc.newEndDate)
					assert.NoError(t, err)
					updaterMock.On("UpdateSubscription", int64(id), tc.newPrice, newEndDate).Return(tc.mockError)
				}

			}

			reqBody := updateTCaseToStr(&tc)

			updateRespCheck(t, logger, updaterMock, &tc, &reqBody, &tc.respError)
		})
	}

	// 2.Request parsing cases
	reqCases := []struct {
		name      string
		respError string
		input     string
	}{
		{
			name:      "Empty request",
			respError: "empty request",
			input:     "",
		},
		{
			name:      "Invalid request body",
			respError: "failed to decode request",
			input: fmt.Sprintf(
				`{"price": "%d", "end_date": "%s"}`,
				900, "07-2027",
			),
		},
	}

	for _, reqTc := range reqCases {
		t.Run(reqTc.name, func(t *testing.T) {
			updaterMock := mocks.NewUpdater(t)

			tc := updateTCase{id: "1", respError: reqTc.respError}

			updateRespCheck(t, logger, updaterMock, &tc, &reqTc.input, &tc.respError)
		})
	}
}

// Helper for check
func updateRespCheck(t *testing.T, l *slog.Logger, u Updater, tc *updateTCase, in *string, expectedRespErr *string) {
	t.Helper()

	router := chi.NewRouter()
	router.Patch("/subscription/{id}", NewUpdateHandler(l, u))

	req, err := http.NewRequest(
		http.MethodPatch,
		"/subscription/"+tc.id,
		bytes.NewReader([]byte(*in)),
	)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()

	var resp Response

	assert.Nil(t, json.Unmarshal([]byte(body), &resp))
	assert.Equal(t, *expectedRespErr, resp.Error)
}

// Transform test case data to string
func updateTCaseToStr(tc *updateTCase) string {
	input := fmt.Sprintf(
		`{"price": %d, "end_date": "%s"}`,
		tc.newPrice, tc.newEndDate,
	)
	return input
}
