package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTradeType_GetName(t *testing.T) {
	tp := NewType()
	assert.Equal(t, TradeX, tp.GetName())
}

func TestTradeType_GetTypeMap(t *testing.T) {
	tp := NewType()
	actoins := tp.GetTypeMap()
	assert.NotNil(t, actoins)
	assert.NotEqual(t, 0, len(actoins))
}

func TestTradeType_GetLogMap(t *testing.T) {
	tp := NewType()
	l := tp.GetLogMap()
	assert.NotNil(t, l)
	assert.NotEqual(t, 0, len(l))
}
