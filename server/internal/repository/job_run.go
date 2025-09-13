package repository

import (
	"maps"
	"slices"

	"github.com/vyxn/yuzu/internal/library"
	"github.com/vyxn/yuzu/internal/pkg/yerr"

	"github.com/google/uuid"
)

type InMemoryJobRunRepository struct {
	jobs map[uuid.UUID]*library.JobRun
}

func NewJobRunRepository() Repository[*library.JobRun, uuid.UUID] {
	return &InMemoryJobRunRepository{
		jobs: make(map[uuid.UUID]*library.JobRun),
	}
}

func (j *InMemoryJobRunRepository) GetAll() ([]*library.JobRun, error) {
	return slices.Collect(maps.Values(j.jobs)), nil
}

func (j *InMemoryJobRunRepository) Get(id uuid.UUID) (*library.JobRun, error) {
	job, ok := j.jobs[id]
	if !ok {
		return nil, yerr.WithStackf("jobRun %q not found", id)
	}

	return job, nil
}

func (j *InMemoryJobRunRepository) Save(job *library.JobRun) error {
	j.jobs[job.ID] = job
	return nil
}

func (j *InMemoryJobRunRepository) Delete(id uuid.UUID) error {
	if _, ok := j.jobs[id]; !ok {
		return yerr.WithStackf("removing inexistent jobRun %q", id)
	}

	delete(j.jobs, id)
	return nil
}
