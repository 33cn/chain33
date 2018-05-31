package rpc

import (
	"errors"
	"testing"

	"encoding/hex"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestDecodeUserWrite(t *testing.T) {
	payload := []byte("#md#hello#world")
	data := decodeUserWrite(payload)
	assert.Equal(t, data, &userWrite{Topic: "md", Content: "hello#world"})

	payload = []byte("hello#world")
	data = decodeUserWrite(payload)
	assert.Equal(t, data, &userWrite{Topic: "", Content: "hello#world"})

	payload = []byte("123#hello#suyanlong")
	data = decodeUserWrite(payload)
	assert.NotEqual(t, data, &userWrite{Topic: "123", Content: "hello#world"})
}

func TestDecodeTx(t *testing.T) {
	tx := types.Transaction{
		Execer:  []byte("coin"),
		Payload: []byte("342412abcd"),
		Nonce:   8978167239,
		To:      "1asd234dsf43fds",
	}

	data, err := DecodeTx(&tx)
	assert.NotNil(t, data)
	assert.Nil(t, err)

	tx.Execer = []byte("coins")
	data, err = DecodeTx(&tx)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	tx = types.Transaction{
		Execer:  []byte("hashlock"),
		Payload: []byte("34"),
		Nonce:   8978167239,
		To:      "1asd234dsf43fds",
	}

	t.Log(string(tx.Execer))
	data, err = DecodeTx(&tx)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestDecodeLogErr(t *testing.T) {
	enc := "0001020304050607"
	dec := []byte{0, 1, 2, 3, 4, 5, 6, 7}

	hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogErr,
		Log: "0x" + enc,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogErr", result.Logs[0].TyName)
	assert.Equal(t, int32(types.TyLogErr), result.Logs[0].Ty)
}

func TestDecodeLogFee(t *testing.T) {
	var account = &types.Account{}
	var logTmp = &types.ReceiptAccountTransfer{
		Prev:    account,
		Current: account,
	}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogFee,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogFee", result.Logs[0].TyName)
}

func TestDecodeLogTransfer(t *testing.T) {
	var account = &types.Account{}
	var logTmp = &types.ReceiptAccountTransfer{
		Prev:    account,
		Current: account,
	}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTransfer,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTransfer", result.Logs[0].TyName)
}

func TestDecodeLogGenesis(t *testing.T) {
	enc := "0001020304050607"

	rlog := &ReceiptLog{
		Ty:  types.TyLogGenesis,
		Log: "0x" + enc,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogGenesis", result.Logs[0].TyName)
}

func TestDecodeLogDeposit(t *testing.T) {
	var account = &types.Account{}
	var logTmp = &types.ReceiptAccountTransfer{
		Prev:    account,
		Current: account,
	}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogDeposit,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogDeposit", result.Logs[0].TyName)
}

func TestDecodeLogExecTransfer(t *testing.T) {
	var account = &types.Account{}
	var logTmp = &types.ReceiptExecAccountTransfer{
		Prev:    account,
		Current: account,
	}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogExecTransfer,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogExecTransfer", result.Logs[0].TyName)
}

func TestDecodeLogExecWithdraw(t *testing.T) {
	var account = &types.Account{}
	var logTmp = &types.ReceiptExecAccountTransfer{
		Prev:    account,
		Current: account,
	}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogExecWithdraw,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogExecWithdraw", result.Logs[0].TyName)
}

func TestDecodeLogExecDeposit(t *testing.T) {
	var account = &types.Account{}
	var logTmp = &types.ReceiptExecAccountTransfer{
		Prev:    account,
		Current: account,
	}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogExecDeposit,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogExecDeposit", result.Logs[0].TyName)
}

func TestDecodeLogExecFrozen(t *testing.T) {
	var account = &types.Account{}
	var logTmp = &types.ReceiptExecAccountTransfer{
		Prev:    account,
		Current: account,
	}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogExecFrozen,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogExecFrozen", result.Logs[0].TyName)
}

func TestDecodeLogExecActive(t *testing.T) {
	var account = &types.Account{}
	var logTmp = &types.ReceiptExecAccountTransfer{
		Prev:    account,
		Current: account,
	}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogExecActive,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogExecActive", result.Logs[0].TyName)
}

func TestDecodeLogGenesisTransfer(t *testing.T) {
	var account = &types.Account{}
	var logTmp = &types.ReceiptAccountTransfer{
		Prev:    account,
		Current: account,
	}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogGenesisTransfer,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogGenesisTransfer", result.Logs[0].TyName)
}

func TestDecodeLogGenesisDeposit(t *testing.T) {
	var account = &types.Account{}
	var logTmp = &types.ReceiptAccountTransfer{
		Prev:    account,
		Current: account,
	}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogGenesisDeposit,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogGenesisDeposit", result.Logs[0].TyName)
}

func TestDecodeLogNewTicket(t *testing.T) {
	var logTmp = &types.ReceiptTicket{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogNewTicket,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogNewTicket", result.Logs[0].TyName)
}

func TestDecodeLogCloseTicket(t *testing.T) {
	var logTmp = &types.ReceiptTicket{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogCloseTicket,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogCloseTicket", result.Logs[0].TyName)
}

func TestDecodeLogMinerTicket(t *testing.T) {
	var logTmp = &types.ReceiptTicket{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogMinerTicket,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogMinerTicket", result.Logs[0].TyName)
}

func TestDecodeLogTicketBind(t *testing.T) {
	var logTmp = &types.ReceiptTicketBind{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTicketBind,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTicketBind", result.Logs[0].TyName)
}

func TestDecodeLogPreCreateToken(t *testing.T) {
	var logTmp = &types.ReceiptToken{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogPreCreateToken,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogPreCreateToken", result.Logs[0].TyName)
}

func TestDecodeLogFinishCreateToken(t *testing.T) {
	var logTmp = &types.ReceiptToken{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogFinishCreateToken,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogFinishCreateToken", result.Logs[0].TyName)
}

func TestDecodeLogRevokeCreateToken(t *testing.T) {
	var logTmp = &types.ReceiptToken{}

	dec := types.Encode(logTmp)

	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogRevokeCreateToken,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogRevokeCreateToken", result.Logs[0].TyName)
}

func TestDecodeLogTradeSellLimit(t *testing.T) {
	var logTmp = &types.ReceiptTradeSell{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTradeSellLimit,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeSell", result.Logs[0].TyName)
}

func TestDecodeLogTradeBuyMarket(t *testing.T) {
	var logTmp = &types.ReceiptTradeBuyMarket{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTradeBuyMarket,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeBuy", result.Logs[0].TyName)
}

func TestDecodeLogTradeSellRevoke(t *testing.T) {
	var logTmp = &types.ReceiptTradeBuyMarket{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTradeSellRevoke,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeRevoke", result.Logs[0].TyName)
}

func TestDecodeLogTradeBuyLimit(t *testing.T) {
	var logTmp = &types.ReceiptTradeBuyLimit{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTradeBuyLimit,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeBuyLimit", result.Logs[0].TyName)
}

func TestDecodeLogTradeSellMarket(t *testing.T) {
	var logTmp = &types.ReceiptSellMarket{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTradeSellMarket,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeSellMarket", result.Logs[0].TyName)
}

func TestDecodeLogTradeBuyRevoke(t *testing.T) {
	var logTmp = &types.ReceiptTradeBuyRevoke{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTradeBuyRevoke,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeBuyRevoke", result.Logs[0].TyName)
}

func TestDecodeLogTokenTransfer(t *testing.T) {
	var logTmp = &types.ReceiptAccountTransfer{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTokenTransfer,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTokenTransfer", result.Logs[0].TyName)
}

func TestDecodeLogTokenDeposit(t *testing.T) {
	var logTmp = &types.ReceiptAccountTransfer{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTokenDeposit,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTokenDeposit", result.Logs[0].TyName)
}

func TestDecodeLogTokenExecTransfer(t *testing.T) {
	var logTmp = &types.ReceiptExecAccountTransfer{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTokenExecTransfer,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTokenExecTransfer", result.Logs[0].TyName)
}

func TestDecodeLogTokenExecWithdraw(t *testing.T) {
	var logTmp = &types.ReceiptExecAccountTransfer{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTokenExecWithdraw,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTokenExecWithdraw", result.Logs[0].TyName)
}

func TestDecodeLogTokenExecDeposit(t *testing.T) {
	var logTmp = &types.ReceiptExecAccountTransfer{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTokenExecDeposit,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTokenExecDeposit", result.Logs[0].TyName)
}

func TestDecodeLogTokenExecFrozen(t *testing.T) {
	var logTmp = &types.ReceiptExecAccountTransfer{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTokenExecFrozen,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTokenExecFrozen", result.Logs[0].TyName)
}

func TestDecodeLogTokenExecActive(t *testing.T) {
	var logTmp = &types.ReceiptExecAccountTransfer{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTokenExecActive,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   0,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTokenExecActive", result.Logs[0].TyName)
}

func TestDecodeLogTokenGenesisTransfer(t *testing.T) {
	var logTmp = &types.ReceiptAccountTransfer{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTokenGenesisTransfer,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   1,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTokenGenesisTransfer", result.Logs[0].TyName)
}

func TestDecodeLogTokenGenesisDeposit(t *testing.T) {
	var logTmp = &types.ReceiptExecAccountTransfer{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogTokenGenesisDeposit,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   2,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTokenGenesisDeposit", result.Logs[0].TyName)
}

func TestDecodeLogModifyConfig(t *testing.T) {
	var logTmp = &types.ReceiptConfig{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &ReceiptLog{
		Ty:  types.TyLogModifyConfig,
		Log: "0x" + strdec,
	}

	logs := []*ReceiptLog{}
	logs = append(logs, rlog)

	var data = &ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogModifyConfig", result.Logs[0].TyName)
}

func newTestChain33(api *mocks.QueueProtocolAPI) *Chain33 {
	return &Chain33{
		cli: channelClient{
			QueueProtocolAPI: api,
		},
	}
}

func TestChain33_CreateRawTransaction(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	// var result interface{}
	// api.On("CreateRawTransaction", nil, &result).Return()
	testChain33 := newTestChain33(api)
	var testResult interface{}
	err := testChain33.CreateRawTransaction(nil, &testResult)
	assert.Nil(t, testResult)
	assert.NotNil(t, err)

	tx := &types.CreateTx{
		To:          "qew",
		Amount:      10,
		Fee:         1,
		Note:        "12312",
		IsWithdraw:  false,
		IsToken:     true,
		TokenSymbol: "CNY",
		ExecName:    "token",
	}

	err = testChain33.CreateRawTransaction(tx, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_SendRawTransaction(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	// var result interface{}
	api.On("SendTx", mock.Anything).Return()

	testChain33 := newTestChain33(api)
	var testResult interface{}
	signedTx := SignedTx{
		Unsign: "123",
		Sign:   "123",
		Pubkey: "123",
		Ty:     1,
	}
	err := testChain33.SendRawTransaction(signedTx, &testResult)
	t.Log(err)
	assert.Nil(t, testResult)
	assert.NotNil(t, err)
	// api.Called(1)
	// mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SendRawTransactionSignError(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	// var result interface{}
	api.On("SendTx", mock.Anything).Return()

	testChain33 := newTestChain33(api)
	var testResult interface{}
	src := []byte("123")
	pubkey := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(pubkey, src)
	signedTx := SignedTx{
		Unsign: "123",
		Sign:   "123",
		Pubkey: string(pubkey),
		Ty:     1,
	}
	err := testChain33.SendRawTransaction(signedTx, &testResult)
	t.Log(err)
	assert.Nil(t, testResult)
	assert.NotNil(t, err)
	// api.Called(1)
	// mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SendRawTransactionUnsignError(t *testing.T) {
	reply := &types.Reply{IsOk: true}
	api := new(mocks.QueueProtocolAPI)
	// var result interface{}
	api.On("SendTx", mock.Anything).Return(reply, nil)

	testChain33 := newTestChain33(api)
	var testResult interface{}
	src := []byte("123")
	pubkey := make([]byte, hex.EncodedLen(len(src)))
	signkey := make([]byte, hex.EncodedLen(len(src)))

	hex.Encode(pubkey, src)
	hex.Encode(signkey, src)

	signedTx := SignedTx{
		Unsign: "123",
		Sign:   string(signkey),
		Pubkey: string(pubkey),
		Ty:     1,
	}
	err := testChain33.SendRawTransaction(signedTx, &testResult)
	t.Log(err)
	assert.Nil(t, testResult)
	assert.NotNil(t, err)

	tx := &types.Transaction{
		To: "to",
	}
	txByte := types.Encode(tx)
	unsign := make([]byte, hex.EncodedLen(len(txByte)))
	hex.Encode(unsign, txByte)

	signedTx = SignedTx{
		Unsign: string(unsign),
		Sign:   string(signkey),
		Pubkey: string(pubkey),
		Ty:     1,
	}
	err = testChain33.SendRawTransaction(signedTx, &testResult)
	t.Log(testResult)
	assert.Nil(t, err)
	assert.Equal(t, "0x", testResult)
	//assert.NotNil(t, err)

	// api.Called(1)
	// mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SendTransaction(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	api.On("SendTx", &types.Transaction{}).Return(nil, errors.New("error value"))
	testChain33 := newTestChain33(api)
	var testResult interface{}
	data := RawParm{
		Data: "",
	}
	err := testChain33.SendTransaction(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetHexTxByHash(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	api.On("QueryTx", &types.ReqHash{}).Return(nil, errors.New("error value"))
	testChain33 := newTestChain33(api)
	var testResult interface{}
	data := QueryParm{
		Hash: "",
	}
	err := testChain33.GetHexTxByHash(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_QueryTransaction(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	api.On("QueryTx", &types.ReqHash{}).Return(nil, errors.New("error value"))
	testChain33 := newTestChain33(api)
	var testResult interface{}
	data := QueryParm{
		Hash: "",
	}
	err := testChain33.QueryTransaction(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetBlocks(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	api.On("GetBlocks", &types.ReqBlocks{Pid: []string{""}}).Return(nil, errors.New("error value"))
	testChain33 := newTestChain33(api)
	var testResult interface{}
	data := BlockParam{}
	err := testChain33.GetBlocks(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetLastHeader(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	api.On("GetBlocks", &types.ReqBlocks{Pid: []string{""}}).Return(nil, errors.New("error value"))
	testChain33 := newTestChain33(api)
	var testResult interface{}
	data := BlockParam{}
	err := testChain33.GetBlocks(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetTxByAddr(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	api.On("GetTransactionByAddr", &types.ReqAddr{}).Return(nil, errors.New("error value"))
	var testResult interface{}
	data := types.ReqAddr{}
	err := testChain33.GetTxByAddr(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetTxByHashes(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	var parm types.ReqHashes
	parm.Hashes = make([][]byte, 0)
	api.On("GetTransactionByHash", &parm).Return(nil, errors.New("error value"))
	var testResult interface{}
	data := ReqHashes{}
	err := testChain33.GetTxByHashes(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetMempool(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	api.On("GetMempool").Return(nil, errors.New("error value"))
	var testResult interface{}
	data := &types.ReqNil{}
	err := testChain33.GetMempool(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetAccounts(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	api.On("WalletGetAccountList").Return(nil, errors.New("error value"))
	var testResult interface{}
	data := &types.ReqNil{}
	err := testChain33.GetAccounts(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_NewAccount(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	api.On("NewAccount", &types.ReqNewAccount{}).Return(nil, errors.New("error value"))

	var testResult interface{}
	err := testChain33.NewAccount(types.ReqNewAccount{}, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_WalletTxList(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletTransactionList{FromTx: []byte("")}
	api.On("WalletTransactionList", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := ReqWalletTransactionList{}
	err := testChain33.WalletTxList(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_ImportPrivkey(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletImportPrivKey{}
	api.On("WalletImportprivkey", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletImportPrivKey{}
	err := testChain33.ImportPrivkey(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SendToAddress(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletSendToAddress{}
	api.On("WalletSendToAddress", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletSendToAddress{}
	err := testChain33.SendToAddress(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SetTxFee(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletSetFee{}
	api.On("WalletSetFee", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletSetFee{}
	err := testChain33.SetTxFee(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SetLabl(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletSetLabel{}
	api.On("WalletSetLabel", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletSetLabel{}
	err := testChain33.SetLabl(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_MergeBalance(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletMergeBalance{}
	api.On("WalletMergeBalance", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletMergeBalance{}
	err := testChain33.MergeBalance(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SetPasswd(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletSetPasswd{}
	api.On("WalletSetPasswd", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletSetPasswd{}
	err := testChain33.SetPasswd(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_Lock(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	// expected := types.ReqNil{}
	api.On("WalletLock").Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqNil{}
	err := testChain33.Lock(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_UnLock(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.WalletUnLock{}
	api.On("WalletUnLock", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.WalletUnLock{}
	err := testChain33.UnLock(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetPeerInfo(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	api.On("PeerInfo").Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqNil{}
	err := testChain33.GetPeerInfo(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetHeaders(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqBlocks{}
	api.On("GetHeaders", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqBlocks{}
	err := testChain33.GetHeaders(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetLastMemPool(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	// expected := &types.ReqBlocks{}
	api.On("GetLastMempool").Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqNil{}
	err := testChain33.GetLastMemPool(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetBlockOverview(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqHash{}
	api.On("GetBlockOverview", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := QueryParm{}
	err := testChain33.GetBlockOverview(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetAddrOverview(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqAddr{}
	api.On("GetAddrOverview", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqAddr{}
	err := testChain33.GetAddrOverview(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	// mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetBlockHash(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqInt{}
	api.On("GetBlockHash", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqInt{}
	err := testChain33.GetBlockHash(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GenSeed(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.GenSeedLang{}
	api.On("GenSeed", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.GenSeedLang{}
	err := testChain33.GenSeed(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SaveSeed(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.SaveSeedByPw{}
	api.On("SaveSeed", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.SaveSeedByPw{}
	err := testChain33.SaveSeed(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetSeed(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.GetSeedByPw{}
	api.On("GetSeed", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.GetSeedByPw{}
	err := testChain33.GetSeed(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetWalletStatus(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	// expected := &types.GetSeedByPw{}
	api.On("GetWalletStatus").Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqNil{}
	err := testChain33.GetWalletStatus(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

// func TestChain33_GetBalance(t *testing.T) {
// 	api := new(mocks.QueueProtocolAPI)
// 	testChain33 := newTestChain33(api)
//
// 	expected := &types.ReqBalance{}
// 	api.On("GetBalance",expected).Return(nil, errors.New("error value"))
//
// 	var testResult interface{}
// 	actual := types.ReqBalance{}
// 	err := testChain33.GetBalance(actual, &testResult)
// 	t.Log(err)
// 	assert.Equal(t, nil, testResult)
// 	assert.NotNil(t, err)
//
// 	mock.AssertExpectationsForObjects(t, api)
// }

// ----------------------------

func TestChain33_Version(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)
	var testResult interface{}
	in := &types.ReqNil{}
	err := testChain33.Version(in, &testResult)
	t.Log(err)
	assert.Equal(t, nil, err)
	assert.NotNil(t, testResult)
}

func TestChain33_CreateRawTokenPreCreateTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTokenPreCreateTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TokenPreCreateTx{
		OwnerAddr: "asdf134",
		Symbol:    "CNY",
		Fee:       123,
	}
	err = client.CreateRawTokenPreCreateTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTokenRevokeTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTokenRevokeTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TokenRevokeTx{
		OwnerAddr: "asdf134",
		Symbol:    "CNY",
		Fee:       123,
	}
	err = client.CreateRawTokenRevokeTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTokenFinishTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTokenFinishTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TokenFinishTx{
		OwnerAddr: "asdf134",
		Symbol:    "CNY",
		Fee:       123,
	}
	err = client.CreateRawTokenFinishTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTradeSellTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeSellTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeSellTx{
		TokenSymbol:       "CNY",
		AmountPerBoardlot: 10,
		MinBoardlot:       1,
		PricePerBoardlot:  100,
		TotalBoardlot:     100,
		Fee:               1,
	}

	err = client.CreateRawTradeSellTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTradeBuyTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeBuyTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeBuyTx{
		SellID:      "sadfghjkhgfdsa",
		BoardlotCnt: 100,
		Fee:         1,
	}

	err = client.CreateRawTradeBuyTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTradeRevokeTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeRevokeTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeRevokeTx{
		SellID: "sadfghjkhgfdsa",
		Fee:    1,
	}

	err = client.CreateRawTradeRevokeTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)

}

func TestChain33_CreateRawTradeBuyLimitTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeBuyLimitTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeBuyLimitTx{
		TokenSymbol:       "CNY",
		AmountPerBoardlot: 10,
		MinBoardlot:       1,
		PricePerBoardlot:  100,
		TotalBoardlot:     100,
		Fee:               1,
	}

	err = client.CreateRawTradeBuyLimitTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)

}

func TestChain33_CreateRawTradeSellMarketTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeSellMarketTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeSellMarketTx{
		BuyID:       "12asdfa",
		BoardlotCnt: 100,
		Fee:         1,
	}

	err = client.CreateRawTradeSellMarketTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)

}

func TestChain33_CreateRawTradeRevokeBuyTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeRevokeBuyTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeRevokeBuyTx{
		BuyID: "12asdfa",
		Fee:   1,
	}

	err = client.CreateRawTradeRevokeBuyTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}
