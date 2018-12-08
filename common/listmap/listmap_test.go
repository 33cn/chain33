package listmap

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestListMap(t *testing.T) {
	l := New()
	l.Push("1", "1")
	assert.Equal(t, 1, l.Size())
	l.Push("1", "11")
	assert.Equal(t, 1, l.Size())
	value, err := l.GetItem("1")
	assert.Equal(t, err, nil)
	assert.Equal(t, "11", value.(string))
	l.Remove("1")
	assert.Equal(t, 0, l.Size())
	assert.Equal(t, false, l.Exist("1"))

	v := l.GetTop()
	assert.Equal(t, nil, v)

	_, err = l.GetItem("11")
	assert.Equal(t, types.ErrNotFound, err)
	l.Push("1", "11")
	assert.Equal(t, true, l.Exist("1"))
	l.Push("2", "2")
	assert.Equal(t, "11", l.GetTop().(string))

	var data [2]string
	i := 0
	l.Walk(func(value interface{}) bool {
		data[i] = value.(string)
		i++
		return true
	})
	assert.Equal(t, data[0], "11")
	assert.Equal(t, data[1], "2")

	var data2 [2]string
	i = 0
	l.Walk(func(value interface{}) bool {
		data2[i] = value.(string)
		i++
		return false
	})
	assert.Equal(t, data2[0], "11")
	assert.Equal(t, data2[1], "")

	l.Walk(nil)
	l.Remove("xxxx")
}
