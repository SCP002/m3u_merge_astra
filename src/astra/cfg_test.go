package astra

import (
	"os"
	"path/filepath"
	"testing"

	"m3u_merge_astra/cli"
	"m3u_merge_astra/util/copier"

	"github.com/stretchr/testify/assert"
)

func TestAddNewGroups(t *testing.T) {
	r := newDefRepo()

	cl1 := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}}},
	}
	cl1Original := copier.TDeep(t, cl1)
	sl1 := []Stream{
		{Groups: map[string]string{"Category 3": "A"}},
		{Groups: map[string]string{"Category 3": "B"}},
		{Groups: map[string]string{"Category 3": "B"}},
		{Groups: map[string]string{"Category 1": ""}},
		{Groups: map[string]string{"Category 1": "B"}},
		{Groups: map[string]string{"Category 2": "D"}},
		{Groups: map[string]string{"Category 2": "C"}},
		{Groups: map[string]string{"Category 2": "B"}},
		{Groups: map[string]string{"Category 2": "A"}},
	}
	sl1Original := copier.TDeep(t, sl1)

	cl2 := r.AddNewGroups(cl1, sl1)

	assert.NotSame(t, &cl1, &cl2, "should return copy of categories")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")

	expected := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}, {Name: "B"}, {Name: "A"}}},
		{Name: "Category 3", Groups: []Group{{Name: "A"}, {Name: "B"}}},
	}
	assert.Exactly(t, expected, cl2, "should add new category with the specified groups")
}

func TestWriteReadCfg(t *testing.T) {
	c1 := Cfg{
		Streams: []Stream{
			{
				ID: "0000",
				Groups: map[string]string{
					"Category 1": "Group 1",
					"Category 2": "Group 2",
				},
			},
		},
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
