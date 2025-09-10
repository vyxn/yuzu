package library

import (
	"log/slog"
	"time"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
	"github.com/vyxn/yuzu/internal/repository"

	"github.com/robfig/cron/v3"
)

type Job struct {
	Schedule  string
	schedule  cron.Schedule
	Providers []JobProvider
	library   *Library
}

func NewJob(j *repository.Job) (*Job, error) {
	jps := []JobProvider{}
	for _, jp := range j.Providers {
		jps = append(jps, NewJobProvider(jp))
	}

	job := &Job{
		Schedule:  j.Schedule,
		Providers: jps,
	}

	if j.Schedule != "" {
		sch, err := cron.ParseStandard(j.Schedule)
		if err != nil {
			return nil, yerr.WithStackf("invalid schedule %q: %w", j.Schedule, err)
		}

		job.schedule = sch
	}

	return job, nil
}

type JobProvider struct {
	ID     string
	Inputs map[string]string
}

func NewJobProvider(jp repository.JobProvider) JobProvider {
	return JobProvider{ID: jp.ID, Inputs: jp.Inputs}
}

// Next returns the next time this job would be called after the provided time
func (j *Job) Next(t time.Time) time.Time {
	return j.schedule.Next(t)
}

func (j *Job) Run() {
	// TODO: fill this function
	println("job executed")
}
