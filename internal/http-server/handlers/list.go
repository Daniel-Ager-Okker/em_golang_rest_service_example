package handlers

import (
	"em_golang_rest_service_example/internal/model"
	"log/slog"
	"net/http"

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
	GetSubscriptions() ([]model.Subscription, error)
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

		// 1.Get subscriptions
		subscriptions, err := listReader.GetSubscriptions()
		if err != nil {
			logger.Error("failed to get subscription", "details", err)

			render.JSON(w, r, ListResponse{Response: RespError("failed to get subscription")})

			return
		}

		logger.Info("got subscriptions")

		// 3.Prepare response and render it
		resp := makeListResp(subscriptions)
		render.JSON(w, r, resp)
	}
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
