package logger

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	log := New(logrus.DebugLevel)
	log.Debug("Debug message")
	log.Info("Info message")
	log.Warning("Warning message")
	log.Error("Error message")
	assert.Panics(t, func() { log.Panic("Panic message") }, "should panic")
}
