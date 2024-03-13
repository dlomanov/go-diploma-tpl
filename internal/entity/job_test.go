package entity_test

import (
	"testing"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewJob(t *testing.T) {
	entityID := uuid.New()
	jobType := entity.JobTypePollAccrual

	job, err := entity.NewJob(entityID, jobType)
	require.NoError(t, err, "error should be nil")
	require.NotNil(t, job, "job should not be nil")
	require.Equal(t, entityID, job.EntityID, "entityID should be equal")
	require.Equal(t, jobType, job.Type, "jobType should be equal")
	require.Equal(t, entity.JobStatusNew, job.Status, "status should be JobStatusNew")
	require.Zero(t, job.Attempt, "attempt should be zero")
	require.Nil(t, job.LastError, "lastError should be nil")
	require.NotZero(t, job.NextAttemptAt, "nextAttemptAt should not be zero")
	require.NotZero(t, job.CreatedAt, "createdAt should not be zero")
	require.NotZero(t, job.UpdatedAt, "updatedAt should not be zero")
}

func TestJobTypeValid(t *testing.T) {
	require.True(t, entity.JobTypePollAccrual.Valid(), "JobTypePollAccrual should be valid")
	require.False(t, entity.JobType("INVALID").Valid(), "INVALID job type should not be valid")
}
