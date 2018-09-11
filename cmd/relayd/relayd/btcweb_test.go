package relayd

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type suiteWeb struct {
	// Include our basic suite logic.
	suite.Suite
	btc BtcClient
}

func TestRunSuiteWeb(t *testing.T) {
	web := new(suiteWeb)
	suite.Run(t, web)
}

func (s *suiteWeb) SetupSuite() {
	s.btc, _ = NewBtcWeb()
}

func (s *suiteWeb) TestGetBlockHeader() {
	blockZeroHeader, err := s.btc.GetBlockHeader(0)
	//s.Nil(err)
	s.T().Log(err)
	s.T().Log(blockZeroHeader)
}

func (s *suiteWeb) TestGetLatestBlock() {
	latestBLock, height, err := s.btc.GetLatestBlock()
	//s.Nil(err)
	//s.NotNil(latestBLock)
	s.T().Log(err)
	s.T().Log(latestBLock)
	s.T().Log(height)
}

func (s *suiteWeb) TestGetTransaction() {
	tx, err := s.btc.GetTransaction("6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4")
	//s.Nil(err)
	//s.NotNil(tx)
	s.T().Log(tx)
	s.T().Log(err)
}

func (s *suiteWeb) TestGetSPV() {
	spv, err := s.btc.GetSPV(22448, "aad85f52da28f808822aadfee72b8df23e2591a22ea5ef3cbc6592681a4baa2e")
	//s.Nil(err)
	//s.NotNil(spv)
	s.T().Log(err)
	s.T().Log(spv)
}
