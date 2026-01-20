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
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestListHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cases := []struct {
		name         string
		limit        string
		offset       string
		respCode     int
		respError    string
		needMockCall bool
		mockError    error
	}{
		{
			name:         "Success no opt params",
			respCode:     http.StatusOK,
			needMockCall: true,
		},
		{
			name:         "Success with opt params",
			respCode:     http.StatusOK,
			limit:        "100",
			offset:       "100",
			needMockCall: true,
		},
		{
			name:         "Limit set, but offset - no",
			limit:        "100",
			respCode:     http.StatusBadRequest,
			respError:    "no offset value while limit is set",
			needMockCall: false,
		},
		{
			name:         "Offset set, but limit - no",
			offset:       "100",
			respCode:     http.StatusBadRequest,
			respError:    "no limit value while offset is set",
			needMockCall: false,
		},
		{
			name:         "Invalid limit",
			limit:        "trash",
			offset:       "100",
			respCode:     http.StatusBadRequest,
			respError:    "invalid limit format",
			needMockCall: false,
		},
		{
			name:         "Invalid offset",
			limit:        "100",
			offset:       "trash",
			respCode:     http.StatusBadRequest,
			respError:    "invalid offset format",
			needMockCall: false,
		},
		{
			name:         "Limit less than zero",
			limit:        "-5",
			offset:       "100",
			respCode:     http.StatusBadRequest,
			respError:    "invalid limit value (less than zero)",
			needMockCall: false,
		},
		{
			name:         "Offset less than zero",
			limit:        "5",
			offset:       "-100",
			respCode:     http.StatusBadRequest,
			respError:    "invalid offset value (less than zero)",
			needMockCall: false,
		},
		{
			name:         "Any other reader error case",
			respCode:     http.StatusInternalServerError,
			respError:    "failed to get subscription",
			needMockCall: true,
			mockError:    errors.New("any error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listMock := mocks.NewListReader(t)

			if tc.needMockCall {
				if tc.limit != "" && tc.offset != "" {
					limit, err := strconv.Atoi(tc.limit)
					assert.NoError(t, err)

					offset, err := strconv.Atoi(tc.offset)
					assert.NoError(t, err)

					listMock.On("GetSubscriptions", &limit, &offset).Return([]model.Subscription{}, tc.mockError)
				} else {
					var limit, offset *int
					listMock.On("GetSubscriptions", limit, offset).Return([]model.Subscription{}, tc.mockError)
				}
			}

			router := chi.NewRouter()
			router.Get("/subscriptions", NewListHandler(logger, listMock))

			req, err := http.NewRequest(
				http.MethodGet,
				constructURL(t, &tc.limit, &tc.offset),
				bytes.NewReader([]byte{}),
			)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.respCode, rr.Code)

			body := rr.Body.String()

			var resp ReadResponse

			assert.Nil(t, json.Unmarshal([]byte(body), &resp))
			assert.Equal(t, tc.respError, resp.Error)
		})
	}
}

// Helper function for cinstruct URL with optional parameters
func constructURL(t *testing.T, limit, offset *string) string {
	t.Helper()

	if *limit == "" && *offset == "" {
		return "/subscriptions"
	}

	if *limit != "" && *offset == "" {
		return fmt.Sprintf("/subscriptions?limit=%s", *limit)
	}

	if *limit == "" && *offset != "" {
		return fmt.Sprintf("/subscriptions?offset=%s", *offset)
	}

	return fmt.Sprintf("/subscriptions?limit=%s&offset=%s", *limit, *offset)
}
