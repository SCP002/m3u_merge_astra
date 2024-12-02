package logger

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/cockroachdb/errors"
	"github.com/fatih/color"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

// Logger represents wrapper over logrus
type Logger struct {
	*logrus.Logger
}

// New returns new configured logger with log level <lvl>
func New(lvl logrus.Level) *Logger {
	formatter := prefixed.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.Stamp,
		ForceFormatting: true,
	}
	log := logrus.Logger{
		Out:       os.Stderr,
		Formatter: &formatter,
		Level:     lvl,
		Hooks:     make(logrus.LevelHooks),
	}
	return &Logger{&log}
}

// InfoCFi prints info level <msg> with formatted and colored <fields>
func (l Logger) InfoCFi(msg string, fields ...any) {
	l.Logger.Infof("%v: %v", msg, buildFields(fields))
}

// WarnCFi prints warning level <msg> with formatted and colored <fields>
func (l Logger) WarnCFi(msg string, fields ...any) {
	l.Logger.Warnf("%v: %v", msg, buildFields(fields))
}

// ErrorCFi prints error level <msg> with formatted and colored <fields>
func (l Logger) ErrorCFi(msg string, fields ...any) {
	l.Logger.Errorf("%v: %v", msg, buildFields(fields))
}

// Debug prints debug level <msg>.
//
// Output is prefixed with caller info.
func (l Logger) Debug(msg any) {
	msgWithCaller := fmt.Sprintf(`(%v): %v`, getCallerInfo(2), msg)

	l.Logger.Debug(msgWithCaller)
}

// Debugf prints debug level message <args> formatted according to a <format> specifier.
//
// Output is prefixed with caller info.
func (l Logger) Debugf(format string, args ...any) {
	formatWithCaller := fmt.Sprintf(`(%v): %v`, getCallerInfo(2), format)

	l.Logger.Debugf(formatWithCaller, args...)
}

// DebugCFi prints debug level <msg> with formatted and colored <fields>.
//
// Output is prefixed with caller info.
func (l Logger) DebugCFi(msg string, fields ...any) {
	msgWithCaller := fmt.Sprintf(`(%v): %v`, getCallerInfo(2), msg)

	l.Logger.Debugf("%v: %v", msgWithCaller, buildFields(fields))
}

// AddFileHook creates log file at <filePath> and adds file hook to logrus.
//
// If <filePath> is empty string, it does nothing and returns nil error.
func (l Logger) AddFileHook(filePath string) (*os.File, error) {
	if filePath == "" {
		return nil, nil
	}

	logFile, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "Open or create log file")
	}
	l.AddHook(fileHook{file: logFile})

	return logFile, nil
}

// fileHook represents logrus file hook
type fileHook struct {
	file *os.File
}

// Levels returns which levels to fire the hook at.
//
// Used to implement logrus Hook interface.
func (h fileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is executed when the hook runs, writing formatted <entry> to file.
//
// Used to implement logrus Hook interface.
func (h fileHook) Fire(entry *logrus.Entry) error {
	time := entry.Time.Format("2006.01.02 15:04:05")
	level := strings.ToUpper(entry.Level.String())

	// Start building message and remove ANSI escape codes from it to cleanup after InfoCFi etc. calls
	msg := fmt.Sprintf("%s %s %s", time, level, stripansi.Strip(entry.Message))

	// Format and add fields
	if len(entry.Data) > 0 {
		var sb strings.Builder
		keys := lo.Keys(entry.Data)
		slices.Sort(keys)
		for _, key := range keys {
			val := entry.Data[key]
			sb.WriteString(fmt.Sprintf("%s=%+v, ", key, val))
		}
		msg = fmt.Sprintf("%s: %+v", msg, sb.String())
		msg = strings.TrimRight(msg, ", ")
	}

	_, err := h.file.WriteString(msg + "\n")
	return err
}

// buildFields returns comma separated <fields> with every second field quoted and colored
func buildFields(fields []any) string {
	cyan := color.New(color.FgHiCyan).SprintFunc()
	var sb strings.Builder

	for i, field := range fields {
		fieldStr := fmt.Sprint(field)
		if i % 2 == 0 {
			sb.WriteString(fieldStr)
			sb.WriteRune(' ')
		} else {
			sb.WriteRune('"')
			sb.WriteString(cyan(fieldStr))
			sb.WriteRune('"')
			if i < len(fields) - 1 {
				sb.WriteString(", ")
			}
		}
	}

	return strings.TrimSpace(sb.String())
}

// getCallerInfo returns runtime caller info with amount of stack frames to <skip> or empty string on error
func getCallerInfo(skip int) string {
	pc, _, line, _ := runtime.Caller(skip)
	return fmt.Sprintf(`%v; L%v`, runtime.FuncForPC(pc).Name(), line)
}
