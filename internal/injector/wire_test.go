package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitializeLogger(t *testing.T) {
	err := InitializeLogger
	assert.NotNil(t, err)
}
