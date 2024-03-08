package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/entity/apperrors"
)

var (
	ErrJobExists      = apperrors.NewInvalid("job already exists")
	ErrJobNotFound    = apperrors.NewNotFound("job not found")
	ErrJobTypeInvalid = apperrors.NewInvalid("invalid job type")

	delays = []time.Duration{
		1 * time.Second,
		1 * time.Second,
		1 * time.Second,
		2 * time.Second,
		2 * time.Second,
		2 * time.Second,
		3 * time.Second,
		5 * time.Second,
		8 * time.Second,
		8 * time.Second,
		13 * time.Second,
		13 * time.Second,
		13 * time.Second,
		21 * time.Second,
		34 * time.Second,
	}
)

type (
	JobUseCase struct {
		repo         JobRepo
		orderUseCase *OrderUseCase
		tx           trm.Manager
	}
	JobRepo interface {
		Create(ctx context.Context, job entity.Job) error
		GetUpdate(ctx context.Context, count uint) ([]entity.Job, error)
		Update(ctx context.Context, job entity.Job) error
		FixProcessingTimeout(ctx context.Context, timeout time.Duration) error
	}
)

func NewJobUseCase(
	jobRepo JobRepo,
	orderUseCase *OrderUseCase,
	trm trm.Manager,
) *JobUseCase {
	return &JobUseCase{
		repo:         jobRepo,
		orderUseCase: orderUseCase,
		tx:           trm,
	}
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

func (uc *JobUseCase) FixProcessingTimeout(ctx context.Context, procTimeout time.Duration) error {
	return uc.repo.FixProcessingTimeout(ctx, procTimeout)
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
		order, err := uc.orderUseCase.UpdateAccrual(ctx, entity.OrderID(job.EntityID))
		switch {
		case errors.Is(err, entity.ErrOrderStatusFinal):
			return uc.fail(ctx, job, entity.ErrOrderStatusFinal)
		case errors.Is(err, ErrOrderNotFound):
			return uc.fail(ctx, job, ErrOrderNotFound)
		case err != nil:
			return err
		default:
		}

		switch {
		case order.Status.Final():
			job.Status = entity.JobStatusProcessed
			job.NextAttemptAt = time.Time{}
			return uc.repo.Update(ctx, job)
		default:
			next := getNextAttemptAt(job.Attempt)
			job.Status = entity.JobStatusProcessing
			job.NextAttemptAt = next
			return uc.repo.Update(ctx, job)
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
