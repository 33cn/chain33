package types

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestDecodeLogNewTicket(t *testing.T) {
	var logTmp = &ReceiptTicket{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &rpctypes.ReceiptLog{
		Ty:  TyLogNewTicket,
		Log: "0x" + strdec,
	}

	logs := []*rpctypes.ReceiptLog{}
	logs = append(logs, rlog)

	var data = &rpctypes.ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := rpctypes.DecodeLog([]byte("ticket"), data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogNewTicket", result.Logs[0].TyName)
}

func TestDecodeLogCloseTicket(t *testing.T) {
	var logTmp = &ReceiptTicket{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &rpctypes.ReceiptLog{
		Ty:  TyLogCloseTicket,
		Log: "0x" + strdec,
	}

	logs := []*rpctypes.ReceiptLog{}
	logs = append(logs, rlog)

	var data = &rpctypes.ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := rpctypes.DecodeLog([]byte("ticket"), data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogCloseTicket", result.Logs[0].TyName)
}

func TestDecodeLogMinerTicket(t *testing.T) {
	var logTmp = &ReceiptTicket{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &rpctypes.ReceiptLog{
		Ty:  TyLogMinerTicket,
		Log: "0x" + strdec,
	}

	logs := []*rpctypes.ReceiptLog{}
	logs = append(logs, rlog)

	var data = &rpctypes.ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := rpctypes.DecodeLog([]byte("ticket"), data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogMinerTicket", result.Logs[0].TyName)
}

func TestDecodeLogTicketBind(t *testing.T) {
	var logTmp = &ReceiptTicketBind{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &rpctypes.ReceiptLog{
		Ty:  TyLogTicketBind,
		Log: "0x" + strdec,
	}

	logs := []*rpctypes.ReceiptLog{}
	logs = append(logs, rlog)

	var data = &rpctypes.ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := rpctypes.DecodeLog([]byte("ticket"), data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTicketBind", result.Logs[0].TyName)
}

func TestProtoNewEncodeOldDecode(t *testing.T) {
	tnew := &TicketMiner{
		Bits:     1,
		Reward:   1,
		TicketId: "id",
		Modify:   []byte("modify"),
		PrivHash: []byte("hash"),
	}
	data := types.Encode(tnew)
	told := &TicketMinerOld{}
	err := types.Decode(data, told)
	assert.Nil(t, err)
	assert.Equal(t, &TicketMinerOld{
		Bits:     1,
		Reward:   1,
		TicketId: "id",
		Modify:   []byte("modify"),
	}, told)
}
