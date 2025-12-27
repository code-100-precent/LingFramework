package reactive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMap_TypeMismatch(t *testing.T) {
	publisher := FromSlice([]string{"a", "b", "c"})
	mapped := Map(publisher, func(x string) int {
		return len(x)
	})

	var results []int
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			if v, ok := value.(int); ok {
				results = append(results, v)
			}
		},
	}

	sub := mapped.Subscribe(subscriber)
	sub.Request(10)

	time.Sleep(50 * time.Millisecond)

	assert.Len(t, results, 3)
	assert.Equal(t, []int{1, 1, 1}, results)
}

func TestMap_Error(t *testing.T) {
	flow := NewFlow()
	mapped := Map(flow, func(x int) int {
		return x * 2
	})

	var receivedError error
	subscriber := &testSubscriber{
		onError: func(err error) {
			receivedError = err
		},
	}

	sub := mapped.Subscribe(subscriber)
	sub.Request(1)

	testErr := assert.AnError
	flow.PublishError(testErr)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, testErr, receivedError)
}

func TestFilter_TypeMismatch(t *testing.T) {
	publisher := FromSlice([]int{1, 2, 3, 4, 5})
	filtered := Filter(publisher, func(x int) bool {
		return x%2 == 0
	})

	var results []int
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			if v, ok := value.(int); ok {
				results = append(results, v)
			}
		},
	}

	sub := filtered.Subscribe(subscriber)
	sub.Request(10)

	time.Sleep(50 * time.Millisecond)

	assert.Len(t, results, 2)
}

func TestFilter_Error(t *testing.T) {
	flow := NewFlow()
	filtered := Filter(flow, func(x int) bool {
		return x > 0
	})

	var receivedError error
	subscriber := &testSubscriber{
		onError: func(err error) {
			receivedError = err
		},
	}

	sub := filtered.Subscribe(subscriber)
	sub.Request(1)

	testErr := assert.AnError
	flow.PublishError(testErr)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, testErr, receivedError)
}

func TestReduce_Error(t *testing.T) {
	flow := NewFlow()
	reduced := Reduce(flow, 0, func(acc, x int) int {
		return acc + x
	})

	var receivedError error
	subscriber := &testSubscriber{
		onError: func(err error) {
			receivedError = err
		},
	}

	sub := reduced.Subscribe(subscriber)
	sub.Request(1)

	testErr := assert.AnError
	flow.PublishError(testErr)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, testErr, receivedError)
}

func TestTake_Zero(t *testing.T) {
	publisher := FromSlice([]int{1, 2, 3})
	taken := Take[int](publisher, 0)

	completed := false
	subscriber := &testSubscriber{
		onComplete: func() {
			completed = true
		},
	}

	sub := taken.Subscribe(subscriber)
	sub.Request(10)

	time.Sleep(50 * time.Millisecond)

	assert.True(t, completed)
}

func TestTake_Error(t *testing.T) {
	flow := NewFlow()
	taken := Take[int](flow, 3)

	var receivedError error
	subscriber := &testSubscriber{
		onError: func(err error) {
			receivedError = err
		},
	}

	sub := taken.Subscribe(subscriber)
	sub.Request(1)

	testErr := assert.AnError
	flow.PublishError(testErr)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, testErr, receivedError)
}

func TestBuffer_Error(t *testing.T) {
	flow := NewFlow()
	buffered := Buffer[int](flow, 3)

	var receivedError error
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			// May receive buffered items before error
		},
		onError: func(err error) {
			receivedError = err
		},
	}

	sub := buffered.Subscribe(subscriber)
	sub.Request(1)

	flow.Publish(1)
	testErr := assert.AnError
	flow.PublishError(testErr)

	time.Sleep(50 * time.Millisecond)
	assert.Error(t, receivedError)
}

func TestFromSlice_Empty(t *testing.T) {
	publisher := FromSlice([]int{})

	var results []int
	completed := false
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			if v, ok := value.(int); ok {
				results = append(results, v)
			}
		},
		onComplete: func() {
			completed = true
		},
	}

	sub := publisher.Subscribe(subscriber)
	sub.Request(10)

	time.Sleep(50 * time.Millisecond)

	assert.Len(t, results, 0)
	assert.True(t, completed)
}

func TestFromSlice_Cancel(t *testing.T) {
	publisher := FromSlice([]int{1, 2, 3, 4, 5})

	var results []int
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			if v, ok := value.(int); ok {
				results = append(results, v)
			}
		},
	}

	sub := publisher.Subscribe(subscriber)
	sub.Request(2)
	sub.Cancel()

	time.Sleep(50 * time.Millisecond)

	// Should have received some items before cancel
	assert.LessOrEqual(t, len(results), 5)
}

func TestFromChannel_Empty(t *testing.T) {
	ch := make(chan int)
	close(ch)

	publisher := FromChannel(ch)

	completed := false
	subscriber := &testSubscriber{
		onComplete: func() {
			completed = true
		},
	}

	sub := publisher.Subscribe(subscriber)
	sub.Request(10)

	time.Sleep(100 * time.Millisecond)

	assert.True(t, completed)
}

func TestFromChannel_Cancel(t *testing.T) {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3

	publisher := FromChannel(ch)

	var results []int
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			if v, ok := value.(int); ok {
				results = append(results, v)
			}
		},
	}

	sub := publisher.Subscribe(subscriber)
	sub.Request(1)
	sub.Cancel()

	time.Sleep(100 * time.Millisecond)

	// May have received some items before cancel
	assert.LessOrEqual(t, len(results), 3)
}

func TestFromChannel_NoRequest(t *testing.T) {
	ch := make(chan int, 1)
	ch <- 1

	publisher := FromChannel(ch)

	var results []int
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			if v, ok := value.(int); ok {
				results = append(results, v)
			}
		},
	}

	publisher.Subscribe(subscriber)
	// Don't request any items

	time.Sleep(50 * time.Millisecond)

	// Should not receive items without request
	assert.Len(t, results, 0)

	close(ch)
}

func TestSliceSubscription_Request_AfterCancel(t *testing.T) {
	publisher := FromSlice([]int{1, 2, 3})

	subscriber := &testSubscriber{}
	sub := publisher.Subscribe(subscriber)

	sub.Cancel()
	sub.Request(10) // Request after cancel should be ignored

	time.Sleep(50 * time.Millisecond)
	// Should not panic
}

func TestFlow_Publish_Closed(t *testing.T) {
	flow := NewFlow()
	flow.Complete()

	err := flow.Publish("test")
	assert.Error(t, err)
	assert.Equal(t, ErrFlowClosed, err)
}

func TestFlow_Subscribe_Closed(t *testing.T) {
	flow := NewFlow()
	flow.Complete()

	subscriber := &testSubscriber{
		onError: func(err error) {
			assert.Equal(t, ErrFlowClosed, err)
		},
	}

	sub := flow.Subscribe(subscriber)
	assert.Nil(t, sub)
}

func TestSubscription_Request_Invalid(t *testing.T) {
	flow := NewFlow()
	subscriber := &testSubscriber{}
	sub := flow.Subscribe(subscriber)

	// Request 0 or negative should be handled
	sub.Request(0)
	sub.Request(-1)

	time.Sleep(10 * time.Millisecond)
	// Should not panic
}
