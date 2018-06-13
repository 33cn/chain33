package relay

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/types"

	"bytes"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"gitlab.33.cn/chain33/chain33/common/difficulty"
)

func btcWireHeader(head *types.BtcHeader) (error, *wire.BlockHeader) {
	preHash, err := chainhash.NewHashFromStr(head.PreviousHash)
	if err != nil {
		return err, nil
	}
	merkleRoot, err := chainhash.NewHashFromStr(head.MerkleRoot)
	if err != nil {
		return err, nil
	}

	h := &wire.BlockHeader{}
	h.Version = int32(head.Version)
	h.PrevBlock = *preHash
	h.MerkleRoot = *merkleRoot
	h.Bits = uint32(head.Bits)
	h.Nonce = uint32(head.Nonce)
	h.Timestamp = time.Unix(head.Time, 0)

	return nil, h
}

type btcStore struct {
	r *relay
	// db         	dbm.KVDB
	height     int64
	lastHeader *types.BtcHeader
}

func (b *btcStore) new(r *relay) {
	b.r = r
	b.height = -1
	b.lastHeader = nil
}

func getBTCHeadHeightFromDb(r *relay, key []byte) (int64, error) {
	val, err := r.GetLocalDB().Get(key)
	if val == nil || err != nil {
		return -1, err
	}
	height, err := decodeHeight(val)
	if err != nil {
		return -1, err
	}

	return height, nil
}

func (b *btcStore) verifyBlockHeader(head *types.BtcHeader) bool {
	if b.lastHeader.Hash != head.PreviousHash || b.lastHeader.Height+1 != head.Height {
		relaylog.Warn("verifyBlockHeader fail", "last height", b.lastHeader.Height, "rcv height", head.Height)
		return false
	}

	err, btcHeader := btcWireHeader(head)
	if err != nil {
		return false
	}
	hash := btcHeader.BlockHash()

	if hash.String() != head.Hash {
		relaylog.Warn("verifyBlockHeader fail", "calc hash", hash.String(), "!= rcv hash", head.Hash)
		return false
	}

	target := difficulty.CompactToBig(uint32(head.Bits))

	// The block hash must be less than the claimed target.
	hashNum := difficulty.HashToBig(hash[:])
	if hashNum.Cmp(target) > 0 {
		relaylog.Warn("verifyBlockHeader fail", "block hash", head.Hash, "is higher than", hashNum.String(), "height", head.Height)
		return false
	}

	return true
}

func (b *btcStore) checkBlockHead(head *types.BtcHeader) bool {
	if head.IsReset {
		return true
	}

	height := atomic.LoadInt64(&b.height)

	// generis height or restart (if node restart, the height init to -1, btcWireHeader sync tx with block will be save directly)
	if height <= 0 || b.lastHeader == nil {
		relaylog.Info("btcStore checkBlockHead", "height", height)
		return true
	}

	return b.verifyBlockHeader(head)
}

func (b *btcStore) saveBlockHead(head *types.BtcHeader) []*types.KeyValue {
	var kv []*types.KeyValue
	var key []byte

	if !b.checkBlockHead(head) {
		return kv
	}

	val, err := proto.Marshal(head)
	if err != nil {
		relaylog.Error("btcStore Marshal header", "height", head.Height, "hash", head.Hash, "error", err)

	} else {
		// hash:header
		key = calcBtcHeaderKeyHash(head.Hash)
		kv = append(kv, &types.KeyValue{key, val})
		// height:header
		key = calcBtcHeaderKeyHeight(int64(head.Height))
		kv = append(kv, &types.KeyValue{key, val})
	}

	// prefix-height:height
	key = calcBtcHeaderKeyHeightList(int64(head.Height))
	heightBytes := types.Encode(&types.Int64{int64(head.Height)})
	kv = append(kv, &types.KeyValue{key, heightBytes})
	// last height
	key = calcBtcHeaderKeyLastHeight()
	kv = append(kv, &types.KeyValue{key, heightBytes})

	// for start with height =-1 case, the base not be set, just return -1
	if head.IsReset {
		key = calcBtcHeaderKeyBaseHeight()
		kv = append(kv, &types.KeyValue{key, heightBytes})
	}
	atomic.StoreInt64(&b.height, int64(head.Height))
	if head != nil {
		b.lastHeader = head
	}

	return kv
}

