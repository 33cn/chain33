package relayd

import (
	"testing"
)

func TestNewBTCWeb(t *testing.T) {
	btc := NewBTCWeb()
	blockZero, err := btc.GetBlock(0)
	if err != nil {
		t.Errorf("GetBlock error: %v", err)
	}
	t.Log(blockZero)

	blockZeroHeader, err := btc.GetBlockHeader(0)
	if err != nil {
		t.Errorf("GetBlockHeader error: %v", err)
	}
	t.Log(blockZeroHeader)

	latestBLock, err := btc.GetLatestBlock()
	if err != nil {
		t.Errorf("GetLatestBlock error: %v", err)
	}
	t.Log(latestBLock)

	spv, err := btc.GetSPV(100000, "8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87")
	if err != nil {
		t.Errorf("GetLatestBlock error: %v", err)
	}
	t.Log(spv)
}
