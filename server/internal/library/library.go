// Package library contains all functionality to manage the libraries
package library

import (
	"io/fs"
	"log/slog"
	"path/filepath"

	"github.com/vyxn/yuzu/internal/repository"

	"github.com/robfig/cron/v3"
)

type Library struct {
	Path      string
	Selectors Selectors
	Jobs      []*Job
}

func FromRepo(l *repository.Library) (*Library, error) {
	selectors, err := NewSelectors(l.Selectors)
	if err != nil {
		return nil, err
	}

	jobs := []*Job{}
	for _, j := range l.Jobs {
		job, err := NewJob(j)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	lib := &Library{
		Path:      l.Path,
		Selectors: selectors,
		Jobs:      jobs,
	}
	for i := range jobs {
		jobs[i].library = lib
	}

	return lib, nil
}

func (l *Library) Select() []*Selection {
	res := []*Selection{}

	filepath.WalkDir(l.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			slog.Info("walking", slog.String("path", path))
		}

		if sel := l.Selectors.run(d.IsDir(), path); sel != nil {
			res = append(res, sel)
		}
		return nil
	})

	return res
}

func (l *Library) AllJobs() []*Job {
	return l.Jobs
}

func (l *Library) ScheduleJobs(c *cron.Cron) {
	for _, j := range l.Jobs {
		c.Schedule(j.schedule, j)
	}
}
