package reactive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 128, config.BufferSize)
	assert.Equal(t, 32, config.Prefetch)
}

func TestConfig_OnError(t *testing.T) {
	var called bool
	config := &Config{
		OnError: func(err error) {
			called = true
		},
	}

	if config.OnError != nil {
		config.OnError(assert.AnError)
	}

	assert.True(t, called)
}
