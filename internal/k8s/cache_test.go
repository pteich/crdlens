package k8s

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	c := NewCache(100 * time.Millisecond)

	// Test Set and Get
	c.Set("foo", "bar")
	val, ok := c.Get("foo")
	require.True(t, ok, "key should exist in cache")
	assert.Equal(t, "bar", val)

	// Test Expiration
	time.Sleep(150 * time.Millisecond)
	_, ok = c.Get("foo")
	assert.False(t, ok, "item should be expired")

	// Test Delete
	c.Set("baz", "qux")
	c.Delete("baz")
	_, ok = c.Get("baz")
	assert.False(t, ok, "item should be deleted")

	// Test Clear
	c.Set("a", 1)
	c.Set("b", 2)
	c.Clear()
	_, ok1 := c.Get("a")
	_, ok2 := c.Get("b")
	assert.False(t, ok1, "item a should be cleared")
	assert.False(t, ok2, "item b should be cleared")
}
