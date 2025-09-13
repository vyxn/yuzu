package library

import (
	"time"

	"github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/pkg/yerr"

	"github.com/google/uuid"
)

type JobRunStatus string

const (
	Waiting  JobRunStatus = "waiting"
	Running  JobRunStatus = "running"
	Finished JobRunStatus = "finished"
	Canceled JobRunStatus = "canceled"
	Crashed  JobRunStatus = "crashed"
	Timedout JobRunStatus = "timedout"
)

type JobRunSaver interface {
	Save(*JobRun) error
}

type JobRun struct {
	ID           uuid.UUID
	Status       JobRunStatus
	StartedAt    time.Time
	EndedAt      time.Time
	ProvidersIDs []string
	LibraryID    string
	Errors       error
}

func NewJobRun(job *Job) (*JobRun, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil,
			yerr.WithStackf("generating uuid for new job %w", err)
	}

	providerIds := []string{}
	for _, p := range job.Providers {
		providerIds = append(providerIds, p.ID)
	}

	return &JobRun{
		ID:           id,
		Status:       Waiting,
		StartedAt:    time.Now(),
		ProvidersIDs: providerIds,
		LibraryID:    job.library.Id,
	}, nil
}

func (j *JobRun) Elapsed() time.Duration {
	switch j.Status {
	case Waiting, Running:
		return time.Since(j.StartedAt)
	case Finished, Canceled, Crashed, Timedout:
		return j.EndedAt.Sub(j.StartedAt)

	default:
		assert.Assert(false, "invalid JobRun status")
		return -1
	}
}
