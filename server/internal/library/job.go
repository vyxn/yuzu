package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vyxn/yuzu/internal/provider"

	"github.com/robfig/cron/v3"
)

type Job struct {
	Schedule  string        `json:"schedule"`
	schedule  cron.Schedule `json:"-"`
	Output    *RawJobOutput `json:"output"`
	output    JobOutput     `json:"-"`
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
	slog.Info("executing job", slog.String("library", j.library.Id))
	jobRun, err := NewJobRun(j)
	if err != nil {
		slog.Error("", slog.Any("error", err))
		return
	}

	err = j.library.jobRunSaver.Save(jobRun)
	if err != nil {
		slog.Error("", slog.Any("error", err))
		return
	}

	var jobRunError error
	var jobRunFinalStatus = Crashed
	defer func() {
		jobRun.EndedAt = time.Now()
		jobRun.Status = jobRunFinalStatus
		jobRun.Errors = jobRunError
		slog.Info("jobRun finished",
			slog.Any("jobRun", jobRun),
			slog.String("library", jobRun.LibraryID),
			slog.String("took", jobRun.Elapsed().String()),
		)
	}()

	jobRun.Status = Running

	providers := map[string]provider.Provider{}
	for _, provider := range j.Providers {
		prov, err := j.library.providerFinder.Get(provider.ID)
		if err != nil {
			slog.Error(
				"running job",
				slog.String("library", j.library.Id),
				slog.Any("error", err),
			)
			jobRunError = errors.Join(jobRunError, err)
			jobRunFinalStatus = Crashed
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

			if err := j.handleSelected(ctx, providers, selection); err != nil {
				slog.Error("for selected", slog.Any("error", err))
				jobRunError = errors.Join(jobRunError, err)
				jobRunFinalStatus = Crashed
			}
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

	jobRunFinalStatus = Finished
}

func (j *Job) handleSelected(
	ctx context.Context,
	providers map[string]provider.Provider,
	selection *Selection,
) error {
	data, err := j.getProvidersData(ctx, providers, selection)
	if err != nil {
		slog.Error(
			"running providers",
			slog.String("job", j.library.Id),
			slog.Any("error", err),
		)
	}

	runEnv := provider.NewRunEnv(selection.Env, nil)

	// TODO: handle the outputs
	return j.output.Run(runEnv, data)
}

func (j *Job) getProvidersData(
	ctx context.Context,
	providers map[string]provider.Provider,
	selection *Selection,
) (any, error) {
	outs := make(chan any, len(j.Providers))
	errc := make(chan error, len(j.Providers))

	wg := sync.WaitGroup{}
	for _, provider := range j.Providers {
		prov, ok := providers[provider.ID]
		if !ok {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

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
				errc <- fmt.Errorf("provider %q: %w", prov.ID(), err)
				return
			}
			outs <- output

			// slog.Info(
			// 	"job output",
			// 	slog.String("output", string(output)),
			// )
		}()
	}

	wg.Wait()
	close(outs)
	close(errc)

	// TODO: merge outputs here
	var fout any
	for out := range outs {
		fout = out
		break
	}

	var errs error
	for err := range errc {
		errs = errors.Join(errs, err)
	}
	return fout, errs
}
