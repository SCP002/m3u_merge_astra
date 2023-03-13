package tw

import (
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert.NotNil(t, New(), "should initialize table writer")
}

func TestRender(t *testing.T) {
	w := New()

	w.AppendHeader(table.Row{"Column 1", "Column 2"})
	w.AppendRow(table.Row{"Data 1 " + strings.Repeat("x", 150), "Data 2"})
	w.Render()

	w.AppendHeader(table.Row{"Column 3", "Column 4"})
	w.AppendRow(table.Row{"Data 3" + strings.Repeat("x", 150), "Data 4 " + strings.Repeat("x", 150)})
	w.Render()
}
