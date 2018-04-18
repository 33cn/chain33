package relay

import (
	"gitlab.33.cn/chain33/chain33/common"
	"strings"

	"bytes"
)

//def helperVerifyHash__(txHash:uint256, txIndex, sibling:arr, txBlockHash):
//	if !self.feePaid(txBlockHash, m_getFeeAmount(txBlockHash), value=msg.value):  # in incentive.se
//	log(type=VerifyTransaction, txHash, ERR_BAD_FEE)
//	return(ERR_BAD_FEE)
//
//	if self.within6Confirms(txBlockHash):
//	log(type=VerifyTransaction, txHash, ERR_CONFIRMATIONS)
//	return(ERR_CONFIRMATIONS)
//
//	if !self.priv_inMainChain__(txBlockHash):
//	log(type=VerifyTransaction, txHash, ERR_CHAIN)
//	return(ERR_CHAIN)
//
//	merkle = self.computeMerkle(txHash, txIndex, sibling)
//	realMerkleRoot = getMerkleRoot(txBlockHash)
//
//	if merkle == realMerkleRoot:
//	log(type=VerifyTransaction, txHash, 1)
//	return(1)
//
//	log(type=VerifyTransaction, txHash, ERR_MERKLE_ROOT)
//	return(ERR_MERKLE_ROOT)
//

func GetMerkleRootFromHeader(blockhash string) {

}

func getRawTxHash(rawtx string) []byte {
	data, _ := common.FromHex(rawtx)
	h := common.DoubleHashH(data)
	return h.Bytes()
}

func getSiblingHash(sibling string) [][]byte {
	sibsarr := strings.Split(sibling, "-")

	sibs := make([][]byte, len(sibsarr))
	for i, val := range sibsarr {
		data, _ := common.FromHex(val)
		sibs[i] = common.BytesToHash(data).Revers().Bytes()
	}

	return sibs[:][:]

}

//rawtx, txindex, sibling, blockhash
//
//sibling like "aaaaaa-bbbbbb-cccccc..."

func BTCVerifyTx(verify *types.RelayVerifyBTC) (bool, error) {
	rawhash := getRawTxHash(verify.rawtx)
	sibs := getSiblingHash(verify.sibling)

	verifymerkleroot := merkle.GetMerkleRootFromBranch(sibs, rawhash, verify.txindex)
	realmerkleroot := GetMerkleRootFromHeader(verify.blockhash)
	return bytes.Equal(realmerkleroot, verifymerkleroot), nil

}
