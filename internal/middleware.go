package internal

import (
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal/pkg/log"
)

var logger *slog.Logger

func init() {
	logger = log.NewLogger()
}

func SetupMiddleware(e *echo.Echo) {
	e.Use(mLogger)
}

// Middleware
func mLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		logger.Info(fmt.Sprintf("← h %s %s", req.Method, req.URL))
		return next(c)
	}
}
