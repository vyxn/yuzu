package internal

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/vyxn/yuzu/internal/lib"
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
	e.GET("/providers/:id/run", hComicInfo)
	e.GET("/lib", hLib)
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

func hComicInfo(c echo.Context) error {
	series := c.QueryParam("s")
	chapter := c.QueryParam("c")
	prov := c.Param("id")

	if p, ok := provider.Providers[prov]; ok {
		data, err := p.Run(map[string]string{
			"series":  series,
			"chapter": chapter,
		})
		if err != nil {
			return echo.ErrBadRequest.SetInternal(err)
		}
		return c.Blob(http.StatusOK, p.MimeType(), data)
	}
	return nil
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
	j, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	var jsonValue any
	err = json.Unmarshal(j, &jsonValue)
	if err != nil {
		return yerr.WithStackf(
			"unmarshalling: %w",
			err,
		)
	}

	slog.Warn("/jsonPath", slog.String("path", p), slog.Any("json", "m"))
	out, err := jsonpath.Retrieve(p, jsonValue)
	if err != nil {
		println(err.Error())
		return yerr.WithStackf("retrieving jsonpath: %s %w", out, err)
	}

	return c.JSON(http.StatusOK, out)
}
