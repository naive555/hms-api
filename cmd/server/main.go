package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/naive555/hms-api/internal/auth"
	"github.com/naive555/hms-api/internal/config"
	"github.com/naive555/hms-api/internal/handler"
	"github.com/naive555/hms-api/internal/repository"
	"github.com/naive555/hms-api/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}

	hospitalRepo := repository.NewHospitalRepo(pool)
	staffRepo := repository.NewStaffRepo(pool)
	tokens := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)
	authSvc := service.NewAuthService(hospitalRepo, staffRepo, tokens, bcrypt.DefaultCost)

	r := handler.NewRouter(handler.NewStaffHandler(authSvc))

	srv := &http.Server{Addr: ":" + cfg.AppPort, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
