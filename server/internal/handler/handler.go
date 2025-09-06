// Package handler contains all http endpoint handler functions
package handler

import "github.com/labstack/echo/v4"

func SetupRoutes(e *echo.Echo) {
	registerStatic(e)
	registerDebug(e)
	registerProvider(e)
	registerLibrary(e)
}
