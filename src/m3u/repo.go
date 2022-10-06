package m3u

import (
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/tw"
	"github.com/sirupsen/logrus"
)

// repo represents dependencies holder for this package
type repo struct {
	log *logrus.Logger
	tw  tw.Writer
	cfg cfg.Root
}

// NewRepo returns new dependencies holder for this package
func NewRepo(log *logrus.Logger, tw tw.Writer, cfg cfg.Root) repo {
	return repo{log: log, tw: tw, cfg: cfg}
}

// Log used to satisfy deps.Global interface
func (r repo) Log() *logrus.Logger {
	return r.log
}

// TW used to satisfy deps.Global interface
func (r repo) TW() tw.Writer {
	return r.tw
}

// Cfg used to satisfy deps.Global interface
func (r repo) Cfg() cfg.Root {
	return r.cfg
}
