package handler

import (
	"net/http"

	"github.com/vyxn/yuzu/internal/config"
	"github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/provider"

	"github.com/labstack/echo/v4"
)

func registerProvider(e *echo.Echo) {
	e.GET("/providers/:id", getProvider)
	e.PUT("/providers/:id", putProvider)
	e.GET("/providers/:id/run", getProviderRun)
}

func getProvider(c echo.Context) error {
	id := c.Param("id")

	if p, ok := config.Cfg.Providers.Load(id); ok {
		return c.JSON(http.StatusOK, p)
	}

	return echo.ErrNotFound
}

func putProvider(c echo.Context) error {
	id := c.Param("id")

	p, err := provider.New(id, c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "parsing provider").
			SetInternal(err)
	}

	config.StoreProvider(id, p)
	return c.JSON(http.StatusOK, p)
}

func getProviderRun(c echo.Context) error {
	id := c.Param("id")
	input := queryToMap(c.QueryParams(), ",")

	pr, ok := config.Cfg.Providers.Load(id)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "provider not found")
	}

	p, ok := pr.(*provider.HTTPProvider)
	assert.Assert(ok, "found unexpected type in providers map")

	data, err := p.Run(input)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error while running provider").
			SetInternal(err)
	}

	return c.Blob(http.StatusOK, p.MimeType(), data)
}
