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

	// "github.com/robfig/cron/v3"
	"github.com/vyxn/yuzu/internal"
	"github.com/vyxn/yuzu/internal/config"
	"github.com/vyxn/yuzu/internal/handler"
	// "github.com/vyxn/yuzu/internal/library"
	"github.com/vyxn/yuzu/internal/pkg/log"
	"github.com/vyxn/yuzu/internal/repository"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

var env = os.Getenv("APP_ENV")

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt, syscall.SIGTERM,
	)
	defer stop()

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

	libRepo, err := repository.NewFileLibraryRepository(
		ctx,
		"libraries",
		config.Cfg.Paths,
	)
	if err != nil {
		panic(err)
	}

	if err := config.Load(); err != nil {
		panic(err)
	}
	config.WatchProviders(ctx)

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
	handler.SetupRoutes(e, libRepo)

	// c := cron.New()
	// libs, err := libRepo.GetAll()
	// for _, lib := range libs {
	// 	lib.ScheduleJobs(c)
	// }
	// go c.Run()
	// defer func() {
	// 	ctx := c.Stop()
	// 	<-ctx.Done()
	// }()

	port := ":8080"
	logger.Info("http server started", slog.String("port", port))
	go func() {
		if err := e.Start(port); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error("error on server run", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("stopping server...")

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("error stopping the server", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}
