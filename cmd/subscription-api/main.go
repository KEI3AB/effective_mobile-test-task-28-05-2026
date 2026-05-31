package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/effective_mobile-test-task-28-05-2026/config"
	"github.com/effective_mobile-test-task-28-05-2026/internal/repository/postgres"
	myhttp "github.com/effective_mobile-test-task-28-05-2026/internal/transport/http"
	"github.com/effective_mobile-test-task-28-05-2026/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/effective_mobile-test-task-28-05-2026/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           API Агрегатора Подписок
// @version         1.0
// @description     REST-сервис для управления и агрегации данных об онлайн-подписках пользователей.
// @contact.name    Junior Golang Developer
// @host            localhost:8080
// @BasePath        /api/v1
func main() {
	// LOGGER
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	logger.Info("starting subscription api service")

	// CONFIG
	cfg := config.MustLoad()
	logger.Info("config loaded", slog.String("port", cfg.HTTP.Port))

	// POSTGRES
	ctx := context.Background()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?pool_max_conns=%d",
		cfg.PG.User, cfg.PG.Password, cfg.PG.Host, cfg.PG.Port, cfg.PG.DBName, cfg.PG.PoolMax,
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Error("failed to create connection pool", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", slog.String("err", err.Error()))
		os.Exit(1)
	}
	logger.Info("connected to postgres successfully")

	// LAYERS
	repo := postgres.NewSubscriptionStorage(pool)
	uc := usecase.NewSubscriptionUseCase(repo)
	handler := myhttp.NewSubscriptionHandler(uc)

	// ROUTER
	r := chi.NewRouter()

	// MIDDLEWARE
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// ROUTES
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/subscriptions", handler.CreateSubscription)
		r.Get("/subscriptions", handler.ListSubscriptions)
		r.Get("/costs", handler.CalculateTotalCost)

		r.Route("/subscriptions/{id}", func(r chi.Router) {
			r.Get("/", handler.GetSubscription)
			r.Put("/", handler.UpdateSubscription)
			r.Delete("/", handler.DeleteSubscription)
		})
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// START SERVER
	srv := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("server is running", slog.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start server", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	// STOP SERVER
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", slog.String("err", err.Error()))
	}

	logger.Info("server stopped")
}
