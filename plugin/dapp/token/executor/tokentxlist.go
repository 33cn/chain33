package executor

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/common/address"
	tp "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	tokenTxPrefix        = "LODB-token-txHash:"
	tokenTxAddrPrefix    = "LODB-token-txAddrHash:"
	tokenTxAddrDirPrefix = "LODB-token-txAddrDirHash:"
)

func TokenTxKvs(tx *types.Transaction, symbol string, height, index int64, isDel bool) ([]*types.KeyValue, error) {
	var kv []*types.KeyValue

	from := address.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	to := tx.GetRealToAddr()
	keys := tokenTxkeys(symbol, from, to, height, index)

	var txInfo []byte
	if !isDel {
		txInfo = makeReplyTxInfo(tx, height, index, symbol)
	}
	for _, k := range keys {
		kv = append(kv, &types.KeyValue{k, txInfo})
	}
	return kv, nil
}

func tokenTxkeys(symbol, from, to string, height, index int64) (result [][]byte) {
	key := CalcTokenTxKey(symbol, height, index)
	result = append(result, key)
	if len(from) > 0 {
		fromKey1 := CalcTokenAddrTxKey(symbol, from, height, index)
		fromKey2 := CalcTokenAddrTxDirKey(symbol, from, dapp.TxIndexFrom, height, index)
		result = append(result, fromKey1)
		result = append(result, fromKey2)
	}
	if len(to) > 0 {
		toKey1 := CalcTokenAddrTxKey(symbol, to, height, index)
		toKey2 := CalcTokenAddrTxDirKey(symbol, to, dapp.TxIndexTo, height, index)
		result = append(result, toKey1)
		result = append(result, toKey2)
	}
	return
}

// token transaction entities in local DB
func CalcTokenTxKey(symbol string, height, index int64) []byte {
	if height == -1 {
		return []byte(fmt.Sprintf(tokenTxPrefix+"%s:%s", symbol, ""))
	}
	return []byte(fmt.Sprintf(tokenTxPrefix+"%s:%s", symbol, dapp.HeightIndexStr(height, index)))
}

func CalcTokenAddrTxKey(symbol, addr string, height, index int64) []byte {
	if height == -1 {
		return []byte(fmt.Sprintf(tokenTxAddrPrefix+"%s:%s:%s", symbol, addr, ""))
	}
	return []byte(fmt.Sprintf(tokenTxAddrPrefix+"%s:%s:%s", symbol, addr, dapp.HeightIndexStr(height, index)))
}

func CalcTokenAddrTxDirKey(symbol, addr string, flag int32, height, index int64) []byte {
	if height == -1 {
		return []byte(fmt.Sprintf(tokenTxAddrDirPrefix+"%s:%s:%d:%s", symbol, addr, flag, ""))
	}
	return []byte(fmt.Sprintf(tokenTxAddrDirPrefix+"%s:%s:%d:%s", symbol, addr, flag,
		dapp.HeightIndexStr(height, index)))
}

func makeReplyTxInfo(tx *types.Transaction, height, index int64, symbol string) []byte {
	var info types.ReplyTxInfo
	info.Hash = tx.Hash()
	info.Height = height
	info.Index = index
	info.Assets = []*types.Asset{{Exec: tp.TokenX, Symbol: symbol}}

	return types.Encode(&info)
}
