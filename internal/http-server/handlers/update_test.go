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
	name           string
	id             string
	newServiceName string
	newPrice       int
	newStartDate   string
	newEndDate     string
	respCode       int
	respError      string
	mockError      error
}

func TestUpdateHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// 1.Common cases
	commonCases := []updateTCase{
		{
			name:           "Success",
			id:             "1",
			newServiceName: "Яндекс",
			newPrice:       350,
			newStartDate:   "03-2026",
			newEndDate:     "04-2026",
			respCode:       http.StatusOK,
		},
		{
			name:      "Invalid id",
			id:        "trash",
			respCode:  http.StatusBadRequest,
			respError: "invalid subscription id format",
		},
		{
			name:      "Validation error on service name",
			id:        "2",
			respCode:  http.StatusBadRequest,
			respError: "request service name is empty",
		},
		{
			name:           "Validation error on price",
			id:             "2",
			newServiceName: "Гугл",
			newPrice:       -5,
			respCode:       http.StatusBadRequest,
			respError:      "request price is invalid",
		},
		{
			name:           "Validation error on start date (empty)",
			id:             "2",
			newServiceName: "Гугл",
			newPrice:       5,
			respCode:       http.StatusBadRequest,
			respError:      "request start date is empty",
		},
		{
			name:           "Validation error on start date (invalid)",
			id:             "2",
			newServiceName: "Гугл",
			newPrice:       5,
			newStartDate:   "trash trashovich",
			respCode:       http.StatusBadRequest,
			respError:      "request start date is invalid",
		},
		{
			name:           "Validation error on new end date",
			id:             "2",
			newServiceName: "Амедиатека",
			newPrice:       155,
			newStartDate:   "01-2027",
			newEndDate:     "trash-garbage",
			respCode:       http.StatusBadRequest,
			respError:      "request end date is invalid",
		},
		{
			name:           "Not found subscription",
			id:             "3",
			newServiceName: "Подписки.net",
			newPrice:       155,
			newStartDate:   "01-2025",
			newEndDate:     "05-2025",
			respCode:       http.StatusNotFound,
			respError:      "subscription not found",
			mockError:      storage.ErrSubscribtionNotFound,
		},
		{
			name:           "Any other storage error",
			id:             "3",
			newServiceName: "Ошибочная подписка",
			newPrice:       155,
			newStartDate:   "04-2025",
			newEndDate:     "05-2025",
			respCode:       http.StatusInternalServerError,
			respError:      "failed to get subscription",
			mockError:      errors.New("some error"),
		},
	}

	for _, tc := range commonCases {
		t.Run(tc.name, func(t *testing.T) {
			updaterMock := mocks.NewUpdater(t)

			if tc.respError == "" || tc.mockError != nil {
				id, err := strconv.Atoi(tc.id)
				if err == nil {
					newStartDate, err := model.DateFromString(tc.newStartDate)
					assert.NoError(t, err)

					newEndDate, err := model.DateFromString(tc.newEndDate)
					assert.NoError(t, err)

					updaterMock.On("UpdateSubscription", int64(id), tc.newServiceName, tc.newPrice, newStartDate, newEndDate).Return(tc.mockError)
				}

			}

			reqBody := updateTCaseToStr(&tc)

			updateRespCheck(t, logger, updaterMock, &tc, &reqBody, tc.respCode, &tc.respError)
		})
	}

	// 2.Request parsing cases
	reqCases := []struct {
		name      string
		respCode  int
		respError string
		input     string
	}{
		{
			name:      "Empty request",
			respCode:  http.StatusBadRequest,
			respError: "empty request",
			input:     "",
		},
		{
			name:      "Invalid request body",
			respCode:  http.StatusBadRequest,
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

			tc := updateTCase{id: "1", respCode: reqTc.respCode, respError: reqTc.respError}

			updateRespCheck(t, logger, updaterMock, &tc, &reqTc.input, tc.respCode, &tc.respError)
		})
	}
}

// Helper for check
func updateRespCheck(t *testing.T, l *slog.Logger, u Updater, tc *updateTCase, in *string, expRespCode int, expectedRespErr *string) {
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

	assert.Equal(t, expRespCode, rr.Code)

	body := rr.Body.String()

	var resp Response

	assert.Nil(t, json.Unmarshal([]byte(body), &resp))
	assert.Equal(t, *expectedRespErr, resp.Error)
}

// Transform test case data to string
func updateTCaseToStr(tc *updateTCase) string {
	input := fmt.Sprintf(
		`{"service_name": "%s", "price": %d, "start_date": "%s", "end_date": "%s"}`, tc.newServiceName,
		tc.newPrice, tc.newStartDate, tc.newEndDate,
	)
	return input
}
