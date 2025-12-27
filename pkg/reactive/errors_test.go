package reactive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	assert.NotNil(t, ErrFlowClosed)
	assert.NotNil(t, ErrInvalidRequest)
	assert.NotNil(t, ErrSubscriptionCancelled)
	assert.NotNil(t, ErrNoSubscribers)
	assert.NotNil(t, ErrBackpressureExceeded)
}

func TestErrFlowClosed(t *testing.T) {
	assert.Equal(t, "flow is closed", ErrFlowClosed.Error())
}

func TestErrInvalidRequest(t *testing.T) {
	assert.Equal(t, "invalid request: must be > 0", ErrInvalidRequest.Error())
}

func TestErrSubscriptionCancelled(t *testing.T) {
	assert.Equal(t, "subscription cancelled", ErrSubscriptionCancelled.Error())
}

func TestErrNoSubscribers(t *testing.T) {
	assert.Equal(t, "no subscribers", ErrNoSubscribers.Error())
}
