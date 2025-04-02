package config

import (
	"os"
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

func TestParseConfig_UnmarshalFailure(t *testing.T) {
	tmpFile, err := os.Create("bad_config.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Некорректная структура: массив вместо объектов
	content := `
port:
  - not_a_number
`
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	viper.Reset()

	_, err = ParseConfig("bad_config")
	assert.Error(t, err)
}

func TestParseConfig_Success(t *testing.T) {
	tmpFile, err := os.Create("good_config.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := `
port: 9090
db: "host=localhost user=postgres password=postgres dbname=test_db sslmode=disable"
allowEndpoints:
  - path: "/ping"
    allowed: true
`
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	viper.Reset()

	conf, err := ParseConfig("good_config")
	assert.NoError(t, err)
	assert.Equal(t, 9090, conf.Port)
	assert.Equal(t, "host=localhost user=postgres password=postgres dbname=test_db sslmode=disable", conf.DB)
	assert.Len(t, conf.AllowEndpoints, 1)
}
