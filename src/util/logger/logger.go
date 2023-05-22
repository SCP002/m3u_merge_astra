package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
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

// buildFields returns comma separated <fields> with every second field quoted and colored
func buildFields(fields []any) (out string) {
	cyan := color.New(color.FgHiCyan).SprintFunc()

	for i, field := range fields {
		fieldStr := fmt.Sprint(field)
		if i % 2 == 0 {
			out += fieldStr + " "
		} else {
			out += `"` + cyan(fieldStr) + `"`
			if i < len(fields) - 1 {
				out += ", "
			}
		}
	}

	return strings.TrimSpace(out)
}

// getCallerInfo returns runtime caller info with amount of stack frames to <skip> or empty string on error
func getCallerInfo(skip int) string {
	pc, _, line, _ := runtime.Caller(skip)
	return fmt.Sprintf(`%v; L%v`, runtime.FuncForPC(pc).Name(), line)
}
