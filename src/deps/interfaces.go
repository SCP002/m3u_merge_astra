package deps

import (
	"m3u_merge_astra/cfg"

	"github.com/sirupsen/logrus"
)

// Global represents global dependencies holder interface
type Global interface {
	Log() *logrus.Logger
	Cfg() cfg.Root
}
