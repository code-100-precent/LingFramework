package reactive

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ReactiveHandler is a handler that processes requests reactively
type ReactiveHandler func(*gin.Context) Publisher

// Handler adapts a ReactiveHandler to a gin.HandlerFunc
func Handler(reactiveHandler ReactiveHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		publisher := reactiveHandler(c)

		subscriber := &HttpSubscriber{
			ctx: c,
		}

		subscription := publisher.Subscribe(subscriber)
		if subscription != nil {
			subscription.Request(1) // Request first item
		}
	}
}

// HttpSubscriber implements Subscriber for HTTP responses
type HttpSubscriber struct {
	ctx        *gin.Context
	firstValue interface{}
	values     []interface{}
	completed  bool
	errored    bool
}

func (s *HttpSubscriber) OnSubscribe(subscription Subscription) {
	// Request more items
	subscription.Request(10) // Request in batches
}

func (s *HttpSubscriber) OnNext(value interface{}) {
	if s.errored {
		return
	}

	if s.firstValue == nil {
		s.firstValue = value
	}
	s.values = append(s.values, value)
}

func (s *HttpSubscriber) OnError(err error) {
	if s.errored || s.completed {
		return
	}
	s.errored = true
	s.ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": err.Error(),
	})
}

func (s *HttpSubscriber) OnComplete() {
	if s.errored || s.completed {
		return
	}
	s.completed = true

	if len(s.values) == 0 {
		s.ctx.JSON(http.StatusOK, gin.H{})
		return
	}

	if len(s.values) == 1 {
		// Single value response
		s.ctx.JSON(http.StatusOK, s.firstValue)
	} else {
		// Array response
		s.ctx.JSON(http.StatusOK, s.values)
	}
}

// StreamHandler creates a handler that streams Server-Sent Events (SSE)
func StreamHandler(reactiveHandler ReactiveHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		publisher := reactiveHandler(c)

		subscriber := &SSESubscriber{
			ctx:    c,
			writer: c.Writer,
		}

		subscription := publisher.Subscribe(subscriber)
		if subscription != nil {
			subscription.Request(1)
		}

		// Wait for context cancellation
		<-c.Request.Context().Done()
		subscription.Cancel()
	}
}

// SSESubscriber implements Subscriber for Server-Sent Events
type SSESubscriber struct {
	ctx     *gin.Context
	writer  http.ResponseWriter
	flusher http.Flusher
}

func (s *SSESubscriber) OnSubscribe(subscription Subscription) {
	s.flusher, _ = s.writer.(http.Flusher)
	subscription.Request(1)
}

func (s *SSESubscriber) OnNext(value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		s.OnError(err)
		return
	}

	s.writer.Write([]byte("data: "))
	s.writer.Write(data)
	s.writer.Write([]byte("\n\n"))

	if s.flusher != nil {
		s.flusher.Flush()
	}
}

func (s *SSESubscriber) OnError(err error) {
	s.writer.Write([]byte("event: error\n"))
	data, _ := json.Marshal(gin.H{"error": err.Error()})
	s.writer.Write([]byte("data: "))
	s.writer.Write(data)
	s.writer.Write([]byte("\n\n"))

	if s.flusher != nil {
		s.flusher.Flush()
	}
}

func (s *SSESubscriber) OnComplete() {
	s.writer.Write([]byte("event: close\n"))
	s.writer.Write([]byte("data: {}\n\n"))

	if s.flusher != nil {
		s.flusher.Flush()
	}
}

// BodyPublisher creates a Publisher from the request body
func BodyPublisher(c *gin.Context) Publisher {
	var body interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		return &ErrorPublisher{err: err}
	}
	return FromSlice([]interface{}{body})
}

// ErrorPublisher publishes an error
type ErrorPublisher struct {
	err error
}

func (e *ErrorPublisher) Subscribe(subscriber Subscriber) Subscription {
	subscriber.OnError(e.err)
	return nil
}

// QueryParamsPublisher creates a Publisher from query parameters
func QueryParamsPublisher(c *gin.Context) Publisher {
	params := make(map[string]interface{})
	for key, values := range c.Request.URL.Query() {
		if len(values) == 1 {
			params[key] = values[0]
		} else {
			params[key] = values
		}
	}
	return FromSlice([]interface{}{params})
}
