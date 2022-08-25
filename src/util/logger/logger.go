package logger

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

// New returns new configured logger
func New(lvl logrus.Level) *logrus.Logger {
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
	return &log
}
