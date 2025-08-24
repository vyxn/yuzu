package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

var isDev = os.Getenv("APP_ENV") != "production"

func SetupErrorHandling(e *echo.Echo) {
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		id := c.Get("request_id").(string)
		var he *echo.HTTPError
		if errors.As(err, &he) {
			data := map[string]any{
				"status":  he.Code,
				"message": he.Message,
			}
			if isDev && he.Internal != nil {
				data["error"] = he.Internal.Error()
				if stack := yerr.GetStack(he.Internal); stack != nil {
					data["stack"] = stack
				}
			}

			// Known HTTP error
			if !c.Response().Committed {
				c.JSON(he.Code, data)
			}
			msg := "HTTP error"
			attrs := []any{
				slog.String("request_id", id),
				slog.Int("status", he.Code),
				slog.String("message", fmt.Sprint(he.Message)),
			}
			if isDev && he.Internal != nil {
				attrs = append(attrs, slog.Any("error", he.Internal.Error()))
				if stack := yerr.GetStack(he.Internal); stack != nil {
					msg += "\n" + strings.Join(stack, "\n") + "\n"
				}
			}
			slog.Warn(msg, attrs...)
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
