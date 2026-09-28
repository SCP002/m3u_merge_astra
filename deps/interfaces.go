package deps

import (
	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/logger"
)

// Global represents global dependencies holder interface.
type Global interface {
	Log() *logger.Logger
	Cfg() cfg.Root
}
