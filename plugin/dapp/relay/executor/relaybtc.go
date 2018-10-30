package executor

import (
	"bytes"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/golang/protobuf/proto"

	"math/big"

	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/difficulty"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/types"
)

type btcStore struct {
	db dbm.KVDB
}

func newBtcStore(db dbm.KVDB) *btcStore {
	return &btcStore{db: db}
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

func (b *btcStore) getBtcHeadByHeight(height int64) (*ty.BtcHeader, error) {
	var head ty.BtcHeader
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

func (b *btcStore) getLastBtcHead() (*ty.BtcHeader, error) {
	height, err := b.getLastBtcHeadHeight()
	if err != nil {
		return nil, err
	}

	head, err := b.getBtcHeadByHeight(height)
	if err != nil {
		return nil, err
	}

	return head, nil
}

func (b *btcStore) saveBlockHead(head *ty.BtcHeader) ([]*types.KeyValue, error) {
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

	return kv, nil
}

func (b *btcStore) saveBlockLastHead(head *ty.ReceiptRelayRcvBTCHeaders) ([]*types.KeyValue, error) {
	var kv []*types.KeyValue

	heightBytes := types.Encode(&types.Int64{int64(head.NewHeight)})
	key := relayBTCHeaderLastHeight
	kv = append(kv, &types.KeyValue{key, heightBytes})

	heightBytes = types.Encode(&types.Int64{int64(head.NewBaseHeight)})
	key = relayBTCHeaderBaseHeight
	kv = append(kv, &types.KeyValue{key, heightBytes})

	return kv, nil
}

func (b *btcStore) delBlockHead(head *ty.BtcHeader) ([]*types.KeyValue, error) {
	var kv []*types.KeyValue

	key := calcBtcHeaderKeyHash(head.Hash)
	kv = append(kv, &types.KeyValue{key, nil})
	// height:header
	key = calcBtcHeaderKeyHeight(int64(head.Height))
	kv = append(kv, &types.KeyValue{key, nil})

	// prefix-height:height
	key = calcBtcHeaderKeyHeightList(int64(head.Height))
	kv = append(kv, &types.KeyValue{key, nil})

	return kv, nil
}

func (b *btcStore) delBlockLastHead(head *ty.ReceiptRelayRcvBTCHeaders) ([]*types.KeyValue, error) {
	var kv []*types.KeyValue
	var key []byte

	heightBytes := types.Encode(&types.Int64{int64(head.LastHeight)})
	key = relayBTCHeaderLastHeight
	kv = append(kv, &types.KeyValue{key, heightBytes})

	heightBytes = types.Encode(&types.Int64{int64(head.LastBaseHeight)})
	key = relayBTCHeaderBaseHeight
	kv = append(kv, &types.KeyValue{key, heightBytes})

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

func (b *btcStore) getBtcCurHeight(req *ty.ReqRelayQryBTCHeadHeight) (types.Message, error) {

	height, err := b.getLastBtcHeadHeight()
	if err == types.ErrNotFound {
		height = -1
	} else if err != nil {
		return nil, err
	}

	key := relayBTCHeaderBaseHeight
	baseHeight, err := b.getBtcHeadHeightFromDb(key)
	if err == types.ErrNotFound {
		baseHeight = -1
	} else if err != nil {
		return nil, err
	}
	var replay ty.ReplayRelayQryBTCHeadHeight
	replay.CurHeight = height
	replay.BaseHeight = baseHeight
	return &replay, nil
}

func (b *btcStore) getMerkleRootFromHeader(blockhash string) (string, error) {
	value, err := b.db.Get(calcBtcHeaderKeyHash(blockhash))
	if err != nil {
		return "", err
	}

	var header ty.BtcHeader
	if err = types.Decode(value, &header); err != nil {
		return "", err
	}

	return header.MerkleRoot, nil

}

func (b *btcStore) verifyBtcTx(verify *ty.RelayVerify, order *ty.RelayOrder) error {
	var foundtx bool
	for _, outtx := range verify.GetTx().GetVout() {
		if outtx.Address == order.CoinAddr && outtx.Value >= order.CoinAmount {
			foundtx = true
		}
	}

	if !foundtx {
		return ty.ErrRelayVerifyAddrNotFound
	}

	acceptTime := time.Unix(order.AcceptTime, 0)
	txTime := time.Unix(verify.GetTx().Time, 0)
	confirmTime := time.Unix(order.ConfirmTime, 0)

	if txTime.Sub(acceptTime) < 0 || confirmTime.Sub(txTime) < 0 {
		relaylog.Error("verifyTx", "tx time not correct to accept", txTime.Sub(acceptTime), "to confirm time", confirmTime.Sub(txTime))
		return ty.ErrRelayBtcTxTimeErr
	}

	height, err := b.getLastBtcHeadHeight()
	if err != nil {
		return err
	}

	if verify.Tx.BlockHeight+uint64(order.CoinWaits) > uint64(height) {
		return ty.ErrRelayWaitBlocksErr
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
		return ty.ErrRelayVerify
	}

	return nil

}

func (b *btcStore) verifyCmdBtcTx(verify *ty.RelayVerifyCli) error {
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
		return ty.ErrRelayVerify
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

func (b *btcStore) getHeadHeightList(req *ty.ReqRelayBtcHeaderHeightList) (types.Message, error) {
	prefix := []byte(relayBTCHeaderHeightList)
	key := calcBtcHeaderKeyHeightList(req.ReqHeight)

	values, err := b.db.List(prefix, key, req.Counts, req.Direction)
	if err != nil {
		values, err = b.db.List(prefix, nil, req.Counts, req.Direction)
		if err != nil {
			return nil, err
		}
	}

	var replay ty.ReplyRelayBtcHeadHeightList
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

func btcWireHeader(head *ty.BtcHeader) (*wire.BlockHeader, error) {
	preHash, err := chainhash.NewHashFromStr(head.PreviousHash)
	if err != nil {
		return nil, err
	}
	merkleRoot, err := chainhash.NewHashFromStr(head.MerkleRoot)
	if err != nil {
		return nil, err
	}

	h := &wire.BlockHeader{}
	h.Version = int32(head.Version)
	h.PrevBlock = *preHash
	h.MerkleRoot = *merkleRoot
	h.Bits = uint32(head.Bits)
	h.Nonce = uint32(head.Nonce)
	h.Timestamp = time.Unix(head.Time, 0)

	return h, nil
}

func verifyBlockHeader(head *ty.BtcHeader, preHead *ty.RelayLastRcvBtcHeader, localDb dbm.KVDB) error {
	if head == nil {
		return types.ErrInvalidParam
	}

	if preHead != nil && preHead.Header != nil && (preHead.Header.Hash != head.PreviousHash || preHead.Header.Height+1 != head.Height) && !head.IsReset {

		return ty.ErrRelayBtcHeadSequenceErr
	}

	//real BTC block not change the bits before height<30000, not match with the calculation result
	if !head.IsReset && head.Height > 30000 {
		newBits, err := calcNextRequiredDifficulty(preHead.Header, localDb)
		if err != nil && err != types.ErrNotFound {
			return err
		}

		if newBits != 0 && newBits != head.Bits {
			return ty.ErrRelayBtcHeadNewBitsErr
		}
	}

	btcHeader, err := btcWireHeader(head)
	if err != nil {
		return err
	}
	hash := btcHeader.BlockHash()

	if hash.String() != head.Hash {
		return ty.ErrRelayBtcHeadHashErr
	}

	target := difficulty.CompactToBig(uint32(head.Bits))

	// The block hash must be less than the claimed target.
	hashNum := difficulty.HashToBig(hash[:])
	if hashNum.Cmp(target) > 0 {
		return ty.ErrRelayBtcHeadBitsErr
	}

	return nil
}

// refer to btcd's blockchain's calcNextRequiredDifficulty() function
// calcNextRequiredDifficulty calculates the required difficulty for the block
// after the passed previous block node based on the difficulty retarget rules.
func calcNextRequiredDifficulty(preHead *ty.BtcHeader, localDb dbm.KVDB) (int64, error) {
	if preHead == nil {
		return 0, nil
	}

	// Genesis block.
	targetTimespan := time.Hour * 24 * 14  // 14 days
	TargetTimePerBlock := time.Minute * 10 // 10 minutes
	retargetAdjustmentFactor := int64(4)   // 25% less, 400% more
	timeSpan := int64(targetTimespan / time.Second)
	timeBlock := int64(TargetTimePerBlock / time.Second)

	// powLimit is the highest proof of work value a Bitcoin block
	// can have for the regression test network.  It is the value 2^255 - 1.
	bigOne := big.NewInt(1)
	powLimit := new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)
	blocksPerRetarget := uint64(timeSpan / timeBlock)
	minRetargetTimespan := timeSpan / retargetAdjustmentFactor
	maxRetargetTimespan := timeSpan * retargetAdjustmentFactor

	// Return the previous block's difficulty requirements if this block
	// is not at a difficulty retarget interval.
	if (preHead.Height+1)%blocksPerRetarget != 0 {
		// For networks that support it, allow special reduction of the
		// required difficulty once too much time has elapsed without
		// mining a block.

		// For the main network (or any unrecognized networks), simply
		// return the previous block's difficulty requirements.
		return preHead.Bits, nil
	}

	// Get the block node at the previous retarget (targetTimespan days
	// worth of blocks).
	btc := newBtcStore(localDb)
	firstHead, err := btc.getBtcHeadByHeight(int64(preHead.Height - (blocksPerRetarget - 1)))
	if err != nil {
		return 0, err
	}

	// Limit the amount of adjustment that can occur to the previous
	// difficulty.
	actualTimespan := preHead.Time - firstHead.Time
	adjustedTimespan := actualTimespan
	if actualTimespan < minRetargetTimespan {
		adjustedTimespan = minRetargetTimespan
	} else if actualTimespan > maxRetargetTimespan {
		adjustedTimespan = maxRetargetTimespan
	}

	// Calculate new target difficulty as:
	//  currentDifficulty * (adjustedTimespan / targetTimespan)
	// The result uses integer division which means it will be slightly
	// rounded down.  Bitcoind also uses integer division to calculate this
	// result.
	oldTarget := difficulty.CompactToBig(uint32(preHead.Bits))
	newTarget := new(big.Int).Mul(oldTarget, big.NewInt(adjustedTimespan))
	newTarget.Div(newTarget, big.NewInt(timeSpan))

	// Limit new value to the proof of work limit.
	if newTarget.Cmp(powLimit) > 0 {
		newTarget.Set(powLimit)
	}

	newTargetBits := difficulty.BigToCompact(newTarget)

	return int64(newTargetBits), nil
}
