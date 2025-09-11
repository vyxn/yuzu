package library

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Job struct {
	Schedule  string        `json:"schedule"`
	schedule  cron.Schedule `json:"-"`
	Providers []JobProvider `json:"providers"`
	library   *Library      `json:"-"`
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
	slog.Info("executing job", slog.String("library", j.library.Id))

	selections := j.library.Select()
	wg := sync.WaitGroup{}
	for _, provider := range j.Providers {
		prov, err := j.library.providerFinder.Get(provider.ID)
		if err != nil {
			slog.Error(
				"running job",
				slog.String("library", j.library.Id),
				slog.Any("error", err),
			)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			for _, selection := range selections {
				if selection.IsDir {
					continue
				}

				inputs := map[string]string{}
				for k, pi := range provider.Inputs {
					v, ok := selection.Env[pi]
					if !ok {
						continue
					}

					inputs[k] = v
				}

				output, err := prov.Run(inputs)
				if err != nil {
					slog.Error(
						"running job",
						slog.String("library", j.library.Id),
						slog.Any("error", err),
					)
				}

				slog.Info("job output", slog.String("output", string(output)))
			}

			slog.Info("job finished", slog.String("library", j.library.Id))
		}()
	}
	wg.Wait()
}
