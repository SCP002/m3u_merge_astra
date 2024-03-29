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
	AstraCfgInput  string       `short:"i" long:"astraCfgInput"  description:"Input astra config. Can be 'clipboard', 'stdio' or file path"`
	AstraCfgOutput string       `short:"o" long:"astraCfgOutput" description:"Output astra config. Can be 'clipboard', 'stdio' or file path"`
}

// AstraCfgIOType represents type of input and output to read or write astra config
type AstraCfgIOType string

const (
	Clipboard AstraCfgIOType = "clipboard"
	Stdio     AstraCfgIOType = "stdio"
)

// Parse returns a structure initialized with command line arguments and error if parsing failed
func Parse() (Flags, error) {
	flags := Flags{
		// Set defaults
		LogLevel:       logrus.InfoLevel,
		ProgramCfgPath: "m3u_merge_astra.yaml",
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
