package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/vyxn/yuzu/internal/library"
	"github.com/vyxn/yuzu/internal/repository"

	"github.com/labstack/echo/v4"
)

func registerJobRun(
	e *echo.Echo,
	r repository.Repository[*library.JobRun, uuid.UUID],
) {
	h := NewJobRunHandler(r)

	e.GET("/jobs", h.getJobRuns)
	e.GET("/jobs/:id", h.getJobRun)
}

type JobRunHandler struct {
	r repository.Repository[*library.JobRun, uuid.UUID]
}

func NewJobRunHandler(
	r repository.Repository[*library.JobRun, uuid.UUID],
) *JobRunHandler {
	return &JobRunHandler{r: r}
}

func (h *JobRunHandler) getJobRuns(c echo.Context) error {
	all, err := h.r.GetAll()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "getting jobRuns").
			SetInternal(err)
	}
	return c.JSON(http.StatusOK, all)
}

func (h *JobRunHandler) getJobRun(c echo.Context) error {
	id := c.Param("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return echo.ErrNotFound.SetInternal(err)
	}

	j, err := h.r.Get(uuid)
	if err != nil {
		return echo.ErrNotFound.SetInternal(err)
	}
	return c.JSON(http.StatusOK, j)
}
