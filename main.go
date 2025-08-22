package main

import (
	"log/slog"
	"os"
	"slices"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal"
	"github.com/vyxn/yuzu/internal/pkg/log"
)

var env = os.Getenv("YUZU_ENV")
var logger *slog.Logger

func init() {
	logger = log.NewLogger()
}

func main() {
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

	internal.SetupMiddleware(e)
	internal.SetupRoutes(e)

	port := ":8080"
	logger.Info("http server started", slog.String("port", port))
	e.Logger.Fatal(e.Start(port))
}
