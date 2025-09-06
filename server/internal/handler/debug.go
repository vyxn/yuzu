package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/vyxn/yuzu/internal/pkg/yerr"

	"github.com/AsaiYusuke/jsonpath/v2"
	"github.com/labstack/echo/v4"
)

func registerDebug(e *echo.Echo) {
	e.GET("/", hello)
	e.POST("/jsonPath", jsonPath)
}

func hello(c echo.Context) error {
	e := c.QueryParam("e")
	if e != "" {
		return echo.NewHTTPError(http.StatusBadRequest, "error on hello").
			SetInternal(fmt.Errorf("wrapped with something: %w", yerr.WithStackf("test yerr")))
	}
	return c.String(http.StatusOK, "Hello, World!")
}

func jsonPath(c echo.Context) error {
	p := c.QueryParam("path")

	var jsonValue any
	if err := c.Bind(&jsonValue); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "parsing body").
			SetInternal(yerr.WithStackf("unmarshalling body: %w", err))
	}

	slog.Warn("/jsonPath", slog.String("path", p), slog.Any("json", "m"))
	out, err := jsonpath.Retrieve(p, jsonValue)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "retrieving jsonpath").
			SetInternal(yerr.WithStackf("retrieving jsonpath: %s %w", out, err))
	}

	return c.JSON(http.StatusOK, out)
}
