package logger

import (
	"bufio"
	"os"
	"path/filepath"
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

func TestErrorCFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.ErrorLevel)

		log.ErrorCFi("message", "field 1", "value 1", "field 2", 10)
	})
	assert.Contains(t, out, `ERROR message: field 1 "value 1", field 2 "10"`)
}

func TestDebug(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.DebugLevel)

		log.Debug("message")
	})
	assert.Contains(t, out, `DEBUG (m3u_merge_astra/util/logger.TestDebug.func1; L64): message`)
}

func TestDebugf(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.DebugLevel)

		log.Debugf("_%v_", "message")
	})
	assert.Contains(t, out, `DEBUG (m3u_merge_astra/util/logger.TestDebugf.func1; L73): _message_`)
}

func TestDebugCFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(logrus.DebugLevel)

		log.DebugCFi("message", "field 1", "value 1", "field 2", 10)
	})
	msg := `DEBUG (m3u_merge_astra/util/logger.TestDebugCFi.func1; L82): message: field 1 "value 1", field 2 "10"`
	assert.Contains(t, out, msg)
}

func TestAddFileHook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	log := New(logrus.DebugLevel)
	file, err := log.AddFileHook("")
	assert.NoError(t, err, "should not return error for empty file path")
	assert.Nil(t, file, "should return nil file for empty file path")

	file, err = log.AddFileHook(path)
	assert.NoError(t, err, "should not return error")
	assert.NotNil(t, file, "should create object")
	defer file.Close()
	assert.FileExists(t, path, "should create log file at given path")
}

func TestFileHookLevels(t *testing.T) {
	assert.Exactly(t, logrus.AllLevels, fileHook{}.Levels())
}

func TestFileHookFire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	log := New(logrus.DebugLevel)
	file, err := log.AddFileHook(path)
	assert.NoError(t, err, "should not return error")
	assert.NotNil(t, file, "should create file object")
	assert.FileExists(t, path, "should create log file at given path")

	log.WithFields(logrus.Fields{"a": "b", "c": "d"}).Info("message 1")
	log.InfoCFi("message 2", "e", "f", "g", "h")

	// file.Sync() does not help, content is empty, closing and opening the file again
	file.Close()
	file, err = os.Open(path)
	assert.NoError(t, err, "should not return error")
	defer file.Close()
	reader := bufio.NewReader(file)

	line, err := reader.ReadString('\n')
	assert.NoError(t, err, "should not return error")
	rx := regexp.MustCompile(`^[0-9]{4}\.[0-9]{2}\.[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} INFO message 1: a=b, c=d, \n$`)
	assert.Regexp(t, rx, line, "first line in log file should match this regexp")

	line, err = reader.ReadString('\n')
	assert.NoError(t, err, "should not return error")
	rx = regexp.MustCompile(`^[0-9]{4}\.[0-9]{2}\.[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} INFO message 2: e "f", g "h"\n$`)
	assert.Regexp(t, rx, line, "second line in log file should match this regexp")
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
	assert.Exactly(t, `m3u_merge_astra/util/logger.TestGetCallerInfo; L136`, getCallerInfo(1))
}
