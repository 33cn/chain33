package rpc

import (
	"math/rand"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/address"
	rTy "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var random = rand.New(rand.NewSource(types.Now().UnixNano()))

type channelClient struct {
	client.QueueProtocolAPI
}

func (c *channelClient) Init(q queue.Client) {
	c.QueueProtocolAPI, _ = client.New(q, nil)
}

func (c *channelClient) CreateRawRelayOrderTx(parm *RelayOrderTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &rTy.RelayCreate{
		Operation: parm.Operation,
		Coin:      parm.Coin,
		Amount:    parm.Amount,
		Addr:      parm.Addr,
		CoinWaits: parm.CoinWait,
		BtyAmount: parm.BtyAmount,
	}

	sell := &rTy.RelayAction{
		Ty:    rTy.RelayActionCreate,
		Value: &rTy.RelayAction_Create{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(sell),
		Fee:     parm.Fee,
		Nonce:   random.Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	tx.SetRealFee(types.MinFee)

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayAcceptTx(parm *RelayAcceptTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &rTy.RelayAccept{OrderId: parm.OrderId, CoinAddr: parm.CoinAddr, CoinWaits: parm.CoinWait}
	val := &rTy.RelayAction{
		Ty:    rTy.RelayActionAccept,
		Value: &rTy.RelayAction_Accept{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   random.Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	tx.SetRealFee(types.MinFee)

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayRevokeTx(parm *RelayRevokeTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &rTy.RelayRevoke{OrderId: parm.OrderId, Target: parm.Target, Action: parm.Action}
	val := &rTy.RelayAction{
		Ty:    rTy.RelayActionRevoke,
		Value: &rTy.RelayAction_Revoke{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   random.Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	tx.SetRealFee(types.MinFee)

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayConfirmTx(parm *RelayConfirmTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &rTy.RelayConfirmTx{OrderId: parm.OrderId, TxHash: parm.TxHash}
	val := &rTy.RelayAction{
		Ty:    rTy.RelayActionConfirmTx,
		Value: &rTy.RelayAction_ConfirmTx{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   random.Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	tx.SetRealFee(types.MinFee)

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayVerifyBTCTx(parm *RelayVerifyBTCTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &rTy.RelayVerifyCli{
		OrderId:    parm.OrderId,
		RawTx:      parm.RawTx,
		TxIndex:    parm.TxIndex,
		MerkBranch: parm.MerklBranch,
		BlockHash:  parm.BlockHash}
	val := &rTy.RelayAction{
		Ty:    rTy.RelayActionVerifyCmdTx,
		Value: &rTy.RelayAction_VerifyCli{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   random.Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	tx.SetRealFee(types.MinFee)

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelaySaveBTCHeadTx(parm *RelaySaveBTCHeadTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	head := &rTy.BtcHeader{
		Hash:         parm.Hash,
		PreviousHash: parm.PreviousHash,
		MerkleRoot:   parm.MerkleRoot,
		Height:       parm.Height,
		IsReset:      parm.IsReset,
	}

	v := &rTy.BtcHeaders{}
	v.BtcHeader = append(v.BtcHeader, head)

	val := &rTy.RelayAction{
		Ty:    rTy.RelayActionRcvBTCHeaders,
		Value: &rTy.RelayAction_BtcHeaders{v},
	}
	tx := &types.Transaction{
		Execer:  types.ExecerRelay,
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   random.Int63(),
		To:      address.ExecAddress(string(types.ExecerRelay)),
	}

	tx.SetRealFee(types.MinFee)

	data := types.Encode(tx)
	return data, nil
}
