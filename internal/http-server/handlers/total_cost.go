package handlers

import (
	"em_golang_rest_service_example/internal/model"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// TotalCostRequest contains filters for calculate needed total cost
// swagger:model TotalCostRequest
// @ID TotalCostRequest
type TotalCostRequest struct {
	// Start date of subscription (required)
	StartDate string `json:"start_date"`

	// Start date of subscription (required)
	EndDate string `json:"end_date"`

	// If of user who purchased the subscription (optional)
	UserID string `json:"user_id,omitempty"`

	// Subscription service name (optional)
	ServiceName string `json:"service_name,omitempty"`
}

// TotalCostResponse contains calculated total cost
// swagger:model TotalCostResponse
// @ID TotalCostResponse
type TotalCostResponse struct {
	// Calculated total cost
	TotalCost int `json:"total_cost"`

	Response
}

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=FilteredDataReader
type FilteredDataReader interface {
	FilterSubscriptions(startDate, endDate model.Date, userId uuid.UUID, serviceName *string) ([]model.Subscription, error)
}

// NewTotalCostHandler godoc
// @Summary Calculate total cost with specified filters
// @Description Calculate total cost with specified filters
// @Accept json
// @Produce json
// @Param request body TotalCostRequest true "filters data"
// @Success 200 {object} TotalCostResponse
// @Router /subscriptions/total-cost [get]
func NewTotalCostHandler(logger *slog.Logger, dataReader FilteredDataReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.total_cost"

		logger := logger.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		// 1.Parse and validate URL data
		start, end, uid, serviceName, ok := getValidatedReqData(r, w, logger)
		if !ok {
			return
		}

		// 3.Get filtered subscriptions
		var sNamePtr *string
		if serviceName != "" {
			sNamePtr = &serviceName
		}

		subscriptions, err := dataReader.FilterSubscriptions(start, end, uid, sNamePtr)
		if err != nil {
			logger.Error("failed to get subscription", "details", err)

			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, TotalCostResponse{Response: RespError("failed to get subscription")})

			return
		}

		// 4.Calculate
		totalCost := calculateTotalCostFiltered(subscriptions)

		logger.Info("got filtered subscriptions total cost", "value", totalCost)

		// 5.Prepare response and render it
		resp := TotalCostResponse{
			TotalCost: totalCost,
			Response:  RespOK(),
		}
		render.JSON(w, r, resp)
	}
}

func getValidatedReqData(r *http.Request, w http.ResponseWriter, logger *slog.Logger) (model.Date, model.Date, uuid.UUID, string, bool) {
	// 1.Dates
	startDateStr := r.URL.Query().Get("start_date")
	if startDateStr == "" {
		logger.Error("request start date is empty")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, TotalCostResponse{Response: RespError("empty start date")})
		return model.Date{}, model.Date{}, uuid.Nil, "", false
	}

	startDate, err := model.DateFromString(startDateStr)
	if err != nil {
		logger.Error("request start date is invalid", "details", err)
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, TotalCostResponse{Response: RespError("request start date is invalid")})
		return model.Date{}, model.Date{}, uuid.Nil, "", false
	}

	endDateStr := r.URL.Query().Get("end_date")
	if endDateStr == "" {
		logger.Error("request end date is empty")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, TotalCostResponse{Response: RespError("empty end date")})
		return model.Date{}, model.Date{}, uuid.Nil, "", false
	}

	endDate, err := model.DateFromString(endDateStr)
	if err != nil {
		logger.Error("request end date is invalid", "details", err)
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, TotalCostResponse{Response: RespError("request end date is invalid")})
		return model.Date{}, model.Date{}, uuid.Nil, "", false
	}

	if startDate.GreaterThan(endDate) {
		logger.Error("request start date greater than end date")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, TotalCostResponse{Response: RespError("request start date greater than end date")})
		return model.Date{}, model.Date{}, uuid.Nil, "", false
	}

	// 2.User ID if have
	userId := uuid.Nil

	userIdStr := r.URL.Query().Get("user_id")
	if userIdStr != "" {
		userId, err = uuid.Parse(userIdStr)
		if err != nil {
			logger.Error("user id filter is invalid", "details", err)

			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, TotalCostResponse{Response: RespError("user id filter is invalid")})

			return model.Date{}, model.Date{}, uuid.Nil, "", false
		}
	}

	// 3.Service name if have
	serviceName := r.URL.Query().Get("service_name")

	return startDate, endDate, userId, serviceName, true
}

func calculateTotalCostFiltered(subs []model.Subscription) int {
	cost := 0

	for i := 0; i < len(subs); i++ {
		monthDiff := model.MonthsBetween(subs[i].StartDate, subs[i].EndDate)
		cost += subs[i].Price * monthDiff
	}

	return cost
}
