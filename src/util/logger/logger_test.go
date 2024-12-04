package logger

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	pLogger "github.com/phuslu/log"
	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

func TestNew(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		log := New(pLogger.DebugLevel)
		log.Trace().Caller(1).Msg("message")
		log.Debug().Caller(1).Msg("message")
		log.Debug().Caller(1).Str("k", "v").Msg("message")
		log.Info().Str("k", "v").Msg("message")
		log.Warn().Msg("message")
		log.Error().Msg("message")
		assert.Panics(t, func() { log.Panic().Msg("message") }, "should panic")
	})
	msg := "should not print trace messages with debug level logger"
	timeRx := `[0-9]{4}\.[0-9]{2}\.[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}`
	assert.NotRegexp(t, regexp.MustCompile(timeRx + `TRACE \[logger\/logger_test\.go:20\] message`), out, msg)
	assert.Regexp(t, regexp.MustCompile(timeRx + ` DEBUG \[logger\/logger_test\.go:21\] message`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx + ` DEBUG \[logger\/logger_test\.go:22\] message: k "v"`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx + ` INFO message: k "v"`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx + ` WARN message`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx + ` ERROR message`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx + ` PANIC message`), out)
}

func TestAddFileWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	log := New(pLogger.DebugLevel)
	file, err := log.AddFileWriter("")
	assert.NoError(t, err, "should not return error for empty file path")
	assert.Nil(t, file, "should return nil file for empty file path")

	file, err = log.AddFileWriter(path)
	assert.NoError(t, err, "should not return error")
	assert.NotNil(t, file, "should create object")
	assert.FileExists(t, path, "should create log file at given path")

	log.Info().Str("field1", "value 1").Str("field2", "value 2").Msg("message 1")
	log.Warn().Str("field1", "value 3").Str("field2", "value 4").Msg("message 2")

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
		Field2  string `json:"field2"`
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
	assert.Exactly(t, "value 2", entry.Field2, "field should have this value")
	assert.Exactly(t, "message 1", entry.Message, "should be that entry message")

	line, err = reader.ReadBytes('\n')
	assert.NoError(t, err, "should not return error")
	err = json.Unmarshal(line, &entry)
	assert.NoError(t, err, "should not return error")
	assert.Regexp(t, timeRx, entry.Time, "time in entry must match regexp format")
	assert.Exactly(t, "warn", entry.Level, "log severity must be that level")
	assert.Exactly(t, "value 3", entry.Field1, "field should have this value")
	assert.Exactly(t, "value 4", entry.Field2, "field should have this value")
	assert.Exactly(t, "message 2", entry.Message, "should be that entry message")
}

func TestNewConsoleFormatter(t *testing.T) {
	// Tested in TestNew
	var formatter func(io.Writer, *pLogger.FormatterArgs) (int, error)
	assert.IsType(t, formatter, newConsoleFormatter(false, ""), "formatter function should have this definition")
}
