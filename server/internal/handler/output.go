package handler

import (
	"net/http"

	"github.com/vyxn/yuzu/internal/output"
	"github.com/vyxn/yuzu/internal/repository"

	"github.com/labstack/echo/v4"
)

func registerOutput(
	e *echo.Echo,
	r repository.Repository[output.Output, string],
) {
	h := NewOutputHandler(r)

	e.GET("/outputs", h.getOutputs)
	e.GET("/outputs/:id", h.getOutput)
	e.PUT("/outputs/:id", h.putOutput)
	e.DELETE("/outputs/:id", h.deleteOutput)
}

type OutputHandler struct {
	r repository.Repository[output.Output, string]
}

func NewOutputHandler(
	r repository.Repository[output.Output, string],
) *OutputHandler {
	return &OutputHandler{r: r}
}

func (h *OutputHandler) getOutputs(c echo.Context) error {
	all, err := h.r.GetAll()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "getting outputs").
			SetInternal(err)
	}
	return c.JSON(http.StatusOK, all)
}

func (h *OutputHandler) getOutput(c echo.Context) error {
	id := c.Param("id")

	out, err := h.r.Get(id)
	if err != nil {
		return echo.ErrNotFound.SetInternal(err)
	}

	return c.JSON(http.StatusOK, out)
}

func (h *OutputHandler) putOutput(c echo.Context) error {
	id := c.Param("id")

	out, err := output.NewOutput(id, c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "parsing output").
			SetInternal(err)
	}

	if err := h.r.Save(out); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "saving output").
			SetInternal(err)
	}

	return c.JSON(http.StatusOK, out)
}

func (h *OutputHandler) deleteOutput(c echo.Context) error {
	id := c.Param("id")

	if err := h.r.Delete(id); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "deleting output").
			SetInternal(err)
	}

	return c.NoContent(http.StatusNoContent)
}
