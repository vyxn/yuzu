package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal"
	"github.com/vyxn/yuzu/internal/pkg/log"
)

var env = os.Getenv("YUZU_ENV")

func main() {
	logger := log.NewLogger()
	slog.SetDefault(logger)

	envs := []string{".env"}
	if env != "test" {
		envs = append(envs, ".env.local")
	}
	if env != "" && env != "development" {
		envs = append(envs, ".env."+env)
	}
	slices.Reverse(envs)
	if err := godotenv.Load(envs...); err != nil {
		slog.Error("error loading env", slog.Any("error", err))
		os.Exit(1)
	}

	db := internal.GetDB()
	err := db.Ping()
	if err != nil {
		panic(err)
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Logger.SetOutput(os.Stdout)

	internal.SetupMiddleware(e)
	internal.SetupErrorHandling(e)
	internal.SetupRoutes(e)

	port := ":8080"
	logger.Info("http server started", slog.String("port", port))
	go func() {
		if err := e.Start(port); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error("error on server run", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		slog.Error("error stopping the server", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}
