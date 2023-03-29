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
	msg := "should not print trace messages with debug level logger"
	assert.NotRegexp(t, regexp.MustCompile(`\[.*\] TRACE message`), out, msg)
	assert.Regexp(t, regexp.MustCompile(`\[.*\] DEBUG message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\]  INFO message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\]  WARN message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\] ERROR message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\] PANIC message`), out)
}

func TestInfoc(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.InfoLevel)

		log.Infoc("message", "field 1", "value 1", "field 2", 10)
	})
	assert.Contains(t, out, `INFO message: field 1 "value 1", field 2 "10"`)
}

func TestDebugc(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.DebugLevel)

		log.Debugc("message", "field 1", "value 1", "field 2", 10)
	})
	msg := `DEBUG (m3u_merge_astra/util/logger.TestDebugc.func1; L44): message: field 1 "value 1", field 2 "10"`
	assert.Contains(t, out, msg)
}

func TestBuildFields(t *testing.T) {
	assert.Exactly(t, ``, buildFields([]any{}))
	assert.Exactly(t, ``, buildFields([]any{""}))
	assert.Exactly(t, `a`, buildFields([]any{"a"}))
	assert.Exactly(t, `a`, buildFields([]any{"a", ""}))
	assert.Exactly(t, `"10"`, buildFields([]any{"", 10}))
	assert.Exactly(t, `"10", b`, buildFields([]any{"", 10, "b", ""}))
	assert.Exactly(t, `a "10"`, buildFields([]any{"a", 10}))
	assert.Exactly(t, `a "10", b "c"`, buildFields([]any{"a", 10, "b", "c"}))
}

func TestGetCallerInfo(t *testing.T) {
	assert.Exactly(t, `m3u_merge_astra/util/logger.TestGetCallerInfo; L62`, getCallerInfo(1))
}
