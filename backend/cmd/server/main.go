package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"taskflow/internal/auth"
	"taskflow/internal/config"
	"taskflow/internal/database"
	"taskflow/internal/middleware"
	"taskflow/internal/project"
	"taskflow/internal/response"
	"taskflow/internal/task"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	if cfg.JWTSecret == "" {
		slog.Error("JWT_SECRET is required")
		os.Exit(1)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Repositories
	authRepo := auth.NewRepository(db)
	projectRepo := project.NewRepository(db)
	taskRepo := task.NewRepository(db)

	// Services
	authSvc := auth.NewService(authRepo, cfg.JWTSecret, cfg.BcryptCost)
	projectSvc := project.NewService(projectRepo)
	taskSvc := task.NewService(taskRepo)

	// Handlers
	authHandler := auth.NewHandler(authSvc)
	projectHandler := project.NewHandler(projectSvc, taskSvc)
	taskHandler := task.NewHandler(taskSvc)

	// Middleware
	authMw := middleware.NewAuthMiddleware(cfg.JWTSecret)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger)
	r.Use(chiMiddleware.Recoverer)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, http.StatusNotFound, "not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	// Public routes (rate-limited: 5 requests/second, burst of 10)
	authLimiter := middleware.NewRateLimiter(5, 10)
	r.Group(func(r chi.Router) {
		r.Use(authLimiter.Limit)
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMw.Authenticate)

		r.Get("/projects", projectHandler.List)
		r.Post("/projects", projectHandler.Create)
		r.Get("/projects/{id}", projectHandler.GetByID)
		r.Get("/projects/{id}/stats", projectHandler.Stats)
		r.Patch("/projects/{id}", projectHandler.Update)
		r.Delete("/projects/{id}", projectHandler.Delete)

		r.Get("/projects/{id}/tasks", taskHandler.List)
		r.Post("/projects/{id}/tasks", taskHandler.Create)
		r.Patch("/tasks/{id}", taskHandler.Update)
		r.Delete("/tasks/{id}", taskHandler.Delete)
	})

	// Server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}
