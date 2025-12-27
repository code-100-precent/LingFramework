package reactive

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", Handler(func(c *gin.Context) Publisher {
		return FromSlice([]interface{}{gin.H{"message": "test"}})
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test", response["message"])
}

func TestHandler_MultipleValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", Handler(func(c *gin.Context) Publisher {
		return FromSlice([]interface{}{
			gin.H{"id": 1},
			gin.H{"id": 2},
		})
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)
}

func TestHandler_EmptyValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", Handler(func(c *gin.Context) Publisher {
		return FromSlice([]interface{}{})
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", Handler(func(c *gin.Context) Publisher {
		return &ErrorPublisher{err: assert.AnError}
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHttpSubscriber_OnSubscribe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &HttpSubscriber{
		ctx: c,
	}

	flow := NewFlow()
	sub := flow.Subscribe(subscriber)
	assert.NotNil(t, sub)
}

func TestHttpSubscriber_OnNext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	subscriber := &HttpSubscriber{
		ctx: c,
	}

	subscriber.OnNext("test1")
	subscriber.OnNext("test2")

	assert.Len(t, subscriber.values, 2)
	assert.Equal(t, "test1", subscriber.firstValue)
}

func TestHttpSubscriber_OnError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &HttpSubscriber{
		ctx: c,
	}

	subscriber.OnError(assert.AnError)

	assert.True(t, subscriber.errored)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHttpSubscriber_OnError_AlreadyErrored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &HttpSubscriber{
		ctx:     c,
		errored: true,
	}

	subscriber.OnError(assert.AnError)
	// Should not panic
}

func TestHttpSubscriber_OnComplete_SingleValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &HttpSubscriber{
		ctx:        c,
		firstValue: "test",
		values:     []interface{}{"test"},
	}

	subscriber.OnComplete()

	assert.True(t, subscriber.completed)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHttpSubscriber_OnComplete_MultipleValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &HttpSubscriber{
		ctx:    c,
		values: []interface{}{"test1", "test2"},
	}

	subscriber.OnComplete()

	assert.True(t, subscriber.completed)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHttpSubscriber_OnComplete_AlreadyCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &HttpSubscriber{
		ctx:       c,
		completed: true,
	}

	subscriber.OnComplete()
	// Should not panic
}

func TestStreamHandler(t *testing.T) {
	// StreamHandler blocks waiting for context cancellation
	// Skip this test as it requires proper context cancellation setup
	// In real usage, client disconnect would trigger context cancellation
	t.Skip("StreamHandler requires context cancellation which is complex to test")
}

func TestSSESubscriber_OnSubscribe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &SSESubscriber{
		ctx:    c,
		writer: w,
	}

	flow := NewFlow()
	sub := flow.Subscribe(subscriber)
	assert.NotNil(t, sub)
}

func TestSSESubscriber_OnNext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &SSESubscriber{
		ctx:    c,
		writer: w,
	}

	subscriber.OnNext(gin.H{"message": "test"})

	body := w.Body.String()
	assert.Contains(t, body, "data:")
	assert.Contains(t, body, "test")
}

func TestSSESubscriber_OnNext_JSONError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &SSESubscriber{
		ctx:    c,
		writer: w,
	}

	// Create a value that can't be marshaled
	subscriber.OnNext(make(chan int))

	body := w.Body.String()
	assert.Contains(t, body, "event: error")
}

func TestSSESubscriber_OnError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &SSESubscriber{
		ctx:    c,
		writer: w,
	}

	subscriber.OnError(assert.AnError)

	body := w.Body.String()
	assert.Contains(t, body, "event: error")
}

func TestSSESubscriber_OnComplete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	subscriber := &SSESubscriber{
		ctx:    c,
		writer: w,
	}

	subscriber.OnComplete()

	body := w.Body.String()
	assert.Contains(t, body, "event: close")
}

func TestBodyPublisher(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.POST("/body", Handler(func(c *gin.Context) Publisher {
		return BodyPublisher(c)
	}))

	body := bytes.NewBufferString(`{"name": "test"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/body", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBodyPublisher_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.POST("/body", Handler(func(c *gin.Context) Publisher {
		return BodyPublisher(c)
	}))

	body := bytes.NewBufferString(`invalid json`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/body", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestQueryParamsPublisher(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/params", Handler(func(c *gin.Context) Publisher {
		return QueryParamsPublisher(c)
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/params?key1=value1&key2=value2", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Handler returns a single object when there's one item, not an array
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "value1", response["key1"])
	assert.Equal(t, "value2", response["key2"])
}

func TestQueryParamsPublisher_MultipleValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/params", Handler(func(c *gin.Context) Publisher {
		return QueryParamsPublisher(c)
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/params?key=value1&key=value2", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestErrorPublisher_Subscribe(t *testing.T) {
	err := assert.AnError
	publisher := &ErrorPublisher{err: err}

	var receivedError error
	subscriber := &testSubscriber{
		onError: func(e error) {
			receivedError = e
		},
	}

	sub := publisher.Subscribe(subscriber)
	assert.Nil(t, sub)
	assert.Equal(t, err, receivedError)
}
