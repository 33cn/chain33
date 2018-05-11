package relayd

import (
	"testing"

	"io/ioutil"
	"path/filepath"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"gitlab.33.cn/chain33/chain33/common/merkle"
)

var (
	certHomeDir = btcutil.AppDataDir("btcd", false)
	// certHomeDir := "/home/suyanlong/.gvm/pkgsets/go1.9.4/global/src/github.com/btcsuite/btcd/cmd/gencerts"
	certs, _ = ioutil.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	connCfg  = &rpcclient.ConnConfig{
		Host:         "127.0.0.1:18556",
		User:         "suyanlong",
		Pass:         "1314",
		HTTPPostMode: true,  // Bitcoin core only supports HTTP POST mode
		DisableTLS:   false, // Bitcoin core does not provide TLS by default
		Certificates: certs,
	}
	reconnectAttempts = 3
)

func TestNewBtcd(t *testing.T) {
	btc, err := NewBtcd(connCfg, reconnectAttempts)
	if err != nil {
		t.Error(err)
	}

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

	spv, err := btc.GetSPV(22448, "aad85f52da28f808822aadfee72b8df23e2591a22ea5ef3cbc6592681a4baa2e")
	if err != nil {
		t.Errorf("GetSPV error: %v", err)
	}
	t.Logf("%+v", spv)
}

func Test_oneTxMerkle(t *testing.T) {
	tx0string := "b86f5ef1da8ddbdb29ec269b535810ee61289eeac7bf2b2523b494551f03897c"
	tx0hash, err := merkle.NewHashFromStr(tx0string)
	if err != nil {
		t.Errorf("NewHashFromStr tx0string err:%s", err.Error())
	}

	tx0byte := tx0hash.CloneBytes()
	leaves := make([][]byte, 1)
	leaves[0] = tx0byte
	// leaves[1] = tx0byte
	t.Log(leaves)
	bitroothash := merkle.GetMerkleRoot(leaves)
	t.Log(bitroothash)
	hash := merkle.GetMerkleBranch(leaves, 0)
	t.Log(hash)
}
