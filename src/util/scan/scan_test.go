package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert.Exactly(t, &Scanner{data: []rune{'a'}, RuneIdx: 1}, New([]rune{'a'}, 1), "should initialize scanner")
}

func TestLines(t *testing.T) {
	input := []rune("line 1\r\n" + "\r\n" + "  line 2\n" + "line 3")
	var lines []string
	var startIdxList []int
	var endIdxList []int

	scanner := New(input, 0)
	for scanner.Lines() {
		lines = append(lines, scanner.Line)
		startIdxList = append(startIdxList, scanner.LineStartIdx)
		endIdxList = append(endIdxList, scanner.LineEndIdx)
	}

	assert.Exactly(t, []string{"line 1\r\n", "  line 2\n", "line 3"}, lines, "should collect these lines")
	assert.Exactly(t, []int{0, 9, 18}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{7, 18, 24}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 24, scanner.RuneIdx, "scanner should read that amount of characters")

	input = []rune("line 1\n" + "  line 2\n" + "line 3\r\n" + "\r\n")
	lines = []string{}
	startIdxList = []int{}
	endIdxList = []int{}

	scanner = New(input, 6)
	for scanner.Lines() {
		lines = append(lines, scanner.Line)
		startIdxList = append(startIdxList, scanner.LineStartIdx)
		endIdxList = append(endIdxList, scanner.LineEndIdx)
	}

	assert.Exactly(t, []string{"  line 2\n", "line 3\r\n"}, lines, "should collect these lines")
	assert.Exactly(t, []int{6, 15}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{15, 23}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 25, scanner.RuneIdx, "scanner should read that amount of characters")

	input = []rune("  \n" + "  \r\n")
	lines = []string{}
	startIdxList = []int{}
	endIdxList = []int{}

	scanner = New(input, 0)
	for scanner.Lines() {
		lines = append(lines, scanner.Line)
		startIdxList = append(startIdxList, scanner.LineStartIdx)
		endIdxList = append(endIdxList, scanner.LineEndIdx)
	}

	assert.Exactly(t, []string{"  \n", "  \r\n"}, lines, "should collect these lines")
	assert.Exactly(t, []int{0, 2}, startIdxList, "should collect these line start indexes")
	assert.Exactly(t, []int{2, 6}, endIdxList, "should collect these line end indexes")
	assert.Exactly(t, 6, scanner.RuneIdx, "scanner should read that amount of characters")
}
