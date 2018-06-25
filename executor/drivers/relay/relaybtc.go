package relay

import (
	"bytes"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/golang/protobuf/proto"

	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/difficulty"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/types"
)

type btcStore struct {
	db         dbm.KVDB
	lastHeader *types.BtcHeader
}

func newBtcStore(r *relay) (*btcStore, error) {
	store := &btcStore{
		db:         r.GetLocalDB(),
		lastHeader: nil,
	}

	height, err := store.getLastBtcHeadHeight()
	if err == types.ErrNotFound {
		return store, nil
	}
	if err != nil {
		return nil, err
	}

	head, err := store.getBtcHeadByHeight(height)
	if err != nil {
		return nil, err
	}

	store.lastHeader = head
	return store, nil

}

func (b *btcStore) getBtcHeadHeightFromDb(key []byte) (int64, error) {
	val, err := b.db.Get(key)
	if err != nil {
		return -1, err
	}
	height, err := decodeHeight(val)
	if err != nil {
		return -1, err
	}

	return height, nil
}

func (b *btcStore) getLastBtcHeadHeight() (int64, error) {
	key := relayBTCHeaderLastHeight
	return b.getBtcHeadHeightFromDb(key)
}

func (b *btcStore) getBtcHeadByHeight(height int64) (*types.BtcHeader, error) {
	var head types.BtcHeader
	key := calcBtcHeaderKeyHeight(height)
	val, err := b.db.Get(key)
	if err != nil {
		return nil, err
	}
	err = types.Decode(val, &head)
	if err != nil {
		return nil, err
	}

	return &head, nil
}

func (b *btcStore) saveBlockHead(head *types.BtcHeader) ([]*types.KeyValue, error) {
	var kv []*types.KeyValue
	var key []byte

	val, err := proto.Marshal(head)
	if err != nil {
		relaylog.Error("saveBlockHead", "height", head.Height, "hash", head.Hash)
		return nil, err

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
	key = relayBTCHeaderLastHeight
	kv = append(kv, &types.KeyValue{key, heightBytes})

	// for start with height =-1 case, the base not be set, just return -1
	if head.IsReset {
		key = relayBTCHeaderBaseHeight
		kv = append(kv, &types.KeyValue{key, heightBytes})
	}
	if head != nil {
		b.lastHeader = head
	}

	return kv, nil
}

func decodeHeight(heightBytes []byte) (int64, error) {
	var height types.Int64
	err := types.Decode(heightBytes, &height)
	if err != nil {
		return -1, err
	}
	return height.Data, nil
}

func (b *btcStore) getBtcHeadDbCurHeight(req *types.ReqRelayQryBTCHeadHeight) (types.Message, error) {
	var height int64

	if b.lastHeader != nil {
		height = int64(b.lastHeader.Height)
	} else {
		height = -1
	}

	key := relayBTCHeaderBaseHeight
	baseHeight, err := b.getBtcHeadHeightFromDb(key)
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

func (b *btcStore) getMerkleRootFromHeader(blockhash string) (string, error) {
	value, err := b.db.Get(calcBtcHeaderKeyHash(blockhash))
	if err != nil {
		return "", err
	}

	var header types.BtcHeader
	if err = types.Decode(value, &header); err != nil {
		return "", err
	}

	return header.MerkleRoot, nil

}

func (b *btcStore) verifyBtcTx(verify *types.RelayVerify, order *types.RelayOrder) error {
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
		relaylog.Error("verifyTx", "tx time not correct to accept", txTime.Sub(acceptTime), "to confirm time", confirmTime.Sub(txTime))
		return types.ErrRelayBtcTxTimeErr
	}

	height, err := b.getLastBtcHeadHeight()
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
	str, err := b.getMerkleRootFromHeader(verify.GetSpv().GetBlockHash())
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

func (b *btcStore) verifyCmdBtcTx(verify *types.RelayVerifyCli) error {
	rawhash, err := getRawTxHash(verify.RawTx)
	if err != nil {
		return err
	}
	sibs, err := getSiblingHash(verify.MerkBranch)
	if err != nil {
		return err
	}

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
		return types.ErrRelayVerify
	}

	return nil
}

func getRawTxHash(rawtx string) ([]byte, error) {
	data, err := common.FromHex(rawtx)
	if err != nil {
		return nil, err
	}
	h := common.DoubleHashH(data)
	return h.Bytes(), nil
}

func getSiblingHash(sibling string) ([][]byte, error) {
	var err error
	sibsarr := strings.Split(sibling, "-")

	sibs := make([][]byte, len(sibsarr))
	for i, val := range sibsarr {
		sibs[i], err = btcHashStrRevers(val)
		if err != nil {
			return nil, err
		}

	}
	return sibs[:][:], nil
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
	prefix := []byte(relayBTCHeaderHeightList)
	key := calcBtcHeaderKeyHeightList(req.ReqHeight)

	values, err := b.db.List(prefix, key, req.Counts, req.Direction)
	if err != nil {
		values, err = b.db.List(prefix, nil, req.Counts, req.Direction)
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
		if (preHead.Hash != head.PreviousHash || preHead.Height+1 != head.Height) && !head.IsReset {
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
