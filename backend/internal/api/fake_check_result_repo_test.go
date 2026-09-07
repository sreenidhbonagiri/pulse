package api

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
	"github.com/sreenidhbonagiri/pulse/backend/internal/repository"
)

type fakeCheckResultRepo struct {
	results map[uuid.UUID]models.CheckResult
}

func newFakeCheckResultRepo() *fakeCheckResultRepo {
	return &fakeCheckResultRepo{
		results: make(map[uuid.UUID]models.CheckResult),
	}
}

func (f *fakeCheckResultRepo) Create(_ context.Context, result *models.CheckResult) error {
	if result.ID == uuid.Nil {
		result.ID = uuid.New()
	}
	if result.CheckedAt.IsZero() {
		result.CheckedAt = time.Now().UTC()
	}
	if result.JobID != uuid.Nil {
		for _, existing := range f.results {
			if existing.JobID == result.JobID {
				return repository.ErrDuplicate
			}
		}
	}
	f.results[result.ID] = *result
	return nil
}

func (f *fakeCheckResultRepo) GetByJobID(_ context.Context, jobID uuid.UUID) (*models.CheckResult, error) {
	for _, result := range f.results {
		if result.JobID == jobID {
			copied := result
			return &copied, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeCheckResultRepo) GetByID(_ context.Context, id uuid.UUID) (*models.CheckResult, error) {
	result, ok := f.results[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copied := result
	return &copied, nil
}

func (f *fakeCheckResultRepo) ListByMonitorID(_ context.Context, monitorID uuid.UUID, limit int) ([]models.CheckResult, error) {
	listed := make([]models.CheckResult, 0)
	for _, result := range f.results {
		if result.MonitorID == monitorID {
			listed = append(listed, result)
		}
	}

	sort.Slice(listed, func(i, j int) bool {
		return listed[i].CheckedAt.After(listed[j].CheckedAt)
	})

	if limit <= 0 {
		limit = 50
	}
	if limit > len(listed) {
		return listed, nil
	}
	return listed[:limit], nil
}
