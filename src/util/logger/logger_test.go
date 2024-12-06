package logger

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	pLog "github.com/phuslu/log"
	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

var timeRx = `[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}`

func TestNew(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.Trace("message")
		log.Debug("message")
		log.DebugFi("message", "k", "v")
		log.InfoFi("message", "1", 2)
		log.Warn("message")
		log.Error("message")
		assert.Panics(t, func() { log.Panic("message") }, "should panic")
	})
	assert.NotRegexp(t, regexp.MustCompile(`TRACE`), out, "should not print trace messages with debug level logger")
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG logger\/logger_test\.go:24 message`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG logger\/logger_test\.go:25 message: k "v"`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message: 1 "2"`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx+` WARN message`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx+` ERROR message`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx+` PANIC message`), out)
}

func TestTrace(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(TraceLevel)
		log.Trace("message")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` TRACE logger\/logger_test\.go:43 message`), out)
}

func TestTracef(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(TraceLevel)
		log.Tracef("%v + %v", "message 1", "message 2")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` TRACE logger\/logger_test\.go:51 message 1 \+ message 2`), out)
}

func TestTraceFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(TraceLevel)
		log.TraceFi("message", "a", "b", 1, 2)
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` TRACE logger\/logger_test\.go:59 message: a "b", 1 "2"`), out)
}

func TestDebug(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.Debug("message")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG logger\/logger_test\.go:67 message`), out)
}

func TestDebugf(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.Debugf("%v + %v", "message 1", "message 2")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG logger\/logger_test\.go:75 message 1 \+ message 2`), out)
}

func TestDebugFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.DebugFi("message", "a", "b", 1, 2)
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG logger\/logger_test\.go:83 message: a "b", 1 "2"`), out)
}

func TestInfo(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.Info("message")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message`), out)
}

func TestInfof(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.Infof("%v + %v", "message 1", "message 2")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message 1 \+ message 2`), out)
}

func TestInfoFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.InfoFi("message", "a", "b", 1, 2)
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message: a "b", 1 "2"`), out)
}

func TestWarn(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.Warn("message")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` WARN message`), out)
}

func TestWarnf(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.Warnf("%v + %v", "message 1", "message 2")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` WARN message 1 \+ message 2`), out)
}

func TestWarnFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.WarnFi("message", "a", "b", 1, 2)
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` WARN message: a "b", 1 "2"`), out)
}

func TestError(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.Error("message")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` ERROR message`), out)
}

func TestErrorf(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.Errorf("%v + %v", "message 1", "message 2")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` ERROR message 1 \+ message 2`), out)
}

func TestErrorFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		log.ErrorFi("message", "a", "b", 1, 2)
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` ERROR message: a "b", 1 "2"`), out)
}

func TestFatal(t *testing.T) {
	if os.Getenv("TestFatal") == "1" {
		log := New(DebugLevel)
		log.Fatal("message")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFatal$")
	cmd.Env = append(os.Environ(), "TestFatal=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		assert.Exactly(t, 255, e.ExitCode(), "should exit with that exit code")
		// Does not collect Stderr
		// assert.Regexp(t, regexp.MustCompile(timeRx+` FATAL message`), string(e.Stderr))
		return
	}
	assert.Failf(t, "", "expected exec.ExitError, got '%v'", err)
}

func TestFatalf(t *testing.T) {
	if os.Getenv("TestFatal") == "1" {
		log := New(DebugLevel)
		log.Fatalf("%v + %v", "message 1", "message 2")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFatal$")
	cmd.Env = append(os.Environ(), "TestFatal=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		assert.Exactly(t, 255, e.ExitCode(), "should exit with that exit code")
		// Does not collect Stderr
		// assert.Regexp(t, regexp.MustCompile(timeRx+` FATAL message 1 \+ message 2`), string(e.Stderr))
		return
	}
	assert.Failf(t, "", "expected exec.ExitError, got '%v'", err)
}

func TestFatalFi(t *testing.T) {
	if os.Getenv("TestFatal") == "1" {
		log := New(DebugLevel)
		log.FatalFi("message", "a", "b", 1, 2)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFatal$")
	cmd.Env = append(os.Environ(), "TestFatal=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		assert.Exactly(t, 255, e.ExitCode(), "should exit with that exit code")
		// Does not collect Stderr
		// assert.Regexp(t, regexp.MustCompile(timeRx+` FATAL message: a "b", 1 "2"`), string(e.Stderr))
		return
	}
	assert.Failf(t, "", "expected exec.ExitError, got '%v'", err)
}

func TestPanic(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		assert.Panics(t, func() { log.Panic("message") }, "should panic")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` PANIC message`), out)
}

func TestPanicf(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		assert.Panics(t, func() { log.Panicf("%v + %v", "message 1", "message 2") }, "should panic")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` PANIC message 1 \+ message 2`), out)
}

func TestPanicFi(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		assert.Panics(t, func() { log.PanicFi("message", "a", "b", 1, 2) }, "should panic")
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` PANIC message: a "b", 1 "2"`), out)
}

func TestAddFileWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	log := New(DebugLevel)
	file, err := log.AddFileWriter("")
	assert.NoError(t, err, "should not return error for empty file path")
	assert.Nil(t, file, "should return nil file for empty file path")

	file, err = log.AddFileWriter(path)
	assert.NoError(t, err, "should not return error")
	assert.NotNil(t, file, "should create object")
	assert.FileExists(t, path, "should create log file at given path")

	log.InfoFi("message 1", "field1", "value 1", "field2", 2)
	log.WarnFi("message 2", "field1", "value 3", "field2", 4)

	// file.Sync() does not help, content is empty, closing and opening the file again
	file.Close()
	file, err = os.Open(path)
	assert.NoError(t, err, "should not return error")
	defer file.Close()
	reader := bufio.NewReader(file)

	type logEntry struct {
		Time    string `json:"time"`
		Level   string `json:"level"`
		Field1  string `json:"field1"`
		Field2  int    `json:"field2"`
		Message string `json:"message"`
	}

	var entry logEntry
	timeRx := regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}\+[0-9]{2}:[0-9]{2}$`)

	line, err := reader.ReadBytes('\n')
	assert.NoError(t, err, "should not return error")
	err = json.Unmarshal(line, &entry)
	assert.NoError(t, err, "should not return error")
	assert.Regexp(t, timeRx, entry.Time, "time in entry must match regexp format")
	assert.Exactly(t, "info", entry.Level, "log severity must be that level")
	assert.Exactly(t, "value 1", entry.Field1, "field should have this value")
	assert.Exactly(t, 2, entry.Field2, "field should have this value")
	assert.Exactly(t, "message 1", entry.Message, "should be that entry message")

	line, err = reader.ReadBytes('\n')
	assert.NoError(t, err, "should not return error")
	err = json.Unmarshal(line, &entry)
	assert.NoError(t, err, "should not return error")
	assert.Regexp(t, timeRx, entry.Time, "time in entry must match regexp format")
	assert.Exactly(t, "warn", entry.Level, "log severity must be that level")
	assert.Exactly(t, "value 3", entry.Field1, "field should have this value")
	assert.Exactly(t, 4, entry.Field2, "field should have this value")
	assert.Exactly(t, "message 2", entry.Message, "should be that entry message")
}

func TestPrint(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		print(log.Logger.Info(), "message", nil)
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message`), out)

	out = capturer.CaptureStderr(func() {
		log := New(DebugLevel)
		print(log.Logger.Info(), "message", []any{"a", "b", 1, 2})
	})
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message: a "b", 1 "2"`), out)
}

func TestNewConsoleFormatter(t *testing.T) {
	// Tested in TestNew
	var formatter func(io.Writer, *pLog.FormatterArgs) (int, error)
	assert.IsType(t, formatter, newConsoleFormatter(false, ""), "formatter function should have this definition")
}
