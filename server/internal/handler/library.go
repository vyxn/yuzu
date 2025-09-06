package handler

import (
	"net/http"

	"github.com/vyxn/yuzu/internal/lib"

	"github.com/labstack/echo/v4"
)

func registerLibrary(e *echo.Echo) {
	e.GET("/lib", library)
}

func library(c echo.Context) error {
	err := lib.Process("testlib")
	if err != nil {
		panic(err)
	}
	return c.String(http.StatusOK, "all good")
}
