package rnd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	uppercaseCharset := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercaseCharset := "abcdefghijklmnopqrstuvwxyz"
	numbersCharset := "0123456789"

	s := String(0, false, false)
	assert.Exactly(t, "", s, "should return empty string")

	s = String(20, false, false)
	assert.Len(t, s, 20, "should be 20 symbols long")
	assert.False(t, strings.ContainsAny(s, uppercaseCharset), "should not contain uppercase characters")
	assert.True(t, strings.ContainsAny(s, lowercaseCharset), "should contain lowercase characters")
	assert.False(t, strings.ContainsAny(s, numbersCharset), "should not contain numbers")

	s = String(20, false, true)
	assert.Len(t, s, 20, "should be 20 symbols long")
	assert.False(t, strings.ContainsAny(s, uppercaseCharset), "should not contain uppercase characters")
	assert.True(t, strings.ContainsAny(s, lowercaseCharset), "should contain lowercase characters")
	assert.True(t, strings.ContainsAny(s, numbersCharset), "should contain numbers")

	s = String(20, true, false)
	assert.Len(t, s, 20, "should be 20 symbols long")
	assert.True(t, strings.ContainsAny(s, uppercaseCharset), "should contain uppercase characters")
	assert.True(t, strings.ContainsAny(s, lowercaseCharset), "should contain lowercase characters")
	assert.False(t, strings.ContainsAny(s, numbersCharset), "should not contain numbers")

	s = String(20, true, true)
	assert.Len(t, s, 20, "should be 20 symbols long")
	assert.True(t, strings.ContainsAny(s, uppercaseCharset), "should contain uppercase characters")
	assert.True(t, strings.ContainsAny(s, lowercaseCharset), "should contain lowercase characters")
	assert.True(t, strings.ContainsAny(s, numbersCharset), "should contain numbers")
}
