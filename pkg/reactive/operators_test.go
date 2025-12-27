package reactive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	publisher := FromSlice([]int{1, 2, 3, 4, 5})
	mapped := Map(publisher, func(x int) int {
		return x * 2
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

	// Wait a bit for async processing
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, results, 5)
	assert.Equal(t, []int{2, 4, 6, 8, 10}, results)
}

func TestFilter(t *testing.T) {
	publisher := FromSlice([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
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

	assert.Len(t, results, 5)
	assert.Equal(t, []int{2, 4, 6, 8, 10}, results)
}

func TestReduce(t *testing.T) {
	publisher := FromSlice([]int{1, 2, 3, 4, 5})
	reduced := Reduce(publisher, 0, func(acc, x int) int {
		return acc + x
	})

	var result int
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			if v, ok := value.(int); ok {
				result = v
			}
		},
	}

	sub := reduced.Subscribe(subscriber)
	sub.Request(10)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 15, result)
}

func TestTake(t *testing.T) {
	publisher := FromSlice([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	taken := Take[int](publisher, 3)

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

	sub := taken.Subscribe(subscriber)
	sub.Request(10)

	time.Sleep(50 * time.Millisecond)

	assert.Len(t, results, 3)
	assert.Equal(t, []int{1, 2, 3}, results)
	assert.True(t, completed)
}

func TestBuffer(t *testing.T) {
	publisher := FromSlice([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	buffered := Buffer[int](publisher, 3)

	var results [][]int
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			if v, ok := value.([]int); ok {
				results = append(results, v)
			}
		},
	}

	sub := buffered.Subscribe(subscriber)
	sub.Request(10)

	time.Sleep(50 * time.Millisecond)

	// Should have 4 batches: [1,2,3], [4,5,6], [7,8,9], [10]
	assert.GreaterOrEqual(t, len(results), 3)
	if len(results) > 0 {
		assert.Len(t, results[0], 3)
	}
}

func TestFromSlice(t *testing.T) {
	items := []string{"a", "b", "c"}
	publisher := FromSlice(items)

	var results []string
	subscriber := &testSubscriber{
		onNext: func(value interface{}) {
			if v, ok := value.(string); ok {
				results = append(results, v)
			}
		},
	}

	sub := publisher.Subscribe(subscriber)
	sub.Request(10)

	time.Sleep(50 * time.Millisecond)

	assert.Len(t, results, 3)
	assert.Equal(t, []string{"a", "b", "c"}, results)
}

func TestFromChannel(t *testing.T) {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)

	publisher := FromChannel(ch)

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

	time.Sleep(100 * time.Millisecond)

	assert.Len(t, results, 3)
	assert.Equal(t, []int{1, 2, 3}, results)
	assert.True(t, completed)
}
