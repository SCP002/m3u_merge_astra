package scan

import (
	"strings"
)

// Scanner represents rune slice scanner
type Scanner struct {
	data         []rune
	done         bool
	RuneIdx      int // Index of the latest rune found
	Line         string
	LineStartIdx int
	LineEndIdx   int
}

// New returns new scanner for <data>, starting from the <startIdx>
func New(data []rune, startIdx int) *Scanner {
	return &Scanner{data: data, RuneIdx: startIdx}
}

// Lines returns true for every line of text in the data given to Scanner.
//
// Unlike bufio.Scanner, it does not trim /r, /n and space character from line.
func (s *Scanner) Lines() bool {
	s.Line = ""
	s.LineStartIdx = s.RuneIdx
	s.LineEndIdx = s.RuneIdx

	for ; ; s.RuneIdx++ {
		if s.done {
			return false
		}
		if s.RuneIdx > len(s.data)-1 {
			s.done = true
			return true
		}

		char := s.data[s.RuneIdx]
		s.Line += string(char)
		s.LineEndIdx = s.RuneIdx

		// Line break character at index from previous run or empty line, continue
		if strings.Trim(s.Line, "\r\n") == "" {
			s.LineStartIdx = s.RuneIdx
			s.Line = ""
			if s.RuneIdx == len(s.data)-1 {
				return false
			}
			continue
		}
		if s.RuneIdx == len(s.data)-1 {
			s.done = true
			return true
		}
		if char == '\n' {
			return true
		}
	}
}
