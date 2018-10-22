package rpc

import (
	"encoding/hex"

	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func CreateRawRelayOrderTx(parm *ty.RelayCreate) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := *parm
	return types.CallCreateTx(types.ExecName(ty.RelayX), "Create", &v)
}

func CreateRawRelayAcceptTx(parm *ty.RelayAccept) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	return types.CallCreateTx(types.ExecName(ty.RelayX), "Accept", parm)
}

func CreateRawRelayRevokeTx(parm *ty.RelayRevoke) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	return types.CallCreateTx(types.ExecName(ty.RelayX), "Revoke", parm)
}

func CreateRawRelayConfirmTx(parm *ty.RelayConfirmTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	return types.CallCreateTx(types.ExecName(ty.RelayX), "ConfirmTx", parm)
}

func CreateRawRelayVerifyBTCTx(parm *ty.RelayVerifyCli) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := *parm
	return types.CallCreateTx(types.ExecName(ty.RelayX), "VerifyCli", &v)
}

func CreateRawRelaySaveBTCHeadTx(parm *ty.BtcHeader) ([]byte, error) {
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
	return types.CallCreateTx(types.ExecName(ty.RelayX), "BtcHeaders", v)
}

func (c *Jrpc) CreateRawRelayOrderTx(in *ty.RelayCreate, result *interface{}) error {
	reply, err := CreateRawRelayOrderTx(in)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Jrpc) CreateRawRelayAcceptTx(in *ty.RelayAccept, result *interface{}) error {
	reply, err := CreateRawRelayAcceptTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Jrpc) CreateRawRelayRevokeTx(in *ty.RelayRevoke, result *interface{}) error {
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

func (c *Jrpc) CreateRawRelayVerifyBTCTx(in *ty.RelayVerifyCli, result *interface{}) error {
	reply, err := CreateRawRelayVerifyBTCTx(in)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Jrpc) CreateRawRelaySaveBTCHeadTx(in *ty.BtcHeader, result *interface{}) error {
	reply, err := CreateRawRelaySaveBTCHeadTx(in)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply)
	return nil
}
