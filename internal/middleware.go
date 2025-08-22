package internal

import (
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/vyxn/yuzu/internal/pkg/log"
)

var logger *slog.Logger

func init() {
	logger = log.NewLogger()
}

func SetupMiddleware(e *echo.Echo) {
	e.Use(
		mLogger,
		middleware.RecoverWithConfig(middleware.RecoverConfig{
			LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
				logger.ErrorContext(
					c.Request().Context(),
					"panic recovered",
					slog.String("method", c.Request().Method),
					slog.String("url", c.Request().URL.String()),
					slog.Any("error", err),
					slog.String("stack", string(stack)),
				)
				return err
			},
		}),
	)
}

// Middleware
func mLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		logger.Info(fmt.Sprintf("← h %s %s", req.Method, req.URL))
		return next(c)
	}
}
