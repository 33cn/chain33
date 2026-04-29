package blockchain

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestIsRecordFaultErr(t *testing.T) {
	assert.True(t, IsRecordFaultErr(types.ErrActionNotSupport))
	assert.False(t, IsRecordFaultErr(types.ErrFutureBlock))
	assert.False(t, IsRecordFaultErr(nil))
}

func TestNoneCacheMethods(t *testing.T) {
	n := &noneCache{}
	n.Add(nil)
	n.Del(0)
	assert.False(t, n.Contains(nil, 0))
}
