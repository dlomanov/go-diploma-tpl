package entity

import (
	"errors"
	"github.com/google/uuid"
	"time"
)

var (
	ErrJobInvalidType = errors.New("invalid job type")
)

const (
	JobTypePollAccrual JobType = "task_type_poll_accrual"

	JobStatusNew        JobStatus = "NEW"
	JobStatusProcessing JobStatus = "PROCESSING"
	JobStatusFailed     JobStatus = "FAILED"
	JobStatusProcessed  JobStatus = "PROCESSED"
)

type (
	Job struct {
		ID            uuid.UUID
		Type          JobType
		Status        JobStatus
		EntityID      uuid.UUID
		Attempt       uint
		LastError     error
		NextAttemptAt time.Time
		CreatedAt     time.Time
		UpdatedAt     time.Time
	}
	JobType   string
	JobStatus string
)

func NewJob(
	entityID uuid.UUID,
	jobType JobType,
) (Job, error) {
	if !jobType.Valid() {
		return Job{}, ErrJobInvalidType
	}

	id, err := uuid.NewUUID()
	if err != nil {
		return Job{}, err
	}

	now := time.Now().UTC()
	return Job{
		ID:            id,
		Type:          jobType,
		Status:        JobStatusNew,
		EntityID:      entityID,
		Attempt:       0,
		LastError:     nil,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (t JobType) Valid() bool {
	return t == JobTypePollAccrual
}
