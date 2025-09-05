package internal

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func SetupMiddleware(e *echo.Echo) {
	e.Use(
		mRequestID,
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{"http://localhost:5173"},
		}),
		middleware.RecoverWithConfig(middleware.RecoverConfig{
			LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
				slog.ErrorContext(
					c.Request().Context(),
					"panic recovered",
					slog.String("request_id", c.Get("request_id").(string)),
					slog.String("method", c.Request().Method),
					slog.String("url", c.Request().URL.String()),
					slog.Any("error", err),
				)
				slog.DebugContext(c.Request().Context(), string(stack))
				return err
			},
		}),
		mLogger,
	)
}

func mRequestID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}

		c.Set("request_id", id.String())
		c.Response().Header().Set("X-Request-ID", id.String())
		return next(c)
	}
}

func mLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Get("request_id").(string)
		req := c.Request()
		slog.Info(
			fmt.Sprintf("← h %s %s", req.Method, req.URL),
			slog.String("request_id", id),
		)

		start := time.Now()
		err := next(c)
		took := time.Since(start)

		res := c.Response()
		slog.Info(
			fmt.Sprintf("← h %s %s --- took ~ %s", req.Method, req.URL, took),
			slog.String("request_id", id),
			slog.Int("status", res.Status),
			slog.Duration("duration", took),
			slog.String("ip", c.RealIP()),
		)

		return err
	}
}
