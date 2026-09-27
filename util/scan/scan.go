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
// If <skipEmpty> is true, do not return true for the blank lines (/n, /r/n).
//
// Unlike bufio.Scanner, it does not trim /r, /n and space characters from line.
func (s *Scanner) Lines(skipEmpty bool) bool {
	s.Line = ""
	s.LineStartIdx = s.RuneIdx
	s.LineEndIdx = s.RuneIdx

	lastIdx := len(s.data) - 1

	// On previous call rune index stopped at \n, advance
	if s.RuneIdx < lastIdx && s.data[s.RuneIdx] == '\n' {
		s.RuneIdx++
	}

	for ; ; s.RuneIdx++ {
		if s.done {
			return false
		}
		if s.RuneIdx > lastIdx {
			s.done = true
			return true
		}

		char := s.data[s.RuneIdx]
		s.Line += string(char)
		s.LineEndIdx = s.RuneIdx

		// Empty line and should skip them
		if skipEmpty && strings.Trim(s.Line, "\r\n") == "" {
			s.LineStartIdx = s.RuneIdx
			s.Line = ""
			// Return something once if size of <data> is 1 character
			if len(s.data) == 1 {
				s.done = true
				return true
			}
			// Prevent returning empty line and it's boundary indexes
			if s.RuneIdx == lastIdx {
				return false
			}
			continue
		}
		if s.RuneIdx == lastIdx {
			s.done = true
			return true
		}
		if char == '\n' {
			return true
		}
	}
}
