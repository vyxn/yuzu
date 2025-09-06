package internal

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/vyxn/yuzu/internal/config"
	"github.com/vyxn/yuzu/internal/lib"
	"github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
	"github.com/vyxn/yuzu/internal/provider"

	"github.com/AsaiYusuke/jsonpath/v2"
	"github.com/labstack/echo/v4"
)

//go:embed static/favicon.ico
var favicon []byte

func SetupRoutes(e *echo.Echo) {
	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "image/x-icon", favicon)
	})

	e.GET("/", hello)
	e.GET("/lib", hLib)
	e.GET("/providers/:id", hProvider)
	e.PUT("/providers/:id", hProviderPut)
	e.GET("/providers/:id/run", hProviderRun)

	// Debug
	e.POST("/jsonPath", hJsonPath)
}

// Handler
func hello(c echo.Context) error {
	e := c.QueryParam("e")
	if e != "" {
		return echo.NewHTTPError(http.StatusBadRequest, "error on hello").
			SetInternal(fmt.Errorf("wrapped with something: %w", yerr.WithStackf("test yerr")))
	}
	return c.String(http.StatusOK, "Hello, World!")
}

func queryToMap(queryParams url.Values, sep string) map[string]string {
	m := make(map[string]string)
	for k, v := range queryParams {
		m[k] = strings.Join(v, sep)
	}
	return m
}

func hProviderRun(c echo.Context) error {
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

func hLib(c echo.Context) error {
	err := lib.Process("testlib")
	if err != nil {
		panic(err)
	}
	return c.String(http.StatusOK, "all good")
}

func hJsonPath(c echo.Context) error {
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

func hProvider(c echo.Context) error {
	id := c.Param("id")

	if p, ok := config.Cfg.Providers.Load(id); ok {
		return c.JSON(http.StatusOK, p)
	}

	return echo.ErrNotFound
}

func hProviderPut(c echo.Context) error {
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
