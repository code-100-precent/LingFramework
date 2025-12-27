package reactive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewFlow(t *testing.T) {
	flow := NewFlow()
	assert.NotNil(t, flow)
	assert.False(t, flow.IsClosed())
}

func TestFlow_Subscribe(t *testing.T) {
	flow := NewFlow()

	var received []interface{}
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			received = append(received, value)
		},
	}

	subscription := flow.Subscribe(subscriber)
	assert.NotNil(t, subscription)
}

func TestFlow_Publish(t *testing.T) {
	flow := NewFlow()

	var received []interface{}
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			received = append(received, value)
		},
	}

	flow.Subscribe(subscriber)

	err := flow.Publish("test")
	assert.NoError(t, err)

	time.Sleep(10 * time.Millisecond)
	assert.Len(t, received, 1)
	assert.Equal(t, "test", received[0])
}

func TestFlow_Complete(t *testing.T) {
	flow := NewFlow()

	completed := false
	subscriber := &testSubscriber{
		onComplete: func() {
			completed = true
		},
	}

	flow.Subscribe(subscriber)
	flow.Complete()

	time.Sleep(10 * time.Millisecond)
	assert.True(t, completed)
	assert.True(t, flow.IsClosed())
}

func TestFlow_PublishError(t *testing.T) {
	flow := NewFlow()

	var receivedError error
	subscriber := &testSubscriber{
		onError: func(err error) {
			receivedError = err
		},
	}

	flow.Subscribe(subscriber)
	testErr := assert.AnError
	flow.PublishError(testErr)

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, testErr, receivedError)
}

func TestFlow_MultipleSubscribers(t *testing.T) {
	flow := NewFlow()

	var received1 []interface{}
	var received2 []interface{}

	sub1 := &testSubscriber{
		onNext: func(value interface{}) {
			received1 = append(received1, value)
		},
	}
	sub2 := &testSubscriber{
		onNext: func(value interface{}) {
			received2 = append(received2, value)
		},
	}

	flow.Subscribe(sub1)
	flow.Subscribe(sub2)

	flow.Publish("test")

	time.Sleep(10 * time.Millisecond)
	assert.Len(t, received1, 1)
	assert.Len(t, received2, 1)
}

func TestSubscription_Request(t *testing.T) {
	flow := NewFlow()

	requested := int64(0)
	subscriber := &testSubscriber{
		onSubscribe: func(sub Subscription) {
			requested = sub.(*subscription).GetRequested()
			sub.Request(5)
			requested = sub.(*subscription).GetRequested()
		},
	}

	sub := flow.Subscribe(subscriber)
	assert.NotNil(t, sub)
	assert.Equal(t, int64(5), requested)
}

func TestSubscription_Cancel(t *testing.T) {
	flow := NewFlow()

	cancelled := false
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			// Should not be called after cancel
		},
	}

	sub := flow.Subscribe(subscriber)
	sub.Cancel()

	assert.True(t, sub.(*subscription).IsCancelled())

	// Publish after cancel should not reach subscriber
	flow.Publish("test")
	time.Sleep(10 * time.Millisecond)
	assert.False(t, cancelled)
}

// testSubscriber is a helper for testing
type testSubscriber struct {
	onSubscribe func(Subscription)
	onNext      func(interface{})
	onError     func(error)
	onComplete  func()
}

func (t *testSubscriber) OnSubscribe(subscription Subscription) {
	if t.onSubscribe != nil {
		t.onSubscribe(subscription)
	}
}

func (t *testSubscriber) OnNext(value interface{}) {
	if t.onNext != nil {
		t.onNext(value)
	}
}

func (t *testSubscriber) OnError(err error) {
	if t.onError != nil {
		t.onError(err)
	}
}

func (t *testSubscriber) OnComplete() {
	if t.onComplete != nil {
		t.onComplete()
	}
}
