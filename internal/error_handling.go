package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

func SetupErrorHandling(e *echo.Echo) {
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		id := c.Get("request_id").(string)
		var he *echo.HTTPError
		if errors.As(err, &he) {
			// Known HTTP error
			if !c.Response().Committed {
				c.JSON(he.Code, map[string]any{"error": he.Message})
			}
			slog.Warn(
				"HTTP error",
				slog.String("request_id", id),
				slog.Int("status", he.Code),
				slog.String("message", fmt.Sprint(he.Message)),
			)
		} else {
			// Unexpected error
			slog.Error("internal server error",
				slog.String("request_id", id),
				slog.Any("error", err),
			)
			if !c.Response().Committed {
				c.JSON(http.StatusInternalServerError, map[string]any{
					"error": "internal server error",
				})
			}
		}
	}
}
