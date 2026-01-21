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

// UpdateRequest represents subscription for update model
// swagger:model UpdateRequest
// @ID UpdateRequest
type UpdateRequest struct {
	// New service name (required)
	ServiceName string `json:"service_name"`

	// New price (required)
	Price int `json:"price"`

	// New start date
	StartDate string `json:"start_date"`

	// New end date (optional)
	EndDate string `json:"end_date,omitempty"`
}

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=Updater
type Updater interface {
	UpdateSubscription(id int64, newServiceName string, newPrice int, newStart, newEnd model.Date) error
}

// NewUpdateHandler godoc
// @Summary Update subscription
// @Description Update subscription
// @Accept json
// @Produce json
// @Param id path int true "Subscription ID"
// @Param request body UpdateRequest true "Subscription new data"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /subscription/{id} [patch]
func NewUpdateHandler(logger *slog.Logger, updater Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.update"

		logger := logger.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		// 1.Get subscription id from request
		idStr := chi.URLParam(r, "id")
		if idStr == "" {
			logger.Info("no subscription id in request")

			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, RespError("no subscription id in request"))

			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Info("invalid subscription id format", "details", err)

			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, RespError("invalid subscription id format"))

			return
		}

		// 2.Parse request body
		var req UpdateRequest
		if ok := parseReq(r, w, logger, &req); !ok {
			return
		}

		// 3.Validate request body data
		validateOk := validateUpdateReq(r, w, &req, logger)
		if !validateOk {
			return
		}

		// 3.Fill end_date with value if need
		startDate, _ := model.DateFromString(req.StartDate)

		endDate := model.Date{}
		if req.EndDate != "" {
			endDate, _ = model.DateFromString(req.EndDate)
		}

		// 4.Update
		err = updater.UpdateSubscription(int64(id), req.ServiceName, req.Price, startDate, endDate)
		if errors.Is(err, storage.ErrSubscribtionNotFound) {
			logger.Info("subscription not found", "id", id)

			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, RespError("subscription not found"))

			return
		}
		if err != nil {
			logger.Error("failed to update subscription", "details", err)

			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, RespError("failed to get subscription"))

			return
		}

		logger.Info("updated subscription",
			"id", id,
			"new_price", req.Price,
			"new_end_date", req.EndDate,
		)

		// 3.Prepare response and render it
		render.JSON(w, r, RespOK())
	}
}

func validateUpdateReq(r *http.Request, w http.ResponseWriter, req *UpdateRequest, logger *slog.Logger) bool {
	// 1.Service name
	if req.ServiceName == "" {
		logger.Error("request service name is empty")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, RespError("request service name is empty"))
		return false
	}

	// 2.Price
	if req.Price < 0 {
		logger.Error("request price cannot be lower than 0")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, RespError("request price is invalid"))
		return false
	}

	// 3.Start date
	if req.StartDate == "" {
		logger.Error("request start date is empty")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, RespError("request start date is empty"))
		return false
	}
	_, err := model.DateFromString(req.StartDate)
	if err != nil {
		logger.Error("request start date is invalid", "details", err)
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, RespError("request start date is invalid"))
		return false
	}

	// 4.End date
	if req.EndDate != "" {
		_, err := model.DateFromString(req.EndDate)
		if err != nil {
			logger.Error("request end date is invalid", "details", err)
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, RespError("request end date is invalid"))
			return false
		}
	}

	return true
}
