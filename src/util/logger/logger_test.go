package logger

import (
	"regexp"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

func TestNew(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.DebugLevel)
		log.Trace("message")
		log.Debug("message")
		log.Info("message")
		log.Warning("message")
		log.Error("message")
		assert.Panics(t, func() { log.Panic("message") }, "should panic")
	})
	assert.NotRegexp(t, regexp.MustCompile(`\[.*\] TRACE message`), out, "should not print messages with trace level")
	assert.Regexp(t, regexp.MustCompile(`\[.*\] DEBUG message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\]  INFO message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\]  WARN message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\] ERROR message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\] PANIC message`), out)
}
