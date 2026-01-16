package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"em_golang_rest_service_example/internal/http-server/handlers/mocks"
	"em_golang_rest_service_example/internal/model"
	"em_golang_rest_service_example/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type readTCase struct {
	name        string
	serviceName string
	price       int
	userId      string
	startDate   string
	endDate     string
	respError   string
	mockError   error
}

func TestCreateHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// 1.Common cases
	cases := []readTCase{
		{
			name:        "Success",
			serviceName: "Yandex",
			price:       400,
			userId:      uuid.NewString(),
			startDate:   "01-2026",
			endDate:     "02-2026",
			respError:   "",
			mockError:   nil,
		},
		{
			name:        "Validation error on emty service name",
			serviceName: "",
			respError:   "empty service name",
		},
		{
			name:        "Validation error on invalid price",
			serviceName: "Netflix",
			price:       -500,
			respError:   "request price is invalid",
		},
		{
			name:        "Validation error on emty user id",
			serviceName: "Any",
			userId:      "",
			respError:   "empty user id",
		},
		{
			name:        "Validation error on invalid user id",
			serviceName: "Any",
			userId:      "Trash",
			respError:   "request user id is invalid",
		},
		{
			name:        "Validation error on emty start date",
			serviceName: "Any",
			userId:      uuid.NewString(),
			startDate:   "",
			respError:   "empty start date",
		},
		{
			name:        "Validation error on invalid start date",
			serviceName: "Any",
			userId:      uuid.NewString(),
			startDate:   "any invalid value",
			respError:   "request start date is invalid",
		},
		{
			name:        "Validation error on invalid end date",
			serviceName: "Any",
			userId:      uuid.NewString(),
			startDate:   "01-2026",
			endDate:     "trash",
			respError:   "request end date is invalid",
		},
		{
			name:        "Validation error on start date greater than end date",
			serviceName: "Any",
			userId:      uuid.NewString(),
			startDate:   "01-2026",
			endDate:     "12-2025",
			respError:   "request start date greater than end date",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			creatorMock := mocks.NewCreator(t)

			if tc.respError == "" || tc.mockError != nil {
				spec := getSpecFromreadTCase(t, &tc)
				creatorMock.On("CreateSubscription", spec).Return(int64(1), tc.mockError)
			}

			reqBody := readTCaseToStr(&tc)
			createRespCheck(t, logger, creatorMock, &reqBody, &tc.respError)
		})
	}

	// 2.Case with empty request body
	t.Run("empty request body", func(t *testing.T) {
		reqInput := ""
		crMock := mocks.NewCreator(t)
		expectedErr := "empty request"
		createRespCheck(t, logger, crMock, &reqInput, &expectedErr)
	})

	// 3.Case when cannot decode request body (price is string, but no integer)
	t.Run("invalid request body", func(t *testing.T) {
		crMock := mocks.NewCreator(t)

		invalidInput := fmt.Sprintf(
			`{"service_name": "%s", "price": "%d", "user_id": "%s", "start_date": "%s", "end_date": "%s"}`,
			"Google", 900, uuid.NewString(), "07-2027", "",
		)

		expectedErr := "failed to decode request"

		createRespCheck(t, logger, crMock, &invalidInput, &expectedErr)
	})

	// 4.Case when got error from mock
	t.Run("already exist subscription", func(t *testing.T) {
		crMock := mocks.NewCreator(t)
		testData := readTCase{
			serviceName: "Google", price: 900, userId: uuid.NewString(), startDate: "07-2027", endDate: "08-2027",
		}
		spec := getSpecFromreadTCase(t, &testData)
		crMock.On("CreateSubscription", spec).Return(int64(0), storage.ErrSubscriptionExists)

		testInput := readTCaseToStr(&testData)

		expectedErr := "subscription already exists"

		createRespCheck(t, logger, crMock, &testInput, &expectedErr)
	})
}

// Helper for check
func createRespCheck(t *testing.T, l *slog.Logger, c Creator, input *string, expectedRespErr *string) {
	t.Helper()

	handler := NewCreateHandler(l, c)

	req, err := http.NewRequest(http.MethodPost, "/subscription", bytes.NewReader([]byte(*input)))
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)

	body := rr.Body.String()

	var resp CreateResponse

	assert.Nil(t, json.Unmarshal([]byte(body), &resp))
	assert.Equal(t, *expectedRespErr, resp.Error)
}

// Helper getter subscription description from test case
func getSpecFromreadTCase(t *testing.T, tc *readTCase) model.SubscriptionSpec {
	t.Helper()

	start, err := model.DateFromString(tc.startDate)
	assert.NoError(t, err)

	end, err := model.DateFromString(tc.endDate)
	assert.NoError(t, err)

	uid, err := uuid.Parse(tc.userId)
	assert.NoError(t, err)

	spec := model.SubscriptionSpec{
		ServiceName: tc.serviceName,
		Price:       tc.price,
		UserID:      uid,
		StartDate:   start,
		EndDate:     end,
	}

	return spec
}

// Transform test case data to string
func readTCaseToStr(tc *readTCase) string {
	input := fmt.Sprintf(
		`{"service_name": "%s", "price": %d, "user_id": "%s", "start_date": "%s", "end_date": "%s"}`,
		tc.serviceName, tc.price, tc.userId, tc.startDate, tc.endDate,
	)
	return input
}
