package deps

import (
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/tw"

	"github.com/sirupsen/logrus"
)

// Global represents global dependencies holder interface
type Global interface {
	Log() *logrus.Logger
	TW() tw.Writer
	Cfg() cfg.Root
}
