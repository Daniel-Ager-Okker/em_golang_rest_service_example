package handlers

import (
	"em_golang_rest_service_example/internal/model"
	"em_golang_rest_service_example/internal/storage"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// CreateRequest represents subscription model
// swagger:model CreateRequest
// @ID CreateRequest
type CreateRequest struct {
	// Subscription service name (required)
	ServiceName string `json:"service_name"`

	// Subscription monthly price (required)
	Price int `json:"price"`

	// If of user who purchased the subscription (required)
	UserID string `json:"user_id"`

	// Start date of subscription (required)
	StartDate string `json:"start_date"`

	// Start date of subscription (optional)
	EndDate string `json:"end_date,omitempty"`
}

// CreateResponse represents response with id on subscription creation
// swagger:model CreateResponse
// @ID CreateResponse
type CreateResponse struct {
	// Subscription identifier
	ID int64 `json:"id"`

	Response
}

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=Creator
type Creator interface {
	CreateSubscription(subscription model.SubscriptionSpec) (int64, error)
}

// NewCreateHandler godoc
// @Summary Create new subscription
// @Description Create new subscription
// @Accept json
// @Produce json
// @Param request body CreateRequest true "Subscription data"
// @Success 201 {object} CreateResponse
// @Router /subscription [post]
func NewCreateHandler(logger *slog.Logger, creator Creator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.create"

		logger := logger.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		// 1.Parse request
		var req CreateRequest
		if ok := parseReq(r, w, logger, &req); !ok {
			return
		}

		// 2.Validate request data
		validateOk := validateCreateReq(r, w, &req, logger)
		if !validateOk {
			return
		}

		// 3.Prepare subscription
		spec := prepareSubscriptionSpec(&req)

		// 4.Create
		id, err := creator.CreateSubscription(spec)
		if errors.Is(err, storage.ErrSubscriptionExists) {
			logger.Info("subscription already exists", "service_name", req.ServiceName, "user_id", req.UserID)

			w.WriteHeader(http.StatusConflict)
			render.JSON(w, r, CreateResponse{Response: RespError("subscription already exists")})

			return
		}
		if err != nil {
			logger.Error("failed to create subscription", "details", err)

			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, CreateResponse{Response: RespError("failed to create subscription")})

			return
		}

		logger.Info("subscription created", "id", id)

		w.WriteHeader(http.StatusCreated)
		render.JSON(w, r, CreateResponse{ID: id, Response: RespOK()})
	}
}

func validateCreateReq(r *http.Request, w http.ResponseWriter, req *CreateRequest, logger *slog.Logger) bool {
	// 1.Service name
	if req.ServiceName == "" {
		logger.Error("request serivce name is empty")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, CreateResponse{Response: RespError("empty service name")})
		return false
	}

	// 2.Price
	if req.Price < 0 {
		logger.Error("request price cannot be lowe than 0")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, CreateResponse{Response: RespError("request price is invalid")})
		return false
	}

	// 3.User ID
	if req.UserID == "" {
		logger.Error("request user id is empty")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, CreateResponse{Response: RespError("empty user id")})
		return false
	}
	_, err := uuid.Parse(req.UserID)
	if err != nil {
		logger.Error("request user id is invalid", "details", err)
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, CreateResponse{Response: RespError("request user id is invalid")})
		return false
	}

	// 4.Dates
	if req.StartDate == "" {
		logger.Error("request start date is empty")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, CreateResponse{Response: RespError("empty start date")})
		return false
	}

	startDate, err := model.DateFromString(req.StartDate)
	if err != nil {
		logger.Error("request start date is invalid", "details", err)
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, CreateResponse{Response: RespError("request start date is invalid")})
		return false
	}

	if req.EndDate != "" {
		endDate, err := model.DateFromString(req.EndDate)
		if err != nil {
			logger.Error("request end date is invalid", "details", err)
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, CreateResponse{Response: RespError("request end date is invalid")})
			return false
		}

		if startDate.GreaterThan(endDate) {
			logger.Error("request start date greater than end date")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, CreateResponse{Response: RespError("request start date greater than end date")})
			return false
		}
	}

	return true
}

func prepareSubscriptionSpec(req *CreateRequest) model.SubscriptionSpec {
	uid, _ := uuid.Parse(req.UserID)

	startDate, _ := model.DateFromString(req.StartDate)

	endDate := model.Date{}
	if req.EndDate == "" {
		endDate = startDate.AddDate(0, 1)
	} else {
		endDate, _ = model.DateFromString(req.EndDate)
	}

	return model.SubscriptionSpec{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      uid,
		StartDate:   startDate,
		EndDate:     endDate,
	}
}
