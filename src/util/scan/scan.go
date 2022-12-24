package scan

import (
	"strings"
)

// Scanner represents rune slice scanner
type Scanner struct {
	data         []rune
	done         bool
	RuneIdx      int    // Index of the latest rune found
	Line         string
	LineStartIdx int
	LineEndIdx   int
}

// New returns new scanner for <data>, starting from the <startIdx>
func New(data []rune, startIdx int) *Scanner {
	return &Scanner{data: data, RuneIdx: startIdx}
}

// Lines returns true for every line of text in the data given to Scanner
func (s *Scanner) Lines() bool {
	s.Line = ""

	for ; ; s.RuneIdx++ {
		if s.done {
			return false
		}
		// No more data, build the final line
		if s.RuneIdx == len(s.data)-1 {
			s.done = true
			s.LineEndIdx = s.RuneIdx
			s.Line = string(s.data[s.LineStartIdx+1:])
			return strings.Trim(s.Line, "\r\n") != ""
		}

		char := s.data[s.RuneIdx]
		s.Line += string(char)

		if char != '\n' {
			continue
		}
		// Line break character at index from previous run or empty line
		if strings.Trim(s.Line, "\r\n") == "" {
			s.LineStartIdx = s.RuneIdx
			s.Line = ""
			continue
		}

		s.LineEndIdx = s.RuneIdx
		return true
	}
}
