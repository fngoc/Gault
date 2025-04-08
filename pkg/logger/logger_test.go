package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	err := NewLogger()
	assert.NoError(t, err)
}

func TestLogInfo(t *testing.T) {
	err := NewLogger()
	assert.NoError(t, err)
	LogInfo("test_message")
}
