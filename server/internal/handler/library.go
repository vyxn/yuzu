package handler

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/vyxn/yuzu/internal/library"

	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal/config"
)

func registerLibrary(e *echo.Echo) {
	e.GET("/lib", lib)
	e.GET("/libraries", getLibraries)
	e.GET("/libraries/:id", getLibrary)
	e.GET("/libraries/:id/select", getLibrarySelect)
	e.GET("/libraries/:id/jobs", getLibraryJobs)
}

func getLibraries(c echo.Context) error {
	ids := []string{}
	config.Cfg.Libraries.Range(func(key any, value any) bool {
		ids = append(ids, key.(string))
		return true
	})

	slices.Sort(ids)
	return c.JSON(http.StatusOK, ids)
}

func getLibrary(c echo.Context) error {
	id := c.Param("id")

	if l, ok := config.Cfg.Libraries.Load(id); ok {
		return c.JSON(http.StatusOK, l)
	}

	return echo.ErrNotFound
}

func getLibrarySelect(c echo.Context) error {
	id := c.Param("id")

	if l, ok := config.Cfg.Libraries.Load(id); ok {
		lib, ok := l.(*library.Library)
		if !ok {
			return echo.ErrNotFound
		}

		res := lib.Select()

		return c.JSON(http.StatusOK, res)
	}

	return echo.ErrNotFound
}

func getLibraryJobs(c echo.Context) error {
	id := c.Param("id")

	l, ok := config.Cfg.Libraries.Load(id)
	if !ok {
		return echo.ErrNotFound
	}

	lib, ok := l.(*library.Library)
	if !ok {
		return echo.ErrNotFound
	}

	res := []map[string]string{}
	now := time.Now()
	for _, j := range lib.AllJobs() {
		next := j.Next(now)
		providers := []string{}
		for _, p := range j.Providers {
			providers = append(providers, p.ID)
		}

		res = append(res, map[string]string{
			"providers":     strings.Join(providers, ","),
			"scheduled for": next.Local().String(),
		})
	}

	return c.JSON(http.StatusOK, res)

}

func lib(c echo.Context) error {
	// err := lib.Process("testlib")
	// if err != nil {
	// 	panic(err)
	// }
	return c.String(http.StatusOK, "all good")
}
