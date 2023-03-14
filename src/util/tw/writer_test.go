package tw

import (
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

func TestNew(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		w := New()
		w.AppendRow(table.Row{"Test"})
		w.Render()
	})
	assert.Contains(t, out, "Test")
}

func TestAppendHeader(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		w := New()
		w.AppendHeader(table.Row{"Column 1", "Column 2"})
		w.AppendRow(table.Row{strings.Repeat("x", 150), strings.Repeat("x", 150)})
		w.Render()
	})
	// Testing against terminal width in 120 characters as it's a fallback value: function calculating terminal width
	// always fail in tests.
	assert.Contains(t, out, "│ " + strings.Repeat("x", 56) + " │") // (120 / 2) - 2 * 2 = 60 - 4 = 56
	assert.Contains(t, out, "│ " + strings.Repeat("x", 38) + strings.Repeat(" ", 18) + " │") // 38 + 18 = 56
}

func TestRender(t *testing.T) {
	out := capturer.CaptureStderr(func() {
		w := New()

		w.AppendHeader(table.Row{"Column 1"})
		w.AppendRow(table.Row{"Data 1"})
		w.AppendFooter(table.Row{"Footer 1"})
		w.Render()

		w.AppendHeader(table.Row{"Column 2"})
		w.AppendRow(table.Row{"Data 2"})
		w.AppendFooter(table.Row{"Footer 2"})
		w.Render()
	})

	assert.Contains(t, out, strings.ToUpper("Column 1"))
	assert.Contains(t, out, "Data 1")
	assert.Contains(t, out, strings.ToUpper("Footer 1"))

	assert.Contains(t, out, strings.ToUpper("Column 2"))
	assert.Contains(t, out, "Data 2")
	assert.Contains(t, out, strings.ToUpper("Footer 2"))
}
