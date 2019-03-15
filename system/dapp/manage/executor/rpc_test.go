package executor

import (
	"testing"

	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestManageConfig(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, sub, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)
	//创建黑名单
	// -o add -v BTY
	create := &types.ModifyConfig{
		Key:   "token-blacklist",
		Op:    "add",
		Value: "BTY",
		Addr:  "",
	}
	jsondata := types.MustPBToJSON(create)
	/*
	  {
	  		"execer": "manage",
	  		"actionName": "Modify",
	  		"payload": {
	  			"key": "token-blacklist",
	  			"value": "BTY",
	  			"op": "add",
	  			"addr": ""
	  		}
	  	}
	*/
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
	assert.Equal(t, txinfo.Receipt.Ty, int32(2))

	create = &types.ModifyConfig{
		Key:   "token-blacklist",
		Op:    "add",
		Value: "YCC",
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
	txinfo, err = mocker.WaitTx(hash)
	assert.Nil(t, err)
	assert.Equal(t, txinfo.Receipt.Ty, int32(2))

	create = &types.ModifyConfig{
		Key:   "token-blacklist",
		Op:    "add",
		Value: "TTT",
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
	txinfo, err = mocker.WaitTx(hash)
	assert.Nil(t, err)
	assert.Equal(t, txinfo.Receipt.Ty, int32(2))
	//做一个查询
	/*
		{
			"execer": "manage",
			"funcName": "GetConfigItem",
			"payload": {
				"data": "token-blacklist"
			}
		}
	*/
	queryreq := &types.ReqString{
		Data: "token-blacklist",
	}
	query := &rpctypes.Query4Jrpc{
		Execer:   "manage",
		FuncName: "GetConfigItem",
		Payload:  types.MustPBToJSON(queryreq),
	}
	var reply types.ReplyConfig
	err = mocker.GetJSONC().Call("Chain33.Query", query, &reply)
	assert.Nil(t, err)
	assert.Equal(t, reply.Key, "token-blacklist")
	assert.Equal(t, reply.Value, "[BTY YCC TTT]")

	create = &types.ModifyConfig{
		Key:   "token-blacklist",
		Op:    "delete",
		Value: "TTT",
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
	txinfo, err = mocker.WaitTx(hash)
	assert.Nil(t, err)
	util.JSONPrint(t, txinfo)
	assert.Equal(t, txinfo.Receipt.Ty, int32(2))

	queryreq = &types.ReqString{
		Data: "token-blacklist",
	}
	query = &rpctypes.Query4Jrpc{
		Execer:   "manage",
		FuncName: "GetConfigItem",
		Payload:  types.MustPBToJSON(queryreq),
	}
	err = mocker.GetJSONC().Call("Chain33.Query", query, &reply)
	assert.Nil(t, err)
	assert.Equal(t, reply.Key, "token-blacklist")
	assert.Equal(t, reply.Value, "[BTY YCC]")
}

func TestTokenFinisher(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, sub, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)
	//添加发币审核地址
	create := &types.ModifyConfig{
		Key:   "token-finisher",
		Op:    "add",
		Value: "1FCX9XJTZXvZteagTrefJEBPZMt8BFmdoi",
		Addr:  "",
	}
	jsondata := types.MustPBToJSON(create)
	/*
	  {
	  		"execer": "manage",
	  		"actionName": "Modify",
	  		"payload": {
	  			"key": "token-finisher",
	  			"value": "1FCX9XJTZXvZteagTrefJEBPZMt8BFmdoi",
	  			"op": "add",
	  			"addr": ""
	  		}
	  	}
	*/
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
	assert.Equal(t, txinfo.Receipt.Ty, int32(2))

	queryreq := &types.ReqString{
		Data: "token-finisher",
	}
	query := &rpctypes.Query4Jrpc{
		Execer:   "manage",
		FuncName: "GetConfigItem",
		Payload:  types.MustPBToJSON(queryreq),
	}
	var reply types.ReplyConfig
	err = mocker.GetJSONC().Call("Chain33.Query", query, &reply)
	assert.Nil(t, err)
	assert.Equal(t, reply.Key, "token-finisher")
	assert.Equal(t, reply.Value, "[1FCX9XJTZXvZteagTrefJEBPZMt8BFmdoi]")
}

func TestModify(t *testing.T) {
	manager := new(Manage)

	log := &types.ReceiptLog{Ty: 0, Log: types.Encode(&types.ReceiptConfig{Prev: &types.ConfigItem{}, Current: &types.ConfigItem{}})}
	receipt := &types.ReceiptData{Logs: []*types.ReceiptLog{log}}

	_, err := manager.ExecDelLocal_Modify(nil, nil, receipt, 0)
	assert.NoError(t, err)

	_, err = manager.ExecLocal_Modify(nil, nil, receipt, 0)
	assert.NoError(t, err)
}
