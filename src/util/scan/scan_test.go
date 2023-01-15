package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert.Exactly(t, &Scanner{data: []rune{'a'}, RuneIdx: 1}, New([]rune{'a'}, 1), "should initialize scanner")
}

func TestLines(t *testing.T) {
	var lines []string
	var startIdxList []int
	var endIdxList []int

	reset := func() {
		lines = []string{}
		startIdxList = []int{}
		endIdxList = []int{}
	}

	collect := func(s *Scanner) {
		lines = append(lines, s.Line)
		startIdxList = append(startIdxList, s.LineStartIdx)
		endIdxList = append(endIdxList, s.LineEndIdx)
	}

	sc := New([]rune("line 1\r\n" + "\r\n" + "  line 2\n" + "line 3"), 0)
	for sc.Lines(true) {
		collect(sc)
	}
	assert.Exactly(t, []string{"line 1\r\n", "  line 2\n", "line 3"}, lines, "should collect these lines")
	assert.Exactly(t, []int{0, 9, 18}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{7, 18, 24}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 24, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("line 1\r\n" + "\r\n" + "  line 2\n" + "line 3"), 0)
	for sc.Lines(false) {
		collect(sc)
	}
	assert.Exactly(t, []string{"line 1\r\n", "\r\n", "  line 2\n", "line 3"}, lines, "should collect these lines")
	assert.Exactly(t, []int{0, 7, 9, 18}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{7, 9, 18, 24}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 24, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("line 1"), 6)
	for sc.Lines(true) {
		collect(sc)
	}
	assert.Exactly(t, []string{""}, lines, "should collect these lines")
	assert.Exactly(t, []int{6}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{6}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 6, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("line 1"), 6)
	for sc.Lines(false) {
		collect(sc)
	}
	assert.Exactly(t, []string{""}, lines, "should collect these lines")
	assert.Exactly(t, []int{6}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{6}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 6, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("line 1"), 3)
	for sc.Lines(true) {
		collect(sc)
	}
	assert.Exactly(t, []string{"e 1"}, lines, "should collect these lines")
	assert.Exactly(t, []int{3}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{5}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 5, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("line 1"), 3)
	for sc.Lines(false) {
		collect(sc)
	}
	assert.Exactly(t, []string{"e 1"}, lines, "should collect these lines")
	assert.Exactly(t, []int{3}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{5}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 5, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("line 1\n" + "line 2"), 13)
	for sc.Lines(true) {
		collect(sc)
	}
	assert.Exactly(t, []string{""}, lines, "should collect these lines")
	assert.Exactly(t, []int{13}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{13}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 13, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("line 1\n" + "line 2"), 13)
	for sc.Lines(false) {
		collect(sc)
	}
	assert.Exactly(t, []string{""}, lines, "should collect these lines")
	assert.Exactly(t, []int{13}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{13}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 13, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune(""), 100)
	for sc.Lines(true) {
		collect(sc)
	}
	assert.Exactly(t, []string{""}, lines, "should collect these lines")
	assert.Exactly(t, []int{100}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{100}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 100, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune(""), 100)
	for sc.Lines(false) {
		collect(sc)
	}
	assert.Exactly(t, []string{""}, lines, "should collect these lines")
	assert.Exactly(t, []int{100}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{100}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 100, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("\n"), 0)
	for sc.Lines(true) {
		collect(sc)
	}
	assert.Exactly(t, []string{""}, lines, "should collect these lines")
	assert.Exactly(t, []int{0}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{0}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 0, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("\n"), 0)
	for sc.Lines(false) {
		collect(sc)
	}
	assert.Exactly(t, []string{"\n"}, lines, "should collect these lines")
	assert.Exactly(t, []int{0}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{0}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 0, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("line 1\n" + "  line 2\n" + "line 3\r\n" + "\r\n"), 6)
	for sc.Lines(true) {
		collect(sc)
	}
	assert.Exactly(t, []string{"  line 2\n", "line 3\r\n"}, lines, "should collect these lines")
	assert.Exactly(t, []int{6, 15}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{15, 23}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 25, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("line 1\n" + "  line 2\n" + "line 3\r\n" + "\r\n"), 6)
	for sc.Lines(false) {
		collect(sc)
	}
	assert.Exactly(t, []string{"  line 2\n", "line 3\r\n", "\r\n"}, lines, "should collect these lines")
	assert.Exactly(t, []int{6, 15, 23}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{15, 23, 25}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 25, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("  \n" + "  \r\n"), 0)
	for sc.Lines(true) {
		collect(sc)
	}
	assert.Exactly(t, []string{"  \n", "  \r\n"}, lines, "should collect these lines")
	assert.Exactly(t, []int{0, 2}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{2, 6}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 6, sc.RuneIdx, "scanner should read that amount of characters")

	reset()
	sc = New([]rune("  \n" + "  \r\n"), 0)
	for sc.Lines(false) {
		collect(sc)
	}
	assert.Exactly(t, []string{"  \n", "  \r\n"}, lines, "should collect these lines")
	assert.Exactly(t, []int{0, 2}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{2, 6}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 6, sc.RuneIdx, "scanner should read that amount of characters")
}
