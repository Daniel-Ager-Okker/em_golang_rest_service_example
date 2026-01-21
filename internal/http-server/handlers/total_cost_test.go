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
	"net/url"
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
			EndDate:     model.Date{Month: 8, Year: 2026},
		},
	}
)

func TestTotalCostHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cases := []struct {
		name         string
		url          string
		expectedCost int
		respCode     int
		respError    string
		mockNeedCall bool
		mockRet      []model.Subscription
		mockError    error
	}{
		{
			name:         "Success no optional params",
			url:          "/subscriptions/total-cost?start_date=12-2025&end_date=08-2026",
			expectedCost: 2700,
			respCode:     http.StatusOK,
			mockNeedCall: true,
			mockRet:      []model.Subscription{sub1, sub2, sub3, sub4, sub5},
		},
		{
			name:         "Success with user id opt param",
			url:          "/subscriptions/total-cost?start_date=12-2025&end_date=04-2026",
			expectedCost: sub4.Price,
			respCode:     http.StatusOK,
			mockNeedCall: true,
			mockRet:      []model.Subscription{sub4},
		},
		{
			name:         "Success with service name opt param",
			url:          fmt.Sprintf("/subscriptions/total-cost?start_date=12-2025&end_date=08-2026&service_name=%s", sub2.ServiceName),
			expectedCost: sub2.Price,
			respCode:     http.StatusOK,
			mockNeedCall: true,
			mockRet:      []model.Subscription{sub2},
		},
		{
			name:         "Success with user id opt param",
			url:          fmt.Sprintf("/subscriptions/total-cost?start_date=12-2025&end_date=08-2026&user_id=%s", sub4.UserID.String()),
			expectedCost: sub4.Price,
			respCode:     http.StatusOK,
			mockNeedCall: true,
			mockRet:      []model.Subscription{sub4},
		},
		{
			name:      "Empty start date",
			url:       "/subscriptions/total-cost?start_date=&end_date=04-2026",
			respCode:  http.StatusBadRequest,
			respError: "empty start date",
		},
		{
			name:      "Error validation on start date (invalid)",
			url:       "/subscriptions/total-cost?start_date=trash&end_date=04-2026",
			respCode:  http.StatusBadRequest,
			respError: "request start date is invalid",
		},
		{
			name:      "Empty end date",
			url:       "/subscriptions/total-cost?start_date=07-2027&end_date=",
			respCode:  http.StatusBadRequest,
			respError: "empty end date",
		},
		{
			name:      "Error validation on end date (invalid)",
			url:       "/subscriptions/total-cost?start_date=07-2027&end_date=trash",
			respCode:  http.StatusBadRequest,
			respError: "request end date is invalid",
		},
		{
			name:      "Error validation on end date less than start",
			url:       "/subscriptions/total-cost?start_date=07-2027&end_date=01-2027",
			respCode:  http.StatusBadRequest,
			respError: "request start date greater than end date",
		},
		{
			name:      "Invalid user id",
			url:       "/subscriptions/total-cost?start_date=07-2027&end_date=09-2027&user_id=trash",
			respCode:  http.StatusBadRequest,
			respError: "user id filter is invalid",
		},
		{
			name:         "Cannot get subscriptions",
			url:          "/subscriptions/total-cost?start_date=07-2027&end_date=09-2027",
			respCode:     http.StatusInternalServerError,
			respError:    "failed to get subscription",
			mockNeedCall: true,
			mockError:    errors.New("some error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filterMock := mocks.NewFilteredDataReader(t)
			if tc.mockNeedCall {
				start, end, uid, sName := getParamsFromTotalCostReqUrl(t, &tc.url)

				var sNamePtr *string
				if sName != "" {
					sNamePtr = &sName
				}

				filterMock.On("FilterSubscriptions", start, end, uid, sNamePtr).Return(tc.mockRet, tc.mockError)
			}

			router := chi.NewRouter()
			router.Get("/subscriptions/total-cost", NewTotalCostHandler(logger, filterMock))

			req, err := http.NewRequest(
				http.MethodGet,
				tc.url,
				bytes.NewReader([]byte{}),
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

// Helper for get total cost calculating params from URL
func getParamsFromTotalCostReqUrl(t *testing.T, rawUrl *string) (model.Date, model.Date, uuid.UUID, string) {
	t.Helper()

	parsed, err := url.Parse(*rawUrl)
	assert.NoError(t, err)

	query := parsed.Query()

	dateStart := query["start_date"][0]
	start, err := model.DateFromString(dateStart)
	assert.NoError(t, err)

	dateEnd := query["end_date"][0]
	end, err := model.DateFromString(dateEnd)
	assert.NoError(t, err)

	uid := uuid.Nil
	userId := query["user_id"]
	if len(userId) > 0 {
		uid, err = uuid.Parse(userId[0])
		assert.NoError(t, err)
	}

	sName := ""
	serviceName := query["service_name"]
	if len(serviceName) > 0 {
		sName = serviceName[0]
	}

	return start, end, uid, sName
}
