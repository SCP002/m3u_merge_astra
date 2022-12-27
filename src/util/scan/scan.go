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

// Lines returns true for every line of text in the data given to Scanner
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

		if strings.Trim(s.Line, "\r\n") == "" {
			s.LineEndIdx = s.RuneIdx
			s.LineStartIdx = s.RuneIdx
			s.Line = ""
			if s.RuneIdx == len(s.data)-1 {
				return false
			}
			continue
		}
		if s.RuneIdx == len(s.data)-1 {
			s.LineEndIdx = s.RuneIdx
			s.done = true
			return true
		}
		if char == '\n' {
			s.LineEndIdx = s.RuneIdx
			return true
		}
	}
}
