package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
)

func parseReq[T any](r *http.Request, w http.ResponseWriter, logger *slog.Logger, req *T) bool {
	err := render.DecodeJSON(r.Body, &req)

	if errors.Is(err, io.EOF) {
		logger.Error("request body is empty")

		render.JSON(w, r, RespError("empty request"))

		return false
	}

	if err != nil {
		logger.Error("failed to decode request body", "details", err)

		render.JSON(w, r, RespError("failed to decode request"))

		return false
	}

	logger.Info("request body decoded", slog.Any("request", req))

	return true
}
