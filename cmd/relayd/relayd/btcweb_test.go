package relayd

import (
	"testing"
)

func TestNewBtcWeb(t *testing.T) {
	btc := NewBtcWeb()
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

	spv, err := btc.GetSPV(100000, "8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87")
	if err != nil {
		t.Errorf("GetLatestBlock error: %v", err)
	}
	t.Log(spv)
}
