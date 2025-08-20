package main

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal"
	"github.com/vyxn/yuzu/internal/pkg/log"
)

var logger *slog.Logger

func init() {
	logger = log.NewLogger()
}

func main() {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	internal.SetupMiddleware(e)
	internal.SetupRoutes(e)

	port := ":8080"
	logger.Info("http server started", slog.String("port", port))
	e.Logger.Fatal(e.Start(port))
}
