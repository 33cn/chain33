package table

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestTransactinList(t *testing.T) {
	rowMeta := &TransactionRow{}
	table, err := NewTable(rowMeta, "Hash", []string{"From", "To"})
	assert.Nil(t, err)
	err = table.Add(&types.Transaction{})
	assert.Nil(t, err)
	err = table.Add(&types.Transaction{})
	assert.Nil(t, err)
	kvs, err := table.Save()
	assert.Nil(t, err)
	assert.NotNil(t, kvs)
}