func decodeHeight(heightBytes []byte) (int64, error) {
	var height types.Int64
	err := types.Decode(heightBytes, &height)
	if err != nil {
		// may be old database format json...
		err = json.Unmarshal(heightBytes, &height.Data)
		if err != nil {
			relaylog.Error("decodeHeight Could not unmarshal height bytes", "error", err.Error())
			return -1, types.ErrUnmarshal
		}
	}
	return height.Data, nil
}

func (b *btcStore) getBtcHeadDbCurHeight(req *types.ReqRelayQryBTCHeadHeight) (types.Message, error) {
	key := calcBtcHeaderKeyLastHeight()
	height, err := getBTCHeadHeightFromDb(b.r, key)
	if err != nil {
		relaylog.Info("relay getBtcHeadDbCurHeight none", "key", string(key), "err", err.Error())
		height = -1
	}

	key = calcBtcHeaderKeyBaseHeight()
	baseHeight, err := getBTCHeadHeightFromDb(b.r, key)
	if err != nil {
		relaylog.Info("relay getBtcHeadDbCurHeight none", "key", string(key), "err", err.Error())
		baseHeight = -1
	}
	var replay types.ReplayRelayQryBTCHeadHeight
	replay.CurHeight = height
	replay.BaseHeight = baseHeight

	relaylog.Debug("relay get db height succ", "cur height", height, "baseHeight", baseHeight)

	return &replay, nil
}

func (b *btcStore) getMerkleRootFromHeader(blockhash string) (string, error) {
	value, err := b.r.GetLocalDB().Get(calcBtcHeaderKeyHash(blockhash))
	if err != nil {
		relaylog.Error("getMerkleRootFromHeader", "Failed to get value from db with blockhash", blockhash)
		return "", err
	}

	var header types.BtcHeader
	if err = types.Decode(value, &header); err != nil {
		relaylog.Error("getMerkleRootFromHeader", "Failed to decode head", blockhash)
		return "", err
	}

	return header.MerkleRoot, nil

}

func (b *btcStore) verifyTx(verify *types.RelayVerify, order *types.RelayOrder) error {
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

	acceptTime := time.Unix(order.AcceptTime, 0)
	txTime := time.Unix(verify.GetTx().Time, 0)
	confirmTime := time.Unix(order.ConfirmTime, 0)

	if txTime.Sub(acceptTime) < 0 || confirmTime.Sub(txTime) < 0 {
		relaylog.Error("verifyTx", "tx time not correct to accept", txTime.Sub(acceptTime), "to confirm time", confirmTime.Sub(txTime))
		return types.ErrRelayBtcTxTimeErr
	}

	curDbBlockHeight := atomic.LoadInt64(&b.height)
	if verify.Tx.BlockHeight+waitBlockHeight > uint64(curDbBlockHeight) {
		relaylog.Info("verifyTx", "Failed to wait 6 blocks, tx height", verify.Tx.BlockHeight, "curHeight ", curDbBlockHeight)
		return types.ErrRelayWaitBlocksErr
	}

	rawHash, err := btcHashStrRevers(verify.GetTx().GetHash())
	if err != nil {
		relaylog.Error("verifyTx", "fail convers tx hash", verify.GetTx().GetHash())
		return types.ErrRelayVerify
	}
	sibs := verify.GetSpv().GetBranchProof()

	verifyMerkleRoot := merkle.GetMerkleRootFromBranch(sibs, rawHash, verify.GetSpv().GetTxIndex())
	str, err := b.getMerkleRootFromHeader(verify.GetSpv().GetBlockHash())
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

// rawtx, txindex, sibling, blockhash
//
// sibling like "aaaaaa-bbbbbb-cccccc..."

func (b *btcStore) verifyBtcTx(verify *types.RelayVerifyCli) error {
	rawhash := getRawTxHash(verify.RawTx)
	sibs := getSiblingHash(verify.MerkBranch)

	verifymerkleroot := merkle.GetMerkleRootFromBranch(sibs, rawhash, verify.TxIndex)
	str, err := b.getMerkleRootFromHeader(verify.BlockHash)
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
		return nil, err
	}
	merkle := common.BytesToHash(data).Revers().Bytes()
	return merkle, nil
}

func (b *btcStore) getHeadHeightList(req *types.ReqRelayBtcHeaderHeightList) (types.Message, error) {
	prefix := calcBtcHeightListKey()
	key := calcBtcHeaderKeyHeightList(req.ReqHeight)

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
