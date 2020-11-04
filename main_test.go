package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitProcedure(t *testing.T) {
	config := initConfig()
	assert.Equal(t, config.Port, 17630, "Expected default port")
}
