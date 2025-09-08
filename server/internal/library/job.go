package library

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type Job struct {
	Schedule  string        `json:"schedule"`
	schedule  cron.Schedule `json:"-"`
	Providers []JobProvider `json:"providers"`
}

func (j *Job) UnmarshalJSON(data []byte) error {
	// shadow type to avoid recursion
	type Alias Job
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(j),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if j.Schedule != "" {
		sch, err := cron.ParseStandard(j.Schedule)
		if err != nil {
			return fmt.Errorf("invalid schedule %q: %w", j.Schedule, err)
		}

		j.schedule = sch
	}

	return nil
}

type JobProvider struct {
	ID     string            `json:"id"`
	Inputs map[string]string `json:"inputs"`
}

// Next returns the next time this job would be called after the provided time
func (j *Job) Next(t time.Time) time.Time {
	return j.schedule.Next(t)
}

func (j *Job) Run() {
	// TODO: fill this function
	println("job executed")
}
