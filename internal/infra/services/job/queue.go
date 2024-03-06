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
		jobRepo     *repo.JobRepo
		pollTrigger func()
	}
)

func NewJobQueue(
	jobRepo *repo.JobRepo,
	trigger func()) *JobQueue {
	return &JobQueue{
		jobRepo:     jobRepo,
		pollTrigger: trigger,
	}
}

func (q *JobQueue) Enqueue(
	ctx context.Context,
	entityID uuid.UUID,
	jobType entity.JobType,
) error {
	job, err := entity.NewJob(entityID, jobType)
	if err != nil {
		return err
	}

	err = q.jobRepo.Create(ctx, job)
	if err != nil {
		return err
	}

	q.pollTrigger()

	return nil
}
