package relay

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/types"

	"bytes"
	"encoding/json"
	"sync"
	"sync/atomic"
)

var (
	lastheaderlock sync.Mutex
)

type relayBTCStore struct {
	r *relay
	//db         	dbm.KVDB
	height     int64
	lastheader *types.BtcHeader
}

func (b *relayBTCStore) new(r *relay) {
	b.r = r
	b.height = -1
	b.lastheader = nil
}

func getBTCHeadHeightFromDb(r *relay, key []byte) (int64, error) {
	val, err := r.GetLocalDB().Get(key)
	if val == nil || err != nil {
		relaylog.Info("relayBTCStore get BTC store", "error", err)
		return -1, err
	}
	height, err := decodeHeight(val)
	if err != nil {
		relaylog.Error("relayBTCStore decode height fail", "error", err)
		return -1, err
	}

	return height, nil
}

func (b *relayBTCStore) checkBlockHead(head *types.BtcHeader) bool {

	if head.Flag == types.RelaySaveBTCHeadFlag_resetHead {
		relaylog.Info("relayBTCStore checkBlockHead head flag=reset", "reset height", head.Height)
		return true
	}
	// generis height or restart (if node restart, the height init to -1, new sync tx with block will be save directly)
	if b.height <= 0 || b.lastheader == nil {
		relaylog.Info("relayBTCStore checkBlockHead", "height", b.height)
		return true
	}

	if head.PreviousHash == b.lastheader.Hash && head.Height == b.lastheader.Height+1 {
		return true
	}

	relaylog.Error("relayBTCStore checkBlockHead fail", "last height", b.lastheader.Height, "check head height", head.Height,
		"last hash", b.lastheader.Hash, "check pre hash", head.PreviousHash)
	return false
}

func (b *relayBTCStore) saveBlockHead(head *types.BtcHeader) []*types.KeyValue {
	var kv []*types.KeyValue
	var key []byte

	if !b.checkBlockHead(head) {
		return kv
	}

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

	//for start with height =-1 case, the base not be set, just return -1
	if head.Flag == types.RelaySaveBTCHeadFlag_resetHead {
		key = getRelayBtcHeaderKeyBaseHeight()
		kv = append(kv, &types.KeyValue{key, heightbytes})
	}
	atomic.StoreInt64(&b.height, int64(head.Height))
	b.lastheader = head

	relaylog.Debug("relayBTCStore saveBlockHead", "height", b.height)
	return kv
}

func decodeHeight(heightbytes []byte) (int64, error) {
	var height types.Int64
	err := types.Decode(heightbytes, &height)
	if err != nil {
		//may be old database format json...
		err = json.Unmarshal(heightbytes, &height.Data)
		if err != nil {
			relaylog.Error("decodeHeight Could not unmarshal height bytes", "error", err.Error())
			return -1, types.ErrUnmarshal
		}
	}
	return height.Data, nil
}

func (b *relayBTCStore) getHeadHeigtList(req *types.ReqRelayBtcHeaderHeightList) (types.Message, error) {
	prefix := getRelayBtcHeightListKey()
	key := getRelayBtcHeaderKeyHeightList(req.ReqHeight)

	values, err := b.r.GetLocalDB().List(prefix, key, req.Counts, req.Direction)
	if err != nil {
		relaylog.Error("getHeadHeigtList Could not list height", "error", err.Error(), "key", key, "count", req.Counts)
		values, err = b.r.GetLocalDB().List(prefix, nil, req.Counts, req.Direction)
		if err != nil {
			return nil, err
		}
	}

	var replay types.ReplyRelayBtcHeadHeightList
	heightGot := make(map[int64]bool)
	for _, heightbyte := range values {
		height, _ := decodeHeight(heightbyte)
		if !heightGot[height] {
			replay.Heights = append(replay.Heights, height)
			heightGot[height] = true
		}
	}

	return &replay, nil

}

