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

	w.SetColumnConfigs([]table.ColumnConfig{{Number: 1, WidthMax: 10}, {Number: 2, WidthMax: 20}})
	w.AppendHeader(table.Row{"Column 1", "Column 2"})
	w.AppendRow(table.Row{"Data 1 " + strings.Repeat("x", 10), "Data 2"})
	w.Render()

	w.SetColumnConfigs([]table.ColumnConfig{{Number: 1, WidthMax: 20}, {Number: 2, WidthMax: 10}})
	w.AppendHeader(table.Row{"Column 3", "Column 4"})
	w.AppendRow(table.Row{"Data 3", "Data 4 " + strings.Repeat("x", 10)})
	w.Render()
}
