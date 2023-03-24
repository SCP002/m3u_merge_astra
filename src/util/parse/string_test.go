package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetIndent(t *testing.T) {
	assert.Exactly(t, 0, GetIndent(""), "should return that amount of initial space characters")
	assert.Exactly(t, 0, GetIndent("a b"), "should return that amount of initial space characters")
	assert.Exactly(t, 1, GetIndent(" a"), "should return that amount of initial space characters")
	assert.Exactly(t, 1, GetIndent(" a b  "), "should return that amount of initial space characters")
	assert.Exactly(t, 3, GetIndent("   "), "should return that amount of initial space characters")
	assert.Exactly(t, 0, GetIndent("	a"), "should return that amount of initial space characters") // Tab
}

func TestLastPathItem(t *testing.T) {
	assert.Exactly(t, "", LastPathItem("", ""))
	assert.Exactly(t, "a.b", LastPathItem("a.b", ""))
	assert.Exactly(t, "", LastPathItem("", "."))
	assert.Exactly(t, ".", LastPathItem(".", "."))
	assert.Exactly(t, "a", LastPathItem(".a", "."))
	assert.Exactly(t, "b", LastPathItem("a.b", "."))
	assert.Exactly(t, "c", LastPathItem("a.b..c", "."))
	assert.Exactly(t, "a.b.c", LastPathItem("a.b.c", "/"))
}
