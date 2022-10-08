package astra

import (
	"os"
	"path/filepath"
	"testing"

	"m3u_merge_astra/cli"
	"m3u_merge_astra/util/copier"

	"github.com/stretchr/testify/assert"
)

func TestAddCategory(t *testing.T) {
	r := newDefRepo()

	cl1 := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}}},
	}
	cl1Original := copier.TDeep(t, cl1)

	cl2 := r.AddCategory(cl1, "Category 3", []string{"A", "B", "C"})

	assert.NotSame(t, &cl1, &cl2, "should return copy of categories")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")

	expected := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}}},
		{Name: "Category 3", Groups: []Group{{Name: "A"}, {Name: "B"}, {Name: "C"}}},
	}
	assert.Exactly(t, expected, cl2, "should add new category with the specified groups")

	cl2 = r.AddCategory(cl1, "Category 2", []string{"D", "C", "B", "A"})

	assert.NotSame(t, &cl1, &cl2, "should return copy of categories")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")

	expected = []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}, {Name: "B"}, {Name: "A"}}},
	}
	assert.Exactly(t, expected, cl2, "should add new groups to exising category")

	cl2 = r.AddCategory(cl1, "Category 1", []string{"", "B"})

	assert.NotSame(t, &cl1, &cl2, "should return copy of categories")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")

	assert.Exactly(t, cl1, cl2, "should not add empty or exising groups to existing category")
}

func TestWriteReadCfg(t *testing.T) {
	c1 := Cfg{
		Streams: []Stream{{ID: "0000"}},
		Unknown: map[string]any{
			"users": map[string]any{
				"user1": map[string]any{
					"enable": true,
				},
			},
			"gid": float64(111111),
		},
	}

	path := filepath.Join(os.TempDir(), "m3u_merge_astra_test_write_cfg.json")
	defer os.Remove(path)

	// Write to file
	err := WriteCfg(c1, path)
	assert.NoError(t, err, "should not return error")

	// Read from file
	c2, err := ReadCfg(path)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, c1, c2, "config should stay the same")

	// Write to clipboard
	err = WriteCfg(c1, string(cli.Clipboard))
	assert.NoError(t, err, "should not return error")

	// Read from clipboard
	c2, err = ReadCfg(string(cli.Clipboard))
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, c1, c2, "config should stay the same")

	// Redirect stdout to stdin for testing
	r, w, err := os.Pipe()
	assert.NoError(t, err, "should not return error")
	os.Stdin = r
	os.Stdout = w

	// Write to stdout
	err = WriteCfg(c1, string(cli.Stdio))
	w.Close()
	assert.NoError(t, err, "should not return error")

	// Read from stdin
	c2, err = ReadCfg(string(cli.Stdio))
	r.Close()
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, c1, c2, "config should stay the same")
}
