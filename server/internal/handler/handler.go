// Package handler contains all http endpoint handler functions
package handler

import (
	"github.com/vyxn/yuzu/internal/library"
	"github.com/vyxn/yuzu/internal/provider"
	"github.com/vyxn/yuzu/internal/repository"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func SetupRoutes(e *echo.Echo,
	r repository.Repository[*library.Library, string],
	pr repository.Repository[provider.Provider, string],
	jrr repository.Repository[*library.JobRun, uuid.UUID],
) {
	registerStatic(e)
	registerDebug(e)
	registerProvider(e, pr)
	registerLibrary(e, r, pr, jrr)
	registerJobRun(e, jrr)

}
