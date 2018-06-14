package relay

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/types"

	"bytes"
	"encoding/json"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"gitlab.33.cn/chain33/chain33/common/difficulty"
)

type btcStore struct {
	lastHeader *types.BtcHeader
}

func newBtcStore(r *relay) (*btcStore, error) {
	height, err := getLastBtcHeadHeight(r)
	if err == types.ErrNotFound {
		return &btcStore{lastHeader: nil}, nil
	}
	if err != nil {
		return nil, err
	}

	head, err := getBtcHeadByHeight(r, height)
	if err != nil {
		return nil, err
	}
	store := &btcStore{
		lastHeader: head,
	}

	return store, nil

}

func getBtcHeadHeightFromDb(r *relay, key []byte) (int64, error) {
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

func getLastBtcHeadHeight(r *relay) (int64, error) {
	key := calcBtcHeaderKeyLastHeight()
	return getBtcHeadHeightFromDb(r, key)
}

func getBtcHeadByHeight(r *relay, height int64) (*types.BtcHeader, error) {
	var head types.BtcHeader
	key := calcBtcHeaderKeyHeight(height)
	val, err := r.GetLocalDB().Get(key)
	if err != nil {
		return nil, err
	}
	err = types.Decode(val, &head)
	if err != nil {
		return nil, err
	}

	return &head, nil
}

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

func verifyBlockHeader(head *types.BtcHeader, preHead *types.BtcHeader) error {
	if head == nil {
		return types.ErrInputPara
	}
	if preHead != nil {
		if preHead.Hash != head.PreviousHash || preHead.Height+1 != head.Height {
			return types.ErrRelayBtcHeadSequenceErr
		}
	}

	err, btcHeader := btcWireHeader(head)
	if err != nil {
		return err
	}
	hash := btcHeader.BlockHash()

	if hash.String() != head.Hash {
		return types.ErrRelayBtcHeadHashErr
	}

	target := difficulty.CompactToBig(uint32(head.Bits))

	// The block hash must be less than the claimed target.
	hashNum := difficulty.HashToBig(hash[:])
	if hashNum.Cmp(target) > 0 {
		return types.ErrRelayBtcHeadBitsErr
	}

	return nil
}

func (b *btcStore) checkBlockHead(head *types.BtcHeader) bool {
	if head.IsReset {
		return true
	}

	if b.lastHeader == nil {
		return true
	}

	//local only accept sequence tx if head not set reset flag
	if b.lastHeader.Hash != head.PreviousHash || b.lastHeader.Height+1 != head.Height {
		relaylog.Warn("checkBlockHead", "last height", b.lastHeader.Height, "rcv height", head.Height)
		return false
	}

	return true
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

func getBtcHeadDbCurHeight(r *relay, req *types.ReqRelayQryBTCHeadHeight) (types.Message, error) {
	height, err := getLastBtcHeadHeight(r)
	if err == types.ErrNotFound {
		height = -1
	} else if err != nil {
		return nil, err
	}

	key := calcBtcHeaderKeyBaseHeight()
	baseHeight, err := getBtcHeadHeightFromDb(r, key)
	if err == types.ErrNotFound {
		baseHeight = -1
	} else if err != nil {
		return nil, err
	}
	var replay types.ReplayRelayQryBTCHeadHeight
	replay.CurHeight = height
	replay.BaseHeight = baseHeight

	return &replay, nil
}

func getMerkleRootFromHeader(r *relay, blockhash string) (string, error) {
	value, err := r.GetLocalDB().Get(calcBtcHeaderKeyHash(blockhash))
	if err != nil {
		return "", err
	}

	var header types.BtcHeader
	if err = types.Decode(value, &header); err != nil {
		return "", err
	}

	return header.MerkleRoot, nil

}

func verifyBtcTx(r *relay, verify *types.RelayVerify, order *types.RelayOrder) error {
	var foundtx bool
	for _, outtx := range verify.GetTx().GetVout() {
		if outtx.Address == order.CoinAddr && outtx.Value == order.CoinAmount {
			foundtx = true
		}
	}

	if !foundtx {
		return types.ErrRelayVerifyAddrNotFound
	}

	acceptTime := time.Unix(order.AcceptTime, 0)
	txTime := time.Unix(verify.GetTx().Time, 0)
	confirmTime := time.Unix(order.ConfirmTime, 0)

	if txTime.Sub(acceptTime) < 0 || confirmTime.Sub(txTime) < 0 {
		relaylog.Info("verifyTx", "tx time not correct to accept", txTime.Sub(acceptTime), "to confirm time", confirmTime.Sub(txTime))
		return types.ErrRelayBtcTxTimeErr
	}

	height, err := getLastBtcHeadHeight(r)
	if err != nil {
		return err
	}
	if verify.Tx.BlockHeight+waitBlockHeight > uint64(height) {
		return types.ErrRelayWaitBlocksErr
	}

	rawHash, err := btcHashStrRevers(verify.GetTx().GetHash())
	if err != nil {
		return err
	}
	sibs := verify.GetSpv().GetBranchProof()

	verifyRoot := merkle.GetMerkleRootFromBranch(sibs, rawHash, verify.GetSpv().GetTxIndex())
	str, err := getMerkleRootFromHeader(r, verify.GetSpv().GetBlockHash())
	if err != nil {
		return err
	}
	realMerkleRoot, err := btcHashStrRevers(str)
	if err != nil {
		return err
	}

	rst := bytes.Equal(realMerkleRoot, verifyRoot)
	if !rst {
		return types.ErrRelayVerify
	}

	return nil

}

// rawtx, txindex, sibling, blockhash
//
// sibling like "aaaaaa-bbbbbb-cccccc..."

func verifyCmdBtcTx(r *relay, verify *types.RelayVerifyCli) error {
	rawhash := getRawTxHash(verify.RawTx)
	sibs := getSiblingHash(verify.MerkBranch)

	verifymerkleroot := merkle.GetMerkleRootFromBranch(sibs, rawhash, verify.TxIndex)
	str, err := getMerkleRootFromHeader(r, verify.BlockHash)
	if err != nil {
		return err
	}
	realmerkleroot, err := btcHashStrRevers(str)
	if err != nil {
		return err
	}

	rst := bytes.Equal(realmerkleroot, verifymerkleroot)
	if !rst {
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

func getHeadHeightList(r *relay, req *types.ReqRelayBtcHeaderHeightList) (types.Message, error) {
	prefix := calcBtcHeightListKey()
	key := calcBtcHeaderKeyHeightList(req.ReqHeight)

	values, err := r.GetLocalDB().List(prefix, key, req.Counts, req.Direction)
	if err != nil {
		values, err = r.GetLocalDB().List(prefix, nil, req.Counts, req.Direction)
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
