package job

import (
	"context"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/repo"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/google/uuid"
)

var (
	_ usecase.BackgroundQueue = (*JobQueue)(nil)
)

type (
	JobQueue struct {
		jobRepo *repo.JobRepo
	}
)

func NewJobQueue(jobRepo *repo.JobRepo) *JobQueue {
	return &JobQueue{
		jobRepo: jobRepo,
	}
}

func (b *JobQueue) Enqueue(
	ctx context.Context,
	entityID uuid.UUID,
	jobType entity.JobType,
) error {
	job, err := entity.NewJob(entityID, jobType)
	if err != nil {
		return err
	}

	return b.jobRepo.Create(ctx, job)
}
