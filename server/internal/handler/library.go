package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vyxn/yuzu/internal/library"
	"github.com/vyxn/yuzu/internal/provider"
	"github.com/vyxn/yuzu/internal/repository"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func registerLibrary(
	e *echo.Echo,
	r repository.Repository[*library.Library, string],
	pr repository.Repository[provider.Provider, string],
	jrr repository.Repository[*library.JobRun, uuid.UUID],
) {
	h := NewLibraryHandler(r, pr, jrr)

	// e.GET("/lib", lib)
	e.GET("/libraries", h.getLibraries)
	e.GET("/libraries/:id", h.getLibrary)
	e.PUT("/libraries/:id", h.putLibrary)
	e.DELETE("/libraries/:id", h.deleteLibrary)
	e.GET("/libraries/:id/select", h.getLibrarySelect)
	e.GET("/libraries/:id/jobs", h.getLibraryJobs)
	e.GET("/libraries/:id/jobs/:job", h.getLibraryJob)
	e.GET("/libraries/:id/jobs/:job/run", h.getRunJob)
}

type LibraryHandler struct {
	r   repository.Repository[*library.Library, string]
	pr  repository.Repository[provider.Provider, string]
	jrr repository.Repository[*library.JobRun, uuid.UUID]
}

func NewLibraryHandler(
	r repository.Repository[*library.Library, string],
	pr repository.Repository[provider.Provider, string],
	jrr repository.Repository[*library.JobRun, uuid.UUID],
) *LibraryHandler {
	return &LibraryHandler{r: r, pr: pr, jrr: jrr}
}

func (h *LibraryHandler) getLibraries(c echo.Context) error {
	all, err := h.r.GetAll()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "getting libraries").
			SetInternal(err)
	}
	return c.JSON(http.StatusOK, all)
}

func (h *LibraryHandler) getLibrary(c echo.Context) error {
	id := c.Param("id")

	lib, err := h.r.Get(id)
	if err != nil {
		return echo.ErrNotFound.SetInternal(err)
	}

	return c.JSON(http.StatusOK, lib)
}

func (h *LibraryHandler) putLibrary(c echo.Context) error {
	id := c.Param("id")

	lib, err := library.NewLibrary(id, c.Request().Body, h.pr, h.jrr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "parsing library").
			SetInternal(err)
	}

	if err := h.r.Save(lib); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "saving library").
			SetInternal(err)
	}

	return c.JSON(http.StatusOK, lib)
}

func (h *LibraryHandler) deleteLibrary(c echo.Context) error {
	id := c.Param("id")

	if err := h.r.Delete(id); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "deleting library").
			SetInternal(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *LibraryHandler) getLibrarySelect(c echo.Context) error {
	id := c.Param("id")

	lib, err := h.r.Get(id)
	if err != nil {
		return echo.ErrNotFound
	}

	return c.JSON(http.StatusOK, lib.Select())
}

func (h *LibraryHandler) getLibraryJobs(c echo.Context) error {
	id := c.Param("id")

	lib, err := h.r.Get(id)
	if err != nil {
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

func (h *LibraryHandler) getLibraryJob(c echo.Context) error {
	id := c.Param("id")
	job, err := strconv.Atoi(c.Param("job"))
	if err != nil {
		return echo.ErrBadRequest.SetInternal(err)
	}

	lib, err := h.r.Get(id)
	if err != nil {
		return echo.ErrNotFound
	}

	jobs := lib.AllJobs()
	if len(jobs) <= job {
		return echo.ErrBadRequest
	}

	return c.JSON(http.StatusOK, jobs[job])
}

func (h *LibraryHandler) getRunJob(c echo.Context) error {
	id := c.Param("id")
	j, err := strconv.Atoi(c.Param("job"))
	if err != nil {
		return echo.ErrBadRequest.SetInternal(err)
	}

	lib, err := h.r.Get(id)
	if err != nil {
		return echo.ErrNotFound
	}

	jobs := lib.AllJobs()
	if len(jobs) <= j {
		return echo.ErrBadRequest
	}

	job := jobs[j]
	go job.Run()

	return c.JSON(http.StatusOK, "run job")
}

// func lib(c echo.Context) error {
// 	// err := lib.Process("testlib")
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	return c.String(http.StatusOK, "all good")
// }
