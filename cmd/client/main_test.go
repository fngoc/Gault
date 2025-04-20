package main

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestRun_VersionFlag(t *testing.T) {
	resetFlags()
	os.Args = []string{"cmd", "--version"}

	err := run()
	assert.NoError(t, err)
}
