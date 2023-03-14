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
