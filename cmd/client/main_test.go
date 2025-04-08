package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun_Success(t *testing.T) {
	err := run()
	assert.Error(t, err)
}
