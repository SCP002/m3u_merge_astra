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
	assert.Regexp(t, regexp.MustCompile(`\[.*\] DEBUG \(.*TestNew.*\): message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\]  INFO message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\]  WARN message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\] ERROR message`), out)
	assert.Regexp(t, regexp.MustCompile(`\[.*\] PANIC message`), out)
}

func TestInfoCFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.InfoLevel)

		log.InfoCFi("message", "field 1", "value 1", "field 2", 10)
	})
	assert.Contains(t, out, `INFO message: field 1 "value 1", field 2 "10"`)
}

func TestWarnCFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.WarnLevel)

		log.WarnCFi("message", "field 1", "value 1", "field 2", 10)
	})
	assert.Contains(t, out, `WARN message: field 1 "value 1", field 2 "10"`)
}

func TestDebug(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.DebugLevel)

		log.Debug("message")
	})
	assert.Contains(t, out, `DEBUG (m3u_merge_astra/util/logger.TestDebug.func1; L53): message`)
}

func TestDebugf(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.DebugLevel)

		log.Debugf("_%v_", "message")
	})
	assert.Contains(t, out, `DEBUG (m3u_merge_astra/util/logger.TestDebugf.func1; L62): _message_`)
}

func TestDebugCFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.DebugLevel)

		log.DebugCFi("message", "field 1", "value 1", "field 2", 10)
	})
	msg := `DEBUG (m3u_merge_astra/util/logger.TestDebugCFi.func1; L71): message: field 1 "value 1", field 2 "10"`
	assert.Contains(t, out, msg)
}

func TestBuildFields(t *testing.T) {
	assert.Exactly(t, ``, buildFields([]any{}))
	assert.Exactly(t, ``, buildFields([]any{""}))
	assert.Exactly(t, `a`, buildFields([]any{"a"}))
	assert.Exactly(t, `a ""`, buildFields([]any{"a", ""}))
	assert.Exactly(t, `"10"`, buildFields([]any{"", 10}))
	assert.Exactly(t, `"10", b ""`, buildFields([]any{"", 10, "b", ""}))
	assert.Exactly(t, `a "10"`, buildFields([]any{"a", 10}))
	assert.Exactly(t, `a "10", b "c"`, buildFields([]any{"a", 10, "b", "c"}))
}

func TestGetCallerInfo(t *testing.T) {
	assert.Exactly(t, `m3u_merge_astra/util/logger.TestGetCallerInfo; L89`, getCallerInfo(1))
}
