package main

import (
	_ "em_golang_rest_service_example/docs"
	"em_golang_rest_service_example/internal/config"
	"em_golang_rest_service_example/internal/http-server/handlers"
	mwLogger "em_golang_rest_service_example/internal/http-server/middleware/logger"
	"em_golang_rest_service_example/internal/model"
	pg "em_golang_rest_service_example/internal/storage/postgres"
	"em_golang_rest_service_example/internal/storage/sqlite"

	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// 1.Configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error while reading configuration: %v\n", err)
		return
	}

	// 2.Logger
	logger := setupLogger(cfg.Env)
	logger = logger.With(slog.String("env", cfg.Env))

	var router *chi.Mux

	// 3.Storage
	var repo Repo

	switch cfg.Env {
	case config.DevEnv:
		sqliteRepo, err := sqlite.NewStorage(&cfg.StorageCfg.StoragePath, logger)
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			return
		}
		defer sqliteRepo.Close()

		repo = &sqliteRepo

	case config.ProdEnv:
		pgRepo, err := pg.NewStorage(&cfg.StorageCfg, logger)
		if err != nil {
			fmt.Printf("Failed to initialize storage: %v\n", err)
			return
		}
		defer pgRepo.Close()

		repo = &pgRepo

	default:
		fmt.Printf("Error: unsupported configuration env")
		return
	}

	// 4.Router
	router = setupRouter(logger, repo)

	// 5.Starting
	logger.Info("starting server", "address", cfg.Address)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Error("failed to start server (or server stopping now)")
		}
	}()
	logger.Info("server started")

	// 6.Stopping
	<-done
	logger.Info("stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("failed to stop server", "details", err)
		return
	}

	logger.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case config.DevEnv:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case config.ProdEnv:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

type Repo interface {
	CreateSubscription(subscription model.SubscriptionSpec) (int64, error)
	GetSubscription(id int64) (model.Subscription, error)
	UpdateSubscription(id int64, newServiceName string, newPrice int, newStart, newEnd model.Date) error
	DeleteSubscription(id int64) error
	GetSubscriptions(limit, offset *int) ([]model.Subscription, error)
}

func setupRouter(l *slog.Logger, repo Repo) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.RequestID) // tracing purposes
	router.Use(mwLogger.New(l))      // logging purposes (using our logger implementation)
	router.Use(middleware.Recoverer) // for panic recovering while handler failing
	router.Use(middleware.URLFormat) // URL parser

	router.Post("/subscription", handlers.NewCreateHandler(l, repo))
	router.Get("/subscription/{id}", handlers.NewReadHandler(l, repo))
	router.Get("/subscriptions", handlers.NewListHandler(l, repo))
	router.Patch("/subscription/{id}", handlers.NewUpdateHandler(l, repo))
	router.Delete("/subscription/{id}", handlers.NewDeleteHandler(l, repo))
	router.Get("/subscriptions/total-cost", handlers.NewTotalCostHandler(l, repo))

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":       "endpoint_not_found",
			"message":     "requested API endpoint not found",
			"path":        r.URL.Path,
			"method":      r.Method,
			"status_code": http.StatusNotFound,
		})
	})

	return router
}
