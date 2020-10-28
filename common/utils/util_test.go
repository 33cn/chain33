package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_spaceLimitCache(t *testing.T) {

	c := NewSpaceLimitCache(3, 10)
	assert.Equal(t, 3, c.capacity)
	assert.True(t, c.Add(1, 1, 1))
	assert.True(t, c.Add(1, 1, 1))
	assert.False(t, c.Add(2, 2, 20))
	assert.Nil(t, c.Get(2))
	assert.True(t, c.Add(2, 1, 10))
	c.Add(3, 2, 2)
	c.Add(4, 2, 2)
	c.Add(5, 2, 2)
	c.Add(6, 2, 1)
	assert.False(t, c.Contains(3))
	assert.Equal(t, 3, c.data.Len())
	assert.Equal(t, 3, len(c.sizeMap))
	assert.Equal(t, 5, c.currSize)
	assert.True(t, c.Contains(4))
	assert.True(t, c.Contains(5))
	assert.True(t, c.Add(7, 7, 8))
	assert.True(t, c.Contains(7))
	assert.Equal(t, 2, c.data.Len())
	assert.Equal(t, 2, len(c.sizeMap))
	assert.Equal(t, 9, c.currSize)
	_, exist := c.Remove(6)
	assert.True(t, exist)
	_, exist = c.Remove(5)
	assert.False(t, exist)
	assert.Equal(t, 8, c.currSize)
}
