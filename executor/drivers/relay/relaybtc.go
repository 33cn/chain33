package relay

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/types"

	"bytes"
	"encoding/json"
	"sync"
	"sync/atomic"
)

var (
	lastHeaderLock sync.Mutex
)

type relayBTCStore struct {
	r *relay
	//db         	dbm.KVDB
	height     int64
	lastHeader *types.BtcHeader
}

func (b *relayBTCStore) new(r *relay) {
	b.r = r
	b.height = -1
	b.lastHeader = nil
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

	if head.IsReset {
		relaylog.Info("relayBTCStore checkBlockHead head flag=reset", "reset height", head.Height)
		return true
	}

	height := atomic.LoadInt64(&b.height)

	lastHeaderLock.Lock()
	defer lastHeaderLock.Unlock()
	// generis height or restart (if node restart, the height init to -1, new sync tx with block will be save directly)
	if height <= 0 || b.lastHeader == nil {
		relaylog.Info("relayBTCStore checkBlockHead", "height", height)
		return true
	}

	if head.PreviousHash == b.lastHeader.Hash && head.Height == b.lastHeader.Height+1 {
		return true
	}

	relaylog.Error("relayBTCStore checkBlockHead fail", "last height", b.lastHeader.Height, "check head height", head.Height,
		"last hash", b.lastHeader.Hash, "check pre hash", head.PreviousHash)
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
		key = getBtcHeaderKeyHash(head.Hash)
		kv = append(kv, &types.KeyValue{key, val})
		//height:header
		key = getBtcHeaderKeyHeight(int64(head.Height))
		kv = append(kv, &types.KeyValue{key, val})
	}

	//prefix-height:height
	key = getBtcHeaderKeyHeightList(int64(head.Height))
	heightBytes := types.Encode(&types.Int64{int64(head.Height)})
	kv = append(kv, &types.KeyValue{key, heightBytes})
	//last height
	key = getBtcHeaderKeyLastHeight()
	kv = append(kv, &types.KeyValue{key, heightBytes})

	//for start with height =-1 case, the base not be set, just return -1
	if head.IsReset {
		key = getBtcHeaderKeyBaseHeight()
		kv = append(kv, &types.KeyValue{key, heightBytes})
	}
	atomic.StoreInt64(&b.height, int64(head.Height))
	lastHeaderLock.Lock()
	defer lastHeaderLock.Unlock()
	if head != nil {
		b.lastHeader = head
	}

	return kv
}

func decodeHeight(heightBytes []byte) (int64, error) {
	var height types.Int64
	err := types.Decode(heightBytes, &height)
	if err != nil {
		//may be old database format json...
		err = json.Unmarshal(heightBytes, &height.Data)
		if err != nil {
			relaylog.Error("decodeHeight Could not unmarshal height bytes", "error", err.Error())
			return -1, types.ErrUnmarshal
		}
	}
	return height.Data, nil
}

func (b *relayBTCStore) getBTCHeadDbCurHeight(req *types.ReqRelayQryBTCHeadHeight) (types.Message, error) {
	key := getBtcHeaderKeyLastHeight()
	height, err := getBTCHeadHeightFromDb(b.r, key)
	if err != nil {
		relaylog.Info("relay getBTCHeadDbCurHeight none", "key", string(key), "err", err.Error())
		height = -1
	}

	key = getBtcHeaderKeyBaseHeight()
	baseHeight, err := getBTCHeadHeightFromDb(b.r, key)
	if err != nil {
		relaylog.Info("relay getBTCHeadDbCurHeight none", "key", string(key), "err", err.Error())
		baseHeight = -1
	}
	var replay types.ReplayRelayQryBTCHeadHeight
	replay.CurHeight = height
	replay.BaseHeight = baseHeight

	relaylog.Debug("relay getBTCHeadDbCurHeight succ", "cur height", height, "baseHeight", baseHeight)

	return &replay, nil
}

