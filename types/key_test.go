package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyCreate(t *testing.T) {
	assert.Equal(t, TotalFeeKey([]byte("1")), []byte("TotalFeeKey:1"))
	assert.Equal(t, CalcLocalPrefix([]byte("1")), []byte("LODB-1-"))
	assert.Equal(t, CalcStatePrefix([]byte("1")), []byte("mavl-1-"))
	assert.Equal(t, CalcRollbackKey([]byte("execer"), []byte("1")), []byte("LODB-execer-rollback-1"))
	assert.Equal(t, StatisticFlag(), []byte("Statistics:Flag"))
}
