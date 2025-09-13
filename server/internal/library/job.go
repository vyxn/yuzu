package library

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/vyxn/yuzu/internal/provider"
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
	ctx := context.Background()
	start := time.Now()
	slog.Info("executing job", slog.String("library", j.library.Id))

	providers := map[string]provider.Provider{}
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

		providers[provider.ID] = prov
	}

	selections := j.library.Select()
	wg := sync.WaitGroup{}
	for _, selection := range selections {
		if selection.IsDir {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			selectionWg := sync.WaitGroup{}
			for _, provider := range j.Providers {
				prov, ok := providers[provider.ID]
				if !ok {
					continue
				}

				selectionWg.Add(1)
				go func() {
					defer selectionWg.Done()

					inputs := map[string]string{}
					for k, pi := range provider.Inputs {
						v, ok := selection.Env[pi]
						if !ok {
							continue
						}

						inputs[k] = v
					}

					output, err := prov.Run(ctx, inputs)
					if err != nil {
						slog.Error(
							"running job",
							slog.String("library", j.library.Id),
							slog.Any("error", err),
						)
					}

					slog.Info("job output", slog.String("output", string(output)))
				}()
			}

			selectionWg.Wait()

			// TODO: handle the outputs

		}()
	}

	wg.Wait()

	for _, p := range providers {
		c := p.(*provider.HTTPProvider).Cache.Stats()
		slog.Info(
			"cache usage",
			slog.Uint64("hits", c.Hits),
			slog.Uint64("misses", c.Misses),
			slog.Float64("hit ration", c.HitRatio()),
		)
	}

	elapsed := time.Since(start)
	slog.Info(
		"job finished",
		slog.String("library", j.library.Id),
		slog.String("took", elapsed.String()),
	)
}
