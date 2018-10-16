package rpc

import (
	"encoding/hex"

	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func CreateRawRelayOrderTx(parm *ty.RelayOrderTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &ty.RelayCreate{
		Operation: parm.Operation,
		Coin:      parm.Coin,
		Amount:    parm.Amount,
		Addr:      parm.Addr,
		CoinWaits: parm.CoinWait,
		BtyAmount: parm.BtyAmount,
	}
	return types.CallCreateTx(types.ExecName(ty.RelayX), "Create", v)
}

func CreateRawRelayAcceptTx(parm *ty.RelayAcceptTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &ty.RelayAccept{OrderId: parm.OrderId, CoinAddr: parm.CoinAddr, CoinWaits: parm.CoinWait}
	return types.CallCreateTx(types.ExecName(ty.RelayX), "Accept", v)
}

func CreateRawRelayRevokeTx(parm *ty.RelayRevokeTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &ty.RelayRevoke{OrderId: parm.OrderId, Target: parm.Target, Action: parm.Action}
	return types.CallCreateTx(types.ExecName(ty.RelayX), "Revoke", v)
}

func CreateRawRelayConfirmTx(parm *ty.RelayConfirmTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &ty.RelayConfirmTx{OrderId: parm.OrderId, TxHash: parm.TxHash}
	return types.CallCreateTx(types.ExecName(ty.RelayX), "ConfirmTx", v)
}

func CreateRawRelayVerifyBTCTx(parm *ty.RelayVerifyBTCTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &ty.RelayVerifyCli{
		OrderId:    parm.OrderId,
		RawTx:      parm.RawTx,
		TxIndex:    parm.TxIndex,
		MerkBranch: parm.MerklBranch,
		BlockHash:  parm.BlockHash,
	}
	return types.CallCreateTx(types.ExecName(ty.RelayX), "VerifyCli", v)
}

func CreateRawRelaySaveBTCHeadTx(parm *ty.RelaySaveBTCHeadTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	head := &ty.BtcHeader{
		Hash:         parm.Hash,
		PreviousHash: parm.PreviousHash,
		MerkleRoot:   parm.MerkleRoot,
		Height:       parm.Height,
		IsReset:      parm.IsReset,
	}

	v := &ty.BtcHeaders{}
	v.BtcHeader = append(v.BtcHeader, head)
	return types.CallCreateTx(types.ExecName(ty.RelayX), "SaveBTCHeadTx", v)
}

func (c *Jrpc) CreateRawRelayOrderTx(in *ty.RelayOrderTx, result *interface{}) error {
	reply, err := CreateRawRelayOrderTx(in)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Jrpc) CreateRawRelayAcceptTx(in *ty.RelayAcceptTx, result *interface{}) error {
	reply, err := CreateRawRelayAcceptTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Jrpc) CreateRawRelayRevokeTx(in *ty.RelayRevokeTx, result *interface{}) error {
	reply, err := CreateRawRelayRevokeTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Jrpc) CreateRawRelayConfirmTx(in *ty.RelayConfirmTx, result *interface{}) error {
	reply, err := CreateRawRelayConfirmTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Jrpc) CreateRawRelayVerifyBTCTx(in *ty.RelayVerifyBTCTx, result *interface{}) error {
	reply, err := CreateRawRelayVerifyBTCTx(in)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Jrpc) CreateRawRelaySaveBTCHeadTx(in *ty.RelaySaveBTCHeadTx, result *interface{}) error {
	reply, err := CreateRawRelaySaveBTCHeadTx(in)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply)
	return nil
}
