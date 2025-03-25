package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestParseConfig_FileNotFound(t *testing.T) {
	_, err := ParseConfig("non_existing_config")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config")
}

func TestParseConfig_InvalidFormat(t *testing.T) {
	viper.SetConfigType("json")
	_, err := ParseConfig("test_config")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config")
}
