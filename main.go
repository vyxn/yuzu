package main

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	// Routes
	e.GET("/", hello)

	// Start server
	if err := e.Start(":8080"); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
	}
}

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
