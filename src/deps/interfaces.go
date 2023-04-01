package deps

import (
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/logger"
)

// Global represents global dependencies holder interface
type Global interface {
	Log() *logger.Logger
	Cfg() cfg.Root
}