func (b *relayBTCStore) GetMerkleRootFromHeader(blockhash string) (string, error) {
	value, err := b.r.GetLocalDB().Get(getBtcHeaderKeyHash(blockhash))
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

func (b *relayBTCStore) verifyTx(verify *types.RelayVerify, order *types.RelayOrder) error {
	var foundtx bool
	for _, outtx := range verify.GetTx().GetVout() {
		if outtx.Address == order.CoinAddr && outtx.Value == order.CoinAmount {
			foundtx = true
		}
	}

	if !foundtx {
		relaylog.Info("verifyTx", "Failed to get order", order.Id, "tx", order.CoinAddr)
		return types.ErrRelayVerifyAddrNotFound
	}

	curDbBlockHeight := atomic.LoadInt64(&b.height)
	if verify.Tx.BlockHeight+WAIT_BLOCK_HEIGHT > uint64(curDbBlockHeight) {
		relaylog.Info("verifyTx", "Failed to wait 6 blocks, curHeight", curDbBlockHeight, "tx height", verify.Tx.BlockHeight)
		return types.ErrRelayWaitBlocksErr
	}

	rawHash, err := btcHashStrRevers(verify.GetTx().GetHash())
	if err != nil {
		relaylog.Error("verifyTx", "fail convers tx hash", verify.GetTx().GetHash())
		return types.ErrRelayVerify
	}
	sibs := verify.GetSpv().GetBranchProof()

	verifyMerkleRoot := merkle.GetMerkleRootFromBranch(sibs, rawHash, verify.GetSpv().GetTxIndex())
	str, err := b.GetMerkleRootFromHeader(verify.GetSpv().GetBlockHash())
	if err != nil {
		return err
	}
	realMerkleRoot, err := btcHashStrRevers(str)
	if err != nil {
		relaylog.Error("verifyTx", "fail convers merkle hash", str)
		return types.ErrRelayVerify
	}

	rst := bytes.Equal(realMerkleRoot, verifyMerkleRoot)
	if !rst {
		relaylog.Error("relay verifyTx", "db merkle root", realMerkleRoot, "tx merkle root", verifyMerkleRoot)
		return types.ErrRelayVerify
	}

	return nil

}

//rawtx, txindex, sibling, blockhash
//
//sibling like "aaaaaa-bbbbbb-cccccc..."

func (b *relayBTCStore) verifyBTCTx(verify *types.RelayVerifyCli) error {
	rawhash := getRawTxHash(verify.RawTx)
	sibs := getSiblingHash(verify.MerkBranch)

	verifymerkleroot := merkle.GetMerkleRootFromBranch(sibs, rawhash, verify.TxIndex)
	str, err := b.GetMerkleRootFromHeader(verify.BlockHash)
	if err != nil {
		return err
	}
	realmerkleroot, err := btcHashStrRevers(str)
	if err != nil {
		return err
	}

	rst := bytes.Equal(realmerkleroot, verifymerkleroot)
	if !rst {
		relaylog.Error("relay verifyTx", "db merkle root", realmerkleroot, "tx merkle root", verifymerkleroot)
		return types.ErrRelayVerify
	}

	return nil
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
		sibs[i], _ = btcHashStrRevers(val)
	}
	return sibs[:][:]
}

func btcHashStrRevers(str string) ([]byte, error) {
	data, err := common.FromHex(str)
	if err != nil {
		relaylog.Error("merkleStrRevers2Bytes", "str", str, "error", err.Error())
		return nil, err
	}
	merkle := common.BytesToHash(data).Revers().Bytes()
	return merkle, nil
}

func (b *relayBTCStore) getHeadHeightList(req *types.ReqRelayBtcHeaderHeightList) (types.Message, error) {
	prefix := getBtcHeightListKey()
	key := getBtcHeaderKeyHeightList(req.ReqHeight)

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
	for _, heightByte := range values {
		height, _ := decodeHeight(heightByte)
		if !heightGot[height] {
			replay.Heights = append(replay.Heights, height)
			heightGot[height] = true
		}
	}

	return &replay, nil

}
