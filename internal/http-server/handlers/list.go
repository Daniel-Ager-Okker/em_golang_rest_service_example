package handlers

import (
	"em_golang_rest_service_example/internal/model"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// ListItem represents one subscription in list model
// swagger:model ListItem
// @ID ListItem
type ListItem struct {
	// Subscription id
	Id int64 `json:"id"`

	// Subscription service name
	ServiceName string `json:"service_name"`

	// Subscription monthly price
	Price int `json:"price"`

	// If of user who purchased the subscription
	UserID string `json:"user_id"`

	// Start date of subscription
	StartDate string `json:"start_date"`

	// Start date of subscription
	EndDate string `json:"end_date"`
}

// ListResponse represents subscription list model
// swagger:model ListResponse
// @ID ListResponse
type ListResponse struct {
	// Data about all subscriptions got
	Items []ListItem `json:"items"`

	Response
}

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=ListReader
type ListReader interface {
	GetSubscriptions(limit, offset *int) ([]model.Subscription, error)
}

// NewListHandler godoc
// @Summary Get all subscriptions
// @Description Get all subscriptions
// @Accept json
// @Produce json
// @Success 200 {object} ListResponse
// @Router /subscriptions [get]
func NewListHandler(logger *slog.Logger, listReader ListReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.list"

		logger := logger.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		// 1.Get optional params and validate it
		limit, offset, ok := getValidatedOptParams(r, w, logger)
		if !ok {
			return
		}

		// 2.Get subscriptions
		var subscriptions []model.Subscription
		var err error

		if limit == 0 && offset == 0 {
			subscriptions, err = listReader.GetSubscriptions(nil, nil)
		} else {
			subscriptions, err = listReader.GetSubscriptions(&limit, &offset)
		}

		if err != nil {
			logger.Error("failed to get subscription", "details", err)

			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, ListResponse{Response: RespError("failed to get subscription")})

			return
		}

		logger.Info("got subscriptions")

		// 3.Prepare response and render it
		resp := makeListResp(subscriptions)
		render.JSON(w, r, resp)
	}
}

func getValidatedOptParams(r *http.Request, w http.ResponseWriter, logger *slog.Logger) (int, int, bool) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	if limitStr == "" && offsetStr == "" {
		return 0, 0, true
	}
	if limitStr != "" && offsetStr == "" {
		logger.Error("no offset value while limit is set")

		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, ListResponse{Response: RespError("no offset value while limit is set")})

		return 0, 0, false
	}
	if limitStr == "" && offsetStr != "" {
		logger.Error("no offset value while limit is set")

		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, ListResponse{Response: RespError("no limit value while offset is set")})

		return 0, 0, false
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		logger.Error("invalid limit format", "details", err)

		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, ListResponse{Response: RespError("invalid limit format")})

		return 0, 0, false
	}
	if limit < 0 {
		logger.Error("invalid limit value (less than zero)")

		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, ListResponse{Response: RespError("invalid limit value (less than zero)")})

		return 0, 0, false
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		logger.Error("invalid offset format", "details", err)

		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, ListResponse{Response: RespError("invalid offset format")})

		return 0, 0, false
	}
	if offset < 0 {
		logger.Error("invalid offset value (less than zero)")

		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, ListResponse{Response: RespError("invalid offset value (less than zero)")})

		return 0, 0, false
	}

	return limit, offset, true
}

func makeListResp(subscriptions []model.Subscription) ListResponse {
	resp := ListResponse{
		Items:    []ListItem{},
		Response: RespOK(),
	}

	for i := 0; i < len(subscriptions); i++ {
		item := ListItem{
			Id:          subscriptions[i].ID,
			ServiceName: subscriptions[i].ServiceName,
			Price:       subscriptions[i].Price,
			UserID:      subscriptions[i].UserID.String(),
			StartDate:   subscriptions[i].StartDate.ToString(),
			EndDate:     subscriptions[i].EndDate.ToString(),
		}
		resp.Items = append(resp.Items, item)
	}

	return resp
}
