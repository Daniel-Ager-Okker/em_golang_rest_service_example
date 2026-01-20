package handlers

import (
	"bytes"
	"em_golang_rest_service_example/internal/http-server/handlers/mocks"
	"em_golang_rest_service_example/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	sub1 = model.Subscription{
		ID: int64(1),
		SubscriptionSpec: model.SubscriptionSpec{
			ServiceName: "Yandex",
			Price:       400,
			UserID:      uuid.New(),
			StartDate:   model.Date{Month: 1, Year: 2026},
			EndDate:     model.Date{Month: 2, Year: 2026},
		},
	}

	sub2 = model.Subscription{
		ID: int64(2),
		SubscriptionSpec: model.SubscriptionSpec{
			ServiceName: "Wink",
			Price:       300,
			UserID:      uuid.New(),
			StartDate:   model.Date{Month: 2, Year: 2026},
			EndDate:     model.Date{Month: 3, Year: 2026},
		},
	}

	sub3 = model.Subscription{
		ID: int64(3),
		SubscriptionSpec: model.SubscriptionSpec{
			ServiceName: "Google",
			Price:       800,
			UserID:      uuid.New(),
			StartDate:   model.Date{Month: 3, Year: 2026},
			EndDate:     model.Date{Month: 4, Year: 2026},
		},
	}

	sub4 = model.Subscription{
		ID: int64(4),
		SubscriptionSpec: model.SubscriptionSpec{
			ServiceName: "Netflix",
			Price:       900,
			UserID:      uuid.New(),
			StartDate:   model.Date{Month: 5, Year: 2026},
			EndDate:     model.Date{Month: 6, Year: 2026},
		},
	}

	sub5 = model.Subscription{
		ID: int64(5),
		SubscriptionSpec: model.SubscriptionSpec{
			ServiceName: "VKMusic",
			Price:       150,
			UserID:      uuid.New(),
			StartDate:   model.Date{Month: 6, Year: 2026},
			EndDate:     model.Date{Month: 7, Year: 2026},
		},
	}

	allSubscriptions = []model.Subscription{sub1, sub2, sub3, sub4, sub5}
)

func TestTotalCostHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cases := []struct {
		name         string
		reqBody      string
		expectedCost int
		respCode     int
		respError    string
		mockError    error
	}{
		{
			name:         "Success no opt params",
			reqBody:      `{"start_date": "01-2025", "end_date": "01-2027"}`,
			expectedCost: 2550,
			respCode:     http.StatusOK,
		},
		{
			name:         "Success with user id opt param",
			reqBody:      fmt.Sprintf(`{"start_date": "01-2025", "end_date": "01-2027", "user_id":"%s"}`, sub4.UserID.String()),
			expectedCost: sub4.Price,
			respCode:     http.StatusOK,
		},
		{
			name:         "Success with service name opt param",
			reqBody:      fmt.Sprintf(`{"start_date": "01-2025", "end_date": "01-2027", "service_name": "%s"}`, sub2.ServiceName),
			expectedCost: sub2.Price,
			respCode:     http.StatusOK,
		},
		{
			name:      "Empty request",
			reqBody:   "",
			respCode:  http.StatusBadRequest,
			respError: "empty request",
		},
		{
			name:      "Cannot parse request",
			reqBody:   `{"start_date": 500, "end_date": "01-2027"}`,
			respCode:  http.StatusBadRequest,
			respError: "failed to decode request",
		},
		{
			name:      "Error validation on start date (empty)",
			reqBody:   `{"start_date": "", "end_date": "01-2027"}`,
			respCode:  http.StatusBadRequest,
			respError: "empty start date",
		},
		{
			name:      "Error validation on start date (invalid)",
			reqBody:   `{"start_date": "trash", "end_date": "01-2027"}`,
			respCode:  http.StatusBadRequest,
			respError: "request start date is invalid",
		},
		{
			name:      "Error validation on end date (empty)",
			reqBody:   `{"start_date": "01-2026", "end_date": ""}`,
			respCode:  http.StatusBadRequest,
			respError: "empty end date",
		},
		{
			name:      "Error validation on end date (invalid)",
			reqBody:   `{"start_date": "01-2026", "end_date": "trash"}`,
			respCode:  http.StatusBadRequest,
			respError: "request end date is invalid",
		},
		{
			name:      "Error validation on end date less than start",
			reqBody:   `{"start_date": "01-2027", "end_date": "01-2026"}`,
			respCode:  http.StatusBadRequest,
			respError: "request start date greater than end date",
		},
		{
			name:      "Invalid user id",
			reqBody:   `{"start_date": "01-2026", "end_date": "01-2027", "user_id":"trash"}`,
			respCode:  http.StatusBadRequest,
			respError: "user id filter is invalid",
		},
		{
			name:      "Cannot get subscriptions",
			reqBody:   `{"start_date": "01-2026", "end_date": "01-2027"}`,
			respCode:  http.StatusInternalServerError,
			respError: "failed to get subscription",
			mockError: errors.New("some error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listMock := mocks.NewListReader(t)
			if tc.mockError != nil || tc.respError == "" {
				var limit, offset *int
				listMock.On("GetSubscriptions", limit, offset).Return(allSubscriptions, tc.mockError)
			}

			router := chi.NewRouter()
			router.Get("/subscriptions/total-cost", NewTotalCostHandler(logger, listMock))

			req, err := http.NewRequest(
				http.MethodGet,
				"/subscriptions/total-cost",
				bytes.NewReader([]byte(tc.reqBody)),
			)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.respCode, rr.Code)

			body := rr.Body.String()

			var resp TotalCostResponse

			assert.Nil(t, json.Unmarshal([]byte(body), &resp))
			assert.Equal(t, tc.respError, resp.Error)
			if tc.respError == "" {
				assert.Equal(t, tc.expectedCost, resp.TotalCost)
			}
		})
	}
}
