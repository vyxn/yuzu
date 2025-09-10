// Package handler contains all http endpoint handler functions
package handler

import (
	"github.com/vyxn/yuzu/internal/repository"

	"github.com/labstack/echo/v4"
)

func SetupRoutes(e *echo.Echo,
	r repository.Repository[*repository.Library, string],
) {
	registerStatic(e)
	registerDebug(e)
	registerProvider(e)
	registerLibrary(e, r)
}
