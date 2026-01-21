package handlers

import (
	"em_golang_rest_service_example/internal/storage"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=Deleter
type Deleter interface {
	DeleteSubscription(id int64) error
}

// NewDeleteHandler godoc
// @Summary Delete subscription
// @Description Delete subscription
// @Produce json
// @Param id path int true "Subscription ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /subscription/{id} [delete]
func NewDeleteHandler(logger *slog.Logger, deleter Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.delete"

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

		// 2.Delete subscription
		err = deleter.DeleteSubscription(int64(id))
		if errors.Is(err, storage.ErrSubscribtionNotFound) {
			logger.Info("subscription not found", "id", id)

			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, RespError("subscription not found"))

			return
		}
		if err != nil {
			logger.Error("failed to delete subscription", "details", err)

			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, RespError("failed to delete subscription"))

			return
		}

		logger.Info("deleted subscription", "id", id)

		// 3.Render response
		render.JSON(w, r, RespOK())
	}
}
