package relayd

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gitlab.33.cn/chain33/chain33/common/merkle"
)

type suiteBctd struct {
	// Include our basic suite logic.
	suite.Suite
	btc BtcClient
}

func TestRunSuiteBtcd(t *testing.T) {
	btcd := new(suiteBctd)
	suite.Run(t, btcd)
}

func (s *suiteBctd) SetupSuite() {
	reconnectAttempts := 3
	certs, _ := ioutil.ReadFile(filepath.Join("./", "rpc.cert"))
	connCfg := &rpcclient.ConnConfig{
		Host:         "127.0.0.1:18556",
		User:         "root",
		Pass:         "1314",
		HTTPPostMode: true,
		Certificates: certs,
	}
	s.btc, _ = NewBtcd(connCfg, reconnectAttempts)
}

func (s *suiteBctd) TestGetBlockHeader() {
	blockZeroHeader, err := s.btc.GetBlockHeader(0)
	s.NotNil(err)
	s.T().Log(blockZeroHeader)
}

func (s *suiteBctd) TestGetLatestBlock() {
	latestBLock, height, err := s.btc.GetLatestBlock()
	s.NotNil(err)
	s.Nil(latestBLock)
	s.T().Log(latestBLock)
	s.T().Log(height)
}

func (s *suiteBctd) TestGetTransaction() {
	tx, err := s.btc.GetTransaction("6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4")
	s.NotNil(err)
	s.Nil(tx)
}

func (s *suiteBctd) TestGetSPV() {
	spv, err := s.btc.GetSPV(22448, "aad85f52da28f808822aadfee72b8df23e2591a22ea5ef3cbc6592681a4baa2e")
	s.NotNil(err)
	s.Nil(spv)
}

func TestOneTxMerkle(t *testing.T) {
	tx0string := "b86f5ef1da8ddbdb29ec269b535810ee61289eeac7bf2b2523b494551f03897c"
	tx0hash, err := merkle.NewHashFromStr(tx0string)
	assert.Nil(t, err)
	tx0byte := tx0hash.CloneBytes()
	leaves := make([][]byte, 1)
	leaves[0] = tx0byte
	t.Log(leaves)
	bitHash := merkle.GetMerkleRoot(leaves)
	t.Log(bitHash)
	hash := merkle.GetMerkleBranch(leaves, 0)
	t.Log(hash)
}
