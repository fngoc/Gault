package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	err := Initialize()
	assert.NoError(t, err)
}
