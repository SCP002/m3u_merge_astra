package input

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"m3u_merge_astra/util/logger"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
)

// AskYesNo returns true if user input is 'y' or 'Y' and false if 'n' or 'N'.
//
// If user types neither 'y', 'Y', 'n' or 'N', it asks again.
//
// <in> is a reader to read data from. Usually it should be 'os.Stdin'.
func AskYesNo(log *logger.Logger, in io.Reader, prompt string) bool {
	answer := ask(log, in, true, prompt, func(input string) bool {
		switch input {
		case "y", "Y", "n", "N":
			return false
		}
		return true
	})
	return lo.Ternary(strings.ToLower(answer) == "y", true, false)
}

// ask returns user input, preliminarily printing <prompt>.
//
// It runs forever until read is successful and <callback> returns false.
//
// If <trim> is true, trim space from user input before passing it to <callback>.
//
// <in> is a reader to read data from. Usually it should be 'os.Stdin'.
func ask(log *logger.Logger, in io.Reader, trim bool, prompt string, callback func(string) bool) string {
	for {
		fmt.Print(prompt)
		reader := bufio.NewReader(in)
		input, err := reader.ReadString('\n')
		if trim {
			input = strings.TrimSpace(input)
		}
		if callback(input) {
			continue
		}
		if err == nil {
			return input
		}
		log.Error(errors.Wrap(err, "Read from standard input"))
	}
}
