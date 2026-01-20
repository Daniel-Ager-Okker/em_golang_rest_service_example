package handlers

import (
	"em_golang_rest_service_example/internal/model"
	"log/slog"
	"net/http"
	"strings"

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

// NewTotalCostHandler godoc
// @Summary Calculate total cost with specified filters
// @Description Calculate total cost with specified filters
// @Accept json
// @Produce json
// @Param request body TotalCostRequest true "filters data"
// @Success 200 {object} TotalCostResponse
// @Router /subscriptions/total-cost [get]
func NewTotalCostHandler(logger *slog.Logger, listReader ListReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.total_cost"

		logger := logger.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		// 1.Parse request
		var req TotalCostRequest
		if ok := parseReq(r, w, logger, &req); !ok {
			return
		}

		// 2.Validate request data
		validateOk := validateTotalCostReq(r, w, &req, logger)
		if !validateOk {
			return
		}

		// 3.Get all subscriptions
		subscriptions, err := listReader.GetSubscriptions(nil, nil)
		if err != nil {
			logger.Error("failed to get subscription", "details", err)

			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, TotalCostResponse{Response: RespError("failed to get subscription")})

			return
		}

		// 4.Filter it
		totalCost := calculateTotalCostFiltered(subscriptions, &req)

		logger.Info("got filtered subscriptions total cost", "value", totalCost)

		// 5.Prepare response and render it
		resp := TotalCostResponse{
			TotalCost: totalCost,
			Response:  RespOK(),
		}
		render.JSON(w, r, resp)
	}
}

func validateTotalCostReq(r *http.Request, w http.ResponseWriter, req *TotalCostRequest, logger *slog.Logger) bool {
	// 1.Dates
	if req.StartDate == "" {
		logger.Error("request start date is empty")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, TotalCostResponse{Response: RespError("empty start date")})
		return false
	}

	startDate, err := model.DateFromString(req.StartDate)
	if err != nil {
		logger.Error("request start date is invalid", "details", err)
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, TotalCostResponse{Response: RespError("request start date is invalid")})
		return false
	}

	if req.EndDate == "" {
		logger.Error("request end date is empty")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, TotalCostResponse{Response: RespError("empty end date")})
		return false
	}

	endDate, err := model.DateFromString(req.EndDate)
	if err != nil {
		logger.Error("request end date is invalid", "details", err)
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, TotalCostResponse{Response: RespError("request end date is invalid")})
		return false
	}

	if startDate.GreaterThan(endDate) {
		logger.Error("request start date greater than end date")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, TotalCostResponse{Response: RespError("request start date greater than end date")})
		return false
	}

	// 2.User ID if have
	if req.UserID != "" {
		_, err := uuid.Parse(req.UserID)
		if err != nil {
			logger.Error("user id filter is invalid", "details", err)

			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, TotalCostResponse{Response: RespError("user id filter is invalid")})

			return false
		}
	}

	return true
}

func calculateTotalCostFiltered(subs []model.Subscription, req *TotalCostRequest) int {
	cost := 0

	// 1.Get start/end bounds as dates
	startBound, _ := model.DateFromString(req.StartDate)
	endBound, _ := model.DateFromString(req.EndDate)

	for i := 0; i < len(subs); i++ {
		// 1.Range check
		start := subs[i].StartDate
		startOk := start.EqualTo(startBound) || start.GreaterThan(startBound)
		if !startOk {
			continue
		}

		end := subs[i].EndDate
		endOk := end.EqualTo(endBound) || endBound.GreaterThan(end)
		if !endOk {
			continue
		}

		// 2.User id filtering if need
		if req.UserID != "" {
			uid, _ := uuid.Parse(req.UserID)
			if subs[i].UserID != uid {
				continue
			}
		}

		// 3.Service name filtering if need
		if req.ServiceName != "" {
			if strings.Compare(req.ServiceName, subs[i].ServiceName) != 0 {
				continue
			}
		}

		cost += subs[i].Price
	}

	return cost
}
