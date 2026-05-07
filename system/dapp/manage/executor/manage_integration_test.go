//go:build integration

package executor_test

import (
	"testing"

	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestManageModifyTokenBlacklist(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	// Create blacklist entry
	create := &types.ModifyConfig{
		Key:   "token-blacklist",
		Op:    "add",
		Value: "TEST",
		Addr:  "",
	}
	jsondata := types.MustPBToJSON(create)
	req := &rpctypes.CreateTxIn{
		Execer:     "manage",
		ActionName: "Modify",
		Payload:    jsondata,
	}
	var txhex string
	err = mocker.GetJSONC().Call("Chain33.CreateTransaction", req, &txhex)
	assert.Nil(t, err)
	hash, err := mocker.SendAndSign(mocker.GetHotKey(), txhex)
	assert.Nil(t, err)
	txinfo, err := mocker.WaitTx(hash)
	assert.Nil(t, err)
	assert.Equal(t, txinfo.Receipt.Ty, int32(types.ExecOk))

	// Query the blacklist
	queryreq := &types.ReqString{Data: "token-blacklist"}
	query := &rpctypes.Query4Jrpc{
		Execer:   "manage",
		FuncName: "GetConfigItem",
		Payload:  types.MustPBToJSON(queryreq),
	}
	var reply types.ReplyConfig
	err = mocker.GetJSONC().Call("Chain33.Query", query, &reply)
	assert.Nil(t, err)
	assert.Equal(t, reply.Key, "token-blacklist")
	assert.Contains(t, reply.Value, "TEST")

	// Delete the entry
	create = &types.ModifyConfig{
		Key:   "token-blacklist",
		Op:    "delete",
		Value: "TEST",
		Addr:  "",
	}
	jsondata = types.MustPBToJSON(create)
	req = &rpctypes.CreateTxIn{
		Execer:     "manage",
		ActionName: "Modify",
		Payload:    jsondata,
	}
	err = mocker.GetJSONC().Call("Chain33.CreateTransaction", req, &txhex)
	assert.Nil(t, err)
	hash, err = mocker.SendAndSign(mocker.GetHotKey(), txhex)
	assert.Nil(t, err)
	_, err = mocker.WaitTx(hash)
	assert.Nil(t, err)
}

func TestManageTokenFinisherAddRemove(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	addr := mocker.GetHotAddress()

	// Add token finisher
	create := &types.ModifyConfig{
		Key:   "token-finisher",
		Op:    "add",
		Value: addr,
		Addr:  "",
	}
	jsondata := types.MustPBToJSON(create)
	req := &rpctypes.CreateTxIn{
		Execer:     "manage",
		ActionName: "Modify",
		Payload:    jsondata,
	}
	var txhex string
	err = mocker.GetJSONC().Call("Chain33.CreateTransaction", req, &txhex)
	assert.Nil(t, err)
	hash, err := mocker.SendAndSign(mocker.GetHotKey(), txhex)
	assert.Nil(t, err)
	txinfo, err := mocker.WaitTx(hash)
	assert.Nil(t, err)
	assert.Equal(t, txinfo.Receipt.Ty, int32(types.ExecOk))

	// Delete token finisher
	create.Op = "delete"
	jsondata = types.MustPBToJSON(create)
	req = &rpctypes.CreateTxIn{
		Execer:     "manage",
		ActionName: "Modify",
		Payload:    jsondata,
	}
	err = mocker.GetJSONC().Call("Chain33.CreateTransaction", req, &txhex)
	assert.Nil(t, err)
	hash, err = mocker.SendAndSign(mocker.GetHotKey(), txhex)
	assert.Nil(t, err)
	_, err = mocker.WaitTx(hash)
	assert.Nil(t, err)
}
