package reactive

import "errors"

var (
	// ErrFlowClosed is returned when trying to operate on a closed flow
	ErrFlowClosed = errors.New("flow is closed")
	// ErrInvalidRequest is returned when requesting invalid number of items
	ErrInvalidRequest = errors.New("invalid request: must be > 0")
	// ErrSubscriptionCancelled is returned when subscription is cancelled
	ErrSubscriptionCancelled = errors.New("subscription cancelled")
	// ErrNoSubscribers is returned when there are no subscribers
	ErrNoSubscribers = errors.New("no subscribers")
)
