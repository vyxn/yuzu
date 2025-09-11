package handler

import (
	_ "embed"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:embed favicon.ico
var favicon []byte

func registerStatic(e *echo.Echo) {
	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "image/x-icon", favicon)
	})
}
