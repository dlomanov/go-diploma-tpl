package usecase

import (
	"context"
	"errors"
	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/google/uuid"
	"time"
)

var (
	ErrJobExists      = errors.New("job already exists")
	ErrJobNotFound    = errors.New("job not found")
	ErrJobTypeInvalid = errors.New("invalid job type")

	delays = []time.Duration{
		0 * time.Second,
		1 * time.Second,
		1 * time.Second,
		2 * time.Second,
		3 * time.Second,
		5 * time.Second,
		8 * time.Second,
		13 * time.Second,
		21 * time.Second,
		34 * time.Second,
	}
)

type (
	JobUseCase struct {
		repo         JobRepo
		orderUseCase OrderUseCase
		procTimeout  time.Duration
		tx           trm.Manager
	}
	JobRepo interface {
		Create(ctx context.Context, job entity.Job) error
		GetUpdate(ctx context.Context, count uint) ([]entity.Job, error)
		Update(ctx context.Context, job entity.Job) error
		FixProcessing(ctx context.Context, timeout time.Duration) error
	}
)

func (uc *JobUseCase) Enqueue(ctx context.Context, entityID uuid.UUID, jobType entity.JobType) error {
	job, err := entity.NewJob(entityID, jobType)
	if err != nil {
		return err
	}

	return uc.repo.Create(ctx, job)
}

func (uc *JobUseCase) Fetch(ctx context.Context, count uint) ([]entity.Job, error) {
	return uc.repo.GetUpdate(ctx, count)
}

func (uc *JobUseCase) Handle(ctx context.Context, job entity.Job) error {
	switch job.Type {
	case entity.JobTypePollAccrual:
		return uc.pollAccrual(ctx, job)
	default:
		return uc.Fail(ctx, job, ErrJobTypeInvalid)
	}
}

func (uc *JobUseCase) FixProcessing(ctx context.Context) error {
	return uc.repo.FixProcessing(ctx, uc.procTimeout)
}

func (uc *JobUseCase) Fail(
	ctx context.Context,
	job entity.Job,
	err error,
) error {
	return uc.fail(ctx, job, err)
}

func (uc *JobUseCase) pollAccrual(ctx context.Context, job entity.Job) error {
	return uc.tx.Do(ctx, func(ctx context.Context) error {
		event, err := uc.orderUseCase.UpdateAccrual(ctx, entity.OrderID(job.EntityID))
		switch {
		case errors.Is(err, entity.ErrOrderStatusFinal):
			return uc.fail(ctx, job, entity.ErrOrderStatusFinal)
		case errors.Is(err, ErrOrderNotFound):
			return uc.Fail(ctx, job, ErrOrderNotFound)
		case errors.Is(err, entity.ErrOrderEventInvalid):
			return uc.fail(ctx, job, entity.ErrOrderEventInvalid)
		case err != nil:
			return err
		default:
		}

		switch event {
		case entity.OrderEventUpdated:
			now := getNextAttemptAt(job.Attempt)
			job.Status = entity.JobStatusProcessing
			job.NextAttemptAt = now
			return uc.repo.Update(ctx, job)
		case entity.OrderEventCompleted:
			job.Status = entity.JobStatusProcessed
			job.NextAttemptAt = time.Time{}
			return uc.repo.Update(ctx, job)
		default:
			return uc.fail(ctx, job, entity.ErrOrderEventInvalid)
		}
	})

}

func getNextAttemptAt(attempt uint) time.Time {
	n := uint(len(delays))
	var d time.Duration
	switch {
	case attempt < n:
		d = delays[attempt]
	default:
		d = delays[n-1]
	}
	return time.Now().UTC().Add(d)
}

func (uc *JobUseCase) fail(
	ctx context.Context,
	job entity.Job,
	err error,
) error {
	job.Status = entity.JobStatusFailed
	job.LastError = err
	job.NextAttemptAt = time.Time{}
	return uc.repo.Update(ctx, job)
}