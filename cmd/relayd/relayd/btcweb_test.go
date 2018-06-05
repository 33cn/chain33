package relayd_test

import (
	"testing"

	. "gitlab.33.cn/chain33/chain33/cmd/relayd/relayd"
)

func TestNewBtcWeb(t *testing.T) {
	btc, _ := NewBtcWeb()
	blockZero, err := btc.GetBlockHeader(0)
	if err != nil {
		t.Errorf("getBlock error: %v", err)
	}
	t.Log(blockZero)

	blockZeroHeader, err := btc.GetBlockHeader(0)
	if err != nil {
		t.Errorf("GetBlockHeader error: %v", err)
	}
	t.Log(blockZeroHeader)

	latestBLock, height, err := btc.GetLatestBlock()
	if err != nil {
		t.Errorf("GetLatestBlock error: %v", err)
	}

	t.Log(latestBLock)
	t.Log(height)

	// 6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4
	tx, err := btc.GetTransaction("6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4")
	if err != nil {
		t.Errorf("GetTransaction error: %v", err)
	}
	t.Log(tx)

	spv, err := btc.GetSPV(100000, "8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87")
	if err != nil {
		t.Errorf("GetLatestBlock error: %v", err)
	}
	t.Log(spv)
}
