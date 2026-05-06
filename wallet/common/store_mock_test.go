package common

import (
	"testing"

	dbmocks "github.com/33cn/chain33/common/db/mocks"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestStoreNewAndClose(t *testing.T) {
	mockDB := new(dbmocks.DB)
	mockBatch := new(dbmocks.Batch)
	mockDB.On("NewBatch", true).Return(mockBatch)
	mockDB.On("Close").Return()

	store := NewStore(mockDB)
	assert.NotNil(t, store)
	assert.Equal(t, mockDB, store.GetDB())
	store.Close()
	mockDB.AssertCalled(t, "Close")
}

func TestStoreGetDB(t *testing.T) {
	mockDB := new(dbmocks.DB)
	mockBatch := new(dbmocks.Batch)
	mockDB.On("NewBatch", true).Return(mockBatch)

	store := NewStore(mockDB)
	assert.Equal(t, mockDB, store.GetDB())
}

func TestStoreNewBatch(t *testing.T) {
	mockDB := new(dbmocks.DB)
	mockBatch := new(dbmocks.Batch)
	mockDB.On("NewBatch", true).Return(mockBatch)
	mockDB.On("NewBatch", false).Return(mockBatch)

	store := NewStore(mockDB)
	b := store.NewBatch(false)
	assert.NotNil(t, b)
}

func TestStoreHasSeedNoSeed(t *testing.T) {
	mockDB := new(dbmocks.DB)
	mockBatch := new(dbmocks.Batch)
	mockDB.On("NewBatch", true).Return(mockBatch)
	mockDB.On("Get", CalcWalletSeed()).Return([]byte{}, types.ErrSeedExist)

	store := NewStore(mockDB)
	has, _ := store.HasSeed()
	assert.False(t, has)
}
