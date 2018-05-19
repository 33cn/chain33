package relay

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/types"

	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	lastheaderlock sync.Mutex
)

type relayBTCStore struct {
	db         dbm.KVDB
	height     int64
	lastheader *types.BtcHeader
}

//func (r *relayBTC) saveHighestHead(header *types.BtcHeader) {
//	if r.blockHeight < header.Height {
//		r.blockHeight = header.Height
//		r.header = header
//	}
//}

func (b *relayBTCStore) setDb(db dbm.KVDB) {
	b.db = db
}

func (b *relayBTCStore) saveBlockHead(head *types.BtcHeader) []*types.KeyValue {
	var kv []*types.KeyValue
	var key []byte

	val, err := proto.Marshal(head)
	if err != nil {
		relaylog.Error("relayBTCStore Marshal header", "height", head.Height, "hash", head.Hash, "error", err)

	} else {
		//hash:header
		key = getRelayBtcHeaderKeyHash(head.Hash)
		kv = append(kv, &types.KeyValue{key, val})
		//height:header
		key = getRelayBtcHeaderKeyHeight(int64(head.Height))
		kv = append(kv, &types.KeyValue{key, val})
	}

	//prefix-height:height
	key = getRelayBtcHeaderKeyHeightList(int64(head.Height))
	heightbytes := types.Encode(&types.Int64{int64(head.Height)})
	kv = append(kv, &types.KeyValue{key, heightbytes})
	//last height
	key = getRelayBtcHeaderKeyLastHeight()
	kv = append(kv, &types.KeyValue{key, heightbytes})

	//global
	atomic.StoreInt64(&b.height, int64(head.Height))
	b.lastheader = head

	return kv
}

func (b *relayBTCStore) curHeight() int64 {
	return atomic.LoadInt64(&b.height)
}

func (b *relayBTCStore) getHeadHeigtList(req *types.ReqRelayBtcHeaderHeightList) (types.Message, error) {
	prefix := getRelayBtcHeightListKey()
	key := fmt.Sprintf(string(prefix)+"%d", req.HeightBase)

	values, err := b.db.List(prefix, []byte(key), req.ReqCounts, 0)
	if err != nil {
		return nil, err
	}

	var replay types.ReplyBtcHeaderHeightList
	heightGot := make(map[string]bool)
	for _, height := range values {
		if !heightGot[string(height)] {
			replay.Heights = append(replay.Heights, height)
			heightGot[string(height)] = true
		}
	}

	return &replay, nil

}

func (b *relayBTCStore) GetMerkleRootFromHeader(blockhash string) (string, error) {
	value, err := b.db.Get(getRelayBtcHeaderKeyHash(blockhash))
	if err != nil {
		relaylog.Error("GetMerkleRootFromHeader", "Failed to get value from db with blockhash", blockhash)
		return "", err
	}

	var header types.BtcHeader
	if err = types.Decode(value, &header); err != nil {
		relaylog.Error("GetMerkleRootFromHeader", "Failed to decode head", blockhash)
		return "", err
	}

	return header.MerkleRoot, nil

}

func (b *relayBTCStore) verifyTx(verify *types.RelayVerify, order *types.RelayOrder) (bool, error) {
	var foundtx bool
	for _, outtx := range verify.GetTx().GetVout() {
		relaylog.Debug("verifyTx", "outtx addr", outtx.Address, "order addr", order.Exchgaddr)
		if outtx.Address == order.Exchgaddr && outtx.Value == order.Exchgamount {
			foundtx = true
		}
	}

	if !foundtx {
		relaylog.Error("verifyTx", "Failed to get order", order.Orderid, "tx", order.Exchgaddr)
		return false, types.ErrTRelayVrfAddrNotFound
	}

	txhash, _ := common.FromHex(verify.GetTx().GetHash())
	rawhash := common.BytesToHash(txhash).Revers().Bytes()
	sibs := verify.GetSpv().GetBranchProof()

	verifymerkleroot := merkle.GetMerkleRootFromBranch(sibs, rawhash, verify.GetSpv().GetTxIndex())
	val, err := b.GetMerkleRootFromHeader(verify.GetSpv().GetBlockHash())
	if err != nil {
		return false, err
	}
	realmerkleroot, _ := common.FromHex(val)
	return bytes.Equal(realmerkleroot, verifymerkleroot), nil

}

//rawtx, txindex, sibling, blockhash
//
//sibling like "aaaaaa-bbbbbb-cccccc..."

func (b *relayBTCStore) verifyBTCTx(verify *types.RelayVerifyBTC) (bool, error) {
	rawhash := getRawTxHash(verify.Rawtx)
	sibs := getSiblingHash(verify.Merkbranch)

	verifymerkleroot := merkle.GetMerkleRootFromBranch(sibs, rawhash, verify.Txindex)
	val, err := b.GetMerkleRootFromHeader(verify.Blockhash)
	if err != nil {
		return false, err
	}
	realmerkleroot, _ := common.FromHex(val)
	return bytes.Equal(realmerkleroot, verifymerkleroot), nil

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
