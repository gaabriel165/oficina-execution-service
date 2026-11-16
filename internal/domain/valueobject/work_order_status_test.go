package valueobject_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
)

func TestNewWorkOrderStatus_ShouldAcceptKnownStatus(t *testing.T) {
	status, err := valueobject.NewWorkOrderStatus("in_diagnosis")

	assert.NoError(t, err)
	assert.Equal(t, valueobject.WorkOrderStatusInDiagnosis, status)
	assert.Equal(t, "in_diagnosis", status.String())
}

func TestNewWorkOrderStatus_ShouldRejectUnknownStatus(t *testing.T) {
	_, err := valueobject.NewWorkOrderStatus("paused")

	assert.ErrorIs(t, err, valueobject.ErrInvalidWorkOrderStatus)
}

func TestCanTransitionTo_ShouldFollowExecutionFlow(t *testing.T) {
	assert.True(t, valueobject.WorkOrderStatusAwaitingDiagnosis.CanTransitionTo(valueobject.WorkOrderStatusInDiagnosis))
	assert.True(t, valueobject.WorkOrderStatusInDiagnosis.CanTransitionTo(valueobject.WorkOrderStatusDiagnosed))
	assert.True(t, valueobject.WorkOrderStatusDiagnosed.CanTransitionTo(valueobject.WorkOrderStatusInExecution))
	assert.True(t, valueobject.WorkOrderStatusInExecution.CanTransitionTo(valueobject.WorkOrderStatusFinished))
}

func TestCanTransitionTo_ShouldAllowCancellationBeforeFinish(t *testing.T) {
	assert.True(t, valueobject.WorkOrderStatusAwaitingDiagnosis.CanTransitionTo(valueobject.WorkOrderStatusCancelled))
	assert.True(t, valueobject.WorkOrderStatusInExecution.CanTransitionTo(valueobject.WorkOrderStatusCancelled))
}

func TestCanTransitionTo_ShouldRejectSkippingSteps(t *testing.T) {
	assert.False(t, valueobject.WorkOrderStatusAwaitingDiagnosis.CanTransitionTo(valueobject.WorkOrderStatusInExecution))
	assert.False(t, valueobject.WorkOrderStatusFinished.CanTransitionTo(valueobject.WorkOrderStatusCancelled))
	assert.False(t, valueobject.WorkOrderStatusCancelled.CanTransitionTo(valueobject.WorkOrderStatusInDiagnosis))
}
