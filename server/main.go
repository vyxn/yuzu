package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/vyxn/yuzu/internal"
	"github.com/vyxn/yuzu/internal/config"
	"github.com/vyxn/yuzu/internal/pkg/log"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

var env = os.Getenv("APP_ENV")

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if err := config.Load(); err != nil {
		panic(err)
	}
	config.WatchProviders(ctx)
	// go provider.Watch(
	// 	ctx,
	// 	"config/providers",
	// 	config.LoadProvider,
	// 	config.UnloadProvider,
	// )

	config.Info()

	db := internal.GetDB()
	if err := db.Ping(); err != nil {
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
	defer close(quit)

	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	s := <-quit
	slog.Info("stopping server...", slog.String("signal", s.String()))

	ctx, cancel = context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("error stopping the server", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}
