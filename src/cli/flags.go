package cli

import (
	"github.com/cockroachdb/errors"
	goFlags "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
)

// Flags represents command line flags
type Flags struct {
	Version        bool         `short:"v" long:"version"        description:"Print the program version"`
	LogLevel       logrus.Level `short:"l" long:"logLevel"       description:"Logging level. Can be from 0 (least verbose) to 6 (most verbose)"`
	ProgramCfgPath string       `short:"c" long:"programCfgPath" description:"Program config file path to read from or initialize a default"`
	M3UPath        string       `short:"m" long:"m3uPath"        description:"M3U file path to get channels from. Can be a local file or URL"`
	AstraAddr      string       `short:"a" long:"astraAddr"      description:"Astra address in format of scheme://host:port"`
	AstraUser      string       `short:"u" long:"astraUser"      description:"Astra user"`
	AstraPwd       string       `short:"p" long:"astraPwd"       description:"Astra password"`
}

// Parse returns a structure initialized with command line arguments and error if parsing failed
func Parse() (Flags, error) {
	flags := Flags{
		// Set defaults
		LogLevel:       logrus.InfoLevel,
		ProgramCfgPath: "m3u_merge_astra.yaml",
		AstraAddr:      "http://127.0.0.1:8000",
	}
	parser := goFlags.NewParser(&flags, goFlags.Options(goFlags.Default))
	_, err := parser.Parse()
	return flags, errors.Wrap(err, "Parse CLI arguments")
}

// IsErrOfType returns true if <err> is of type <t>
func IsErrOfType(err error, t goFlags.ErrorType) bool {
	goFlagsErr := &goFlags.Error{}
	if ok := errors.As(err, &goFlagsErr); ok && goFlagsErr.Type == t {
		return true
	}
	return false
}
