package handlers

import (
	"em_golang_rest_service_example/internal/model"
	"em_golang_rest_service_example/internal/storage"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// ReadResponse represents response with subscription was read
// swagger:model ReadResponse
// @ID ReadResponse
type ReadResponse struct {
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

	Response
}

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=Reader
type Reader interface {
	GetSubscription(id int64) (model.Subscription, error)
}

// NewReadHandler godoc
// @Summary Read subscription
// @Description Read subscription
// @Produce json
// @Success 200 {object} ReadResponse
// @Router /subscription/{id} [get]
func NewReadHandler(logger *slog.Logger, reader Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.read"

		logger := logger.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1.Get subscription id from request
		idStr := chi.URLParam(r, "id")
		if idStr == "" {
			logger.Info("no subscription id in request")

			render.JSON(w, r, ReadResponse{Response: RespError("no subscription id in request")})

			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Info("invalid subscription id format", "details", err)

			render.JSON(w, r, ReadResponse{Response: RespError("invalid subscription id format")})

			return
		}

		// 2.Get subscription
		subscription, err := reader.GetSubscription(int64(id))
		if errors.Is(err, storage.ErrSubscribtionNotFound) {
			logger.Info("subscription not found", "id", id)

			render.JSON(w, r, ReadResponse{Response: RespError("subscription not found")})

			return
		}
		if err != nil {
			logger.Error("failed to get subscription", "details", err)

			render.JSON(w, r, ReadResponse{Response: RespError("failed to get subscription")})

			return
		}

		logger.Info("got subscription",
			"id", subscription.ID,
			"service_name", subscription.ServiceName,
			"price", subscription.Price,
			"user_id", subscription.UserID,
			"start_date", subscription.StartDate.ToString(),
			"end_date", subscription.EndDate.ToString(),
		)

		// 3.Prepare response and render it
		resp := makeReadResp(&subscription)
		render.JSON(w, r, resp)
	}
}

func makeReadResp(subscription *model.Subscription) ReadResponse {
	return ReadResponse{
		Id:          subscription.ID,
		ServiceName: subscription.ServiceName,
		Price:       subscription.Price,
		UserID:      subscription.UserID.String(),
		StartDate:   subscription.StartDate.ToString(),
		EndDate:     subscription.EndDate.ToString(),
		Response:    RespOK(),
	}
}
