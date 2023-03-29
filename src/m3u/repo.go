package m3u

import (
	"m3u_merge_astra/cfg"
	"github.com/sirupsen/logrus"
)

// repo represents dependencies holder for this package
type repo struct {
	log *logrus.Logger
	cfg cfg.Root
}

// NewRepo returns new dependencies holder for this package
func NewRepo(log *logrus.Logger, cfg cfg.Root) repo {
	return repo{log: log, cfg: cfg}
}

// Log used to satisfy deps.Global interface
func (r repo) Log() *logrus.Logger {
	return r.log
}

// Cfg used to satisfy deps.Global interface
func (r repo) Cfg() cfg.Root {
	return r.cfg
}
