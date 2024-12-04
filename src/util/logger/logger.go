package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/fatih/color"
	pLogger "github.com/phuslu/log"
	"github.com/samber/lo"
)

// levelColorMap represents mapping between log level string and it's version for console writer
var levelColorMap = map[string]string{
	"trace": color.BlueString("TRACE"),
	"debug": color.BlueString("DEBUG"),
	"info":  color.GreenString("INFO"),
	"warn":  color.YellowString("WARN"),
	"error": color.RedString("ERROR"),
	"fatal": color.RedString("FATAL"),
	"panic": color.RedString("PANIC"),
}

// Logger represents wrapper over logging library
type Logger struct {
	writer *pLogger.MultiEntryWriter
	*pLogger.Logger
}

// New returns new configured logger with log level <lvl>
func New(lvl pLogger.Level) *Logger {
	writer := pLogger.MultiEntryWriter{
		&pLogger.ConsoleWriter{
			Formatter: newConsoleFormatter(true, "2006.01.02 15:04:05"),
			Writer:    os.Stderr,
		},
	}

	log := pLogger.Logger{
		Level:  lvl,
		Writer: &writer,
	}

	return &Logger{Logger: &log, writer: &writer}
}

// AddFileWriter creates log file at <filePath> and adds file writer to logger.
//
// If <filePath> is empty string, it does nothing and returns nil error.
func (l *Logger) AddFileWriter(filePath string) (*os.File, error) {
	if filePath == "" {
		return nil, nil
	}

	logFile, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "Open or create log file")
	}
	*l.writer = append(*l.writer, &pLogger.IOWriter{
		Writer: logFile,
	})

	return logFile, nil
}

// newConsoleFormatter returns formatter funtion with <timeFormat> for console writer.
//
// If <colorize> is true, add colors to output.
func newConsoleFormatter(colorize bool, timeFormat string) func(io.Writer, *pLogger.FormatterArgs) (int, error) {
	return func(w io.Writer, a *pLogger.FormatterArgs) (int, error) {
		gray := color.RGB(118, 118, 118).SprintFunc()
		var messageSb strings.Builder

		formatterTime, err := time.Parse(time.RFC3339Nano, a.Time) // Get time object from FormatterArgs
		if err != nil {
			return 0, err
		}
		properTime := formatterTime.Format(timeFormat)
		if colorize {
			messageSb.WriteString(gray(properTime))
		} else {
			messageSb.WriteString(properTime)
		}
		messageSb.WriteRune(' ')

		if colorize {
			messageSb.WriteString(levelColorMap[a.Level])
		} else {
			messageSb.WriteString(strings.ToUpper(a.Level))
		}
		messageSb.WriteRune(' ')

		if a.Caller != "" {
			messageSb.WriteRune('[')
			if colorize {
				messageSb.WriteString(gray(a.Caller))
			} else {
				messageSb.WriteString(a.Caller)
			}
			messageSb.WriteString("] ")
		}

		messageSb.WriteString(a.Message)

		a.KeyValues = lo.Reject(a.KeyValues, func(kv struct{Key string; Value string; ValueType byte}, _ int) bool {
			return kv.Key == "callerfunc"
		})
		if len(a.KeyValues) > 0 {
			messageSb.WriteString(": ")
		}
		for idx, item := range a.KeyValues {
 			messageSb.WriteString(item.Key + " \"")
			if colorize {
				messageSb.WriteString(color.CyanString(item.Value))
			} else {
				messageSb.WriteString(item.Value)
			}
			messageSb.WriteRune('"')
			if idx < len(a.KeyValues)-1 {
				messageSb.WriteString(", ")
			}
		}
		messageSb.WriteRune('\n')

		return fmt.Fprint(w, messageSb.String())
	}
}