func getDbMissedHeights(db dbm.KVDB, prefix []byte, baseHeight int64, direct int32) ([]int64, error) {
	var heightarray []int64
	key := getRelayBtcHeaderKeyHeightList(baseHeight)

	count := 1000
	if baseHeight < 1000 {
		count = int(baseHeight)
	}

	values, err := db.List(prefix, key, int32(count), direct)
	if err != nil {
		relaylog.Error("enter getDbMissedHeights", "key", key, "count", count)
		return nil, err
	}

	heightlist := make(map[int64]bool)
	for _, heightbyte := range values {
		height, _ := decodeHeight(heightbyte)
		heightlist[height] = true
	}

	index := baseHeight - int64(count)
	for i := 0; i < count; i++ {
		if !heightlist[index] {
			heightarray = append(heightarray, index)
		}
		index++
	}

	return heightarray, nil
}

func (b *relayBTCStore) getHeadMissedHeigtList(req *types.ReqRelayBtcHeaderHeightList) (types.Message, error) {

	var replay types.ReplyRelayBtcHeadHeightList
	prefix := getRelayBtcHeightListKey()
	for baseheight := req.ReqHeight; baseheight > 0; baseheight = baseheight - 1000 {
		heightlist, _ := getDbMissedHeights(b.r.GetLocalDB(), prefix, baseheight, req.Direction)
		replay.Heights = append(replay.Heights, heightlist...)
	}

	return &replay, nil

}

func (b *relayBTCStore) getBTCHeadDbCurHeight(req *types.ReqRelayQryBTCHeadHeight) (types.Message, error) {
	key := getRelayBtcHeaderKeyLastHeight()
	height, err := getBTCHeadHeightFromDb(b.r, key)
	if err != nil {
		relaylog.Error("relay getBTCHeadDbCurHeight fail", "key", string(key), "err", err.Error())
		return nil, err
	}

	key = getRelayBtcHeaderKeyBaseHeight()
	baseheight, err := getBTCHeadHeightFromDb(b.r, key)
	if err != nil {
		relaylog.Error("relay getBTCHeadDbCurHeight fail", "key", string(key), "err", err.Error())
		baseheight = -1
	}
	var replay types.ReplayRelayQryBTCHeadHeight
	replay.CurHeight = height
	replay.BaseHeight = baseheight

	relaylog.Debug("relay getBTCHeadDbCurHeight succ", "cur height", height, "baseheight", baseheight)

	return &replay, nil
}

func (b *relayBTCStore) GetMerkleRootFromHeader(blockhash string) (string, error) {
	value, err := b.r.GetLocalDB().Get(getRelayBtcHeaderKeyHash(blockhash))
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
	str, err := b.GetMerkleRootFromHeader(verify.GetSpv().GetBlockHash())
	if err != nil {
		return false, err
	}
	realmerkleroot := merkelStrRevers2Bytes(str)
	relaylog.Debug("relay verifyTx", "db merkle root", realmerkleroot, "verify m-root", verifymerkleroot)
	return bytes.Equal(realmerkleroot, verifymerkleroot), nil

}

//rawtx, txindex, sibling, blockhash
//
//sibling like "aaaaaa-bbbbbb-cccccc..."

func (b *relayBTCStore) verifyBTCTx(verify *types.RelayVerifyBTC) (bool, error) {
	rawhash := getRawTxHash(verify.Rawtx)
	sibs := getSiblingHash(verify.Merkbranch)

	verifymerkleroot := merkle.GetMerkleRootFromBranch(sibs, rawhash, verify.Txindex)
	str, err := b.GetMerkleRootFromHeader(verify.Blockhash)
	if err != nil {
		return false, err
	}
	realmerkleroot := merkelStrRevers2Bytes(str)

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

func merkelStrRevers2Bytes(str string) []byte {
	data, _ := common.FromHex(str)
	merkle := common.BytesToHash(data).Revers().Bytes()
	return merkle

}

func merkelStrRevers(str string) string {
	data, _ := common.FromHex(str)
	merkle := common.BytesToHash(data).Revers().String()
	return merkle

}

func merkleBytes2Str(bts []byte) string {
	return common.BytesToHash(bts).String()
}

func merkleStr2Byte(str string) []byte {
	data, _ := common.FromHex(str)
	return data
}
