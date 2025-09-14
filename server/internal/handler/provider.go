package handler

import (
	"net/http"

	"github.com/kaptinlin/jsonschema"
	"github.com/vyxn/yuzu/internal/provider"
	"github.com/vyxn/yuzu/internal/repository"

	"github.com/labstack/echo/v4"
)

func registerProvider(
	e *echo.Echo,
	r repository.Repository[provider.Provider, string],
) {
	h := NewProviderHandler(r)

	e.GET("/providers", h.getProviders)
	e.GET("/providers/:id", h.getProvider)
	e.PUT("/providers/:id", h.putProvider)
	e.DELETE("/providers/:id", h.deleteProvider)
	// e.GET("/providers/:id/run", h.getProviderRun)
	e.GET("/schemas/providers/http", h.getProviderSchema)
}

type ProviderHandler struct {
	r repository.Repository[provider.Provider, string]
}

func NewProviderHandler(
	r repository.Repository[provider.Provider, string],
) *ProviderHandler {
	return &ProviderHandler{r: r}
}

func (h *ProviderHandler) getProviders(c echo.Context) error {
	all, err := h.r.GetAll()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "getting providers").
			SetInternal(err)
	}
	return c.JSON(http.StatusOK, all)
}

func (h *ProviderHandler) getProvider(c echo.Context) error {
	id := c.Param("id")

	prov, err := h.r.Get(id)
	if err != nil {
		return echo.ErrNotFound.SetInternal(err)
	}

	return c.JSON(http.StatusOK, prov)
}

func (h *ProviderHandler) putProvider(c echo.Context) error {
	id := c.Param("id")

	p, err := provider.NewProvider(id, c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "parsing provider").
			SetInternal(err)
	}

	if err := h.r.Save(p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "saving provider").
			SetInternal(err)
	}

	return c.JSON(http.StatusOK, p)
}

func (h *ProviderHandler) deleteProvider(c echo.Context) error {
	id := c.Param("id")

	if err := h.r.Delete(id); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "deleting provider").
			SetInternal(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// func (h *ProviderHandler) getProviderRun(c echo.Context) error {
// 	id := c.Param("id")
// 	input := queryToMap(c.QueryParams(), ",")
//
// 	p, err := h.r.Get(id)
// 	if err != nil {
// 		return echo.ErrNotFound
// 	}
//
// 	data, err := p.Run(c.Request().Context(), input)
// 	if err != nil {
// 		return echo.NewHTTPError(http.StatusBadRequest, "error while running provider").
// 			SetInternal(err)
// 	}
//
// 	return c.Blob(http.StatusOK, p.MimeType(), data)
// }

func (h *ProviderHandler) getProviderSchema(c echo.Context) error {
	schema := jsonschema.FromStruct[provider.HTTPProvider]()
	return c.JSON(http.StatusOK, schema)
}
