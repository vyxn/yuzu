package handler

import (
	"encoding/json"
	"net/http"

	"github.com/vyxn/yuzu/internal/config"
	"github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
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
	p := provider.RawProvider{}

	// Bind request body to struct
	if err := c.Bind(&p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "parsing body content").
			SetInternal(yerr.WithStackf("unmarshaling provider JSON: %v", err))
	}

	switch p.Type {
	case "http":
		hp := provider.HTTPProvider{}
		if err := json.Unmarshal(p.Raw, &hp); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "parsing body content of type http").
				SetInternal(yerr.WithStackf("unmarshaling provider JSON: %v", err))
		}
		hp.ID = id
		config.StoreProvider(id, &hp)
		return c.JSON(http.StatusOK, hp)
	case "cli":
		return c.JSON(http.StatusOK, p)
	default:
		return yerr.WithStackf("type %s is not supported", p.Type)
	}
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
