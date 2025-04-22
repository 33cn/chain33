// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/system/crypto/common/authority/utils"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/system/crypto/sm2"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
	"github.com/pkg/errors"

	// TODO: 暂时将插件中的类型引用起来，后续需要修改

	"encoding/hex"

	"fmt"
	"time"
)

// DecodeTransaction decode transaction function
func DecodeTransaction(tx *rpctypes.Transaction) *TxResult {
	result := &TxResult{
		Execer:     tx.Execer,
		Payload:    tx.Payload,
		RawPayload: tx.RawPayload,
		Signature:  tx.Signature,
		Fee:        tx.FeeFmt,
		Amount:     tx.AmountFmt,
		Expire:     tx.Expire,
		Nonce:      tx.Nonce,
		To:         tx.To,
		From:       tx.From,
		GroupCount: tx.GroupCount,
		Header:     tx.Header,
		Next:       tx.Next,
		Hash:       tx.Hash,
		ChainID:    tx.ChainID,
	}
	return result
}

// DecodeAccount decode account func
func DecodeAccount(acc *types.Account, precision int64) *AccountResult {
	balanceResult := types.FormatAmount2FloatDisplay(acc.GetBalance(), precision, true)
	frozenResult := types.FormatAmount2FloatDisplay(acc.GetFrozen(), precision, true)
	accResult := &AccountResult{
		Addr:     acc.GetAddr(),
		Currency: acc.GetCurrency(),
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	return accResult
}

// SendToAddress send to address func
func SendToAddress(rpcAddr string, from string, to string, amount int64, note string, isToken bool, tokenSymbol string, isWithdraw bool) {
	amt := amount
	if isWithdraw {
		amt = -amount
	}
	params := &types.ReqWalletSendToAddress{From: from, To: to, Amount: amt, Note: note}
	if !isToken {
		params.IsToken = false
	} else {
		params.IsToken = true
		params.TokenSymbol = tokenSymbol
	}

	var res rpctypes.ReplyHash
	ctx := jsonclient.NewRPCCtx(rpcAddr, "Chain33.SendToAddress", params, &res)
	ctx.Run()
}

// CreateRawTx create rawtransaction func
func CreateRawTx(paraName string, to string, amount float64, note string, isWithdraw bool, tokenSymbol, execName string, cfg *rpctypes.ChainConfigInfo) (string, error) {
	if amount < 0 {
		return "", types.ErrAmount
	}
	if float64(types.MaxCoin) < amount {
		return "", types.ErrAmount
	}
	//检测to地址的合法性
	if to != "" {
		if err := address.CheckAddress(to, -1); err != nil {
			return "", types.ErrInvalidAddress
		}
	}

	amountInt64, err := types.FormatFloatDisplay2Value(amount, cfg.CoinPrecision)
	if err != nil {
		return "", err
	}

	if execName != "" && !types.IsAllowExecName([]byte(execName), []byte(execName)) {
		return "", types.ErrExecNameNotMatch
	}
	var tx *types.Transaction
	transfer := &cty.CoinsAction{}
	if !isWithdraw {
		if execName != "" {
			v := &cty.CoinsAction_TransferToExec{TransferToExec: &types.AssetsTransferToExec{Amount: amountInt64, Note: []byte(note), ExecName: execName, To: to}}
			transfer.Value = v
			transfer.Ty = cty.CoinsActionTransferToExec
		} else {
			v := &cty.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{Amount: amountInt64, Note: []byte(note), To: to}}
			transfer.Value = v
			transfer.Ty = cty.CoinsActionTransfer
		}
	} else {
		v := &cty.CoinsAction_Withdraw{Withdraw: &types.AssetsWithdraw{Amount: amountInt64, Note: []byte(note), ExecName: execName, To: to}}
		transfer.Value = v
		transfer.Ty = cty.CoinsActionWithdraw
	}
	execer := []byte(getRealExecName(paraName, cfg.CoinExec))
	if paraName == "" {
		tx = &types.Transaction{Execer: execer, Payload: types.Encode(transfer), To: to}
	} else {
		execAddr, err := address.GetExecAddress(string(execer), cfg.DefaultAddressID)
		if err != nil {
			return "", err
		}
		tx = &types.Transaction{Execer: execer, Payload: types.Encode(transfer), To: execAddr}
	}

	tx, err = types.FormatTxExt(cfg.ChainID, len(paraName) > 0, cfg.MinTxFeeRate, string(execer), tx)
	if err != nil {
		return "", err
	}
	txHex := types.Encode(tx)
	return hex.EncodeToString(txHex), nil
}

// GetExecAddr get exec address func
// addressID should based on server side(node)
func GetExecAddr(exec string, addressID int32) (string, error) {
	if ok := types.IsAllowExecName([]byte(exec), []byte(exec)); !ok {
		return "", types.ErrExecNameNotAllow
	}
	addrResult, err := address.GetExecAddress(exec, addressID)
	return addrResult, err
}

func getRealExecName(paraName string, name string) string {
	if strings.HasPrefix(name, "user.p.") {
		return name
	}
	return paraName + name
}

func parseTxHeight(expire string) error {
	if len(expire) == 0 {
		return errors.New("expire string should not be empty")
	}

	if expire[0] == 'H' && expire[1] == ':' {
		txHeight, err := strconv.Atoi(expire[2:])
		if err != nil {
			return err
		}
		if txHeight <= 0 {
			//fmt.Printf("txHeight should be grate to 0")
			return errors.New("txHeight should be grate to 0")
		}

		return nil
	}

	return errors.New("Invalid expire format. Should be one of {time:\"3600s/1min/1h\" block:\"123\" txHeight:\"H:123\"}")
}

// CheckExpireOpt parse expire option in command
func CheckExpireOpt(expire string) (string, error) {
	//时间格式123s/1m/1h
	expireTime, err := time.ParseDuration(expire)
	if err == nil {
		if expireTime < time.Minute*2 && expireTime != time.Second*0 {
			expire = "120s"
			fmt.Println("expire time must longer than 2 minutes, changed expire time into 2 minutes")
		}

		return expire, nil
	}

	//区块高度格式，123
	blockInt, err := strconv.Atoi(expire)
	if err == nil {
		if blockInt <= 0 {
			fmt.Printf("block height should be grate to 0")
			return "", errors.New("block height should be grate to 0")
		}
		return expire, nil
	}

	//Txheight格式，H:123
	err = parseTxHeight(expire)
	if err != nil {
		return "", err
	}

	return expire, err
}

// ReadFile 读取文件
func ReadFile(file string) ([]byte, error) {
	fileCont, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return fileCont, nil
}

// LoadPrivKeyFromLocal 加载账户
func LoadPrivKeyFromLocal(signType string, filePath string) (crypto.PrivKey, error) {
	if signType == "" {
		signType = secp_256k1
	}
	if signType == secp_256k1 {
		//TODO
		return nil, errors.New("not support")
	} else if signType == sm_2 {
		content, err := ReadFile(filePath)
		if err != nil {
			fmt.Println("GetKeyByte.read key file failed.", "file", filePath, "error", err.Error())
			return nil, err
		}
		keyBytes, err := common.FromHex(string(content))
		if err != nil {
			fmt.Println("GetKeyByte.FromHex.", "error", err.Error())
			return nil, err
		}
		if len(keyBytes) != sm2.SM2PrivateKeyLength {
			fmt.Println("GetKeyByte.private key length error", "len", len(keyBytes), "expect", sm2.SM2PrivateKeyLength)
			return nil, errors.New("private key length error")
		}
		driver := sm2.Driver{}
		privKey, err := driver.PrivKeyFromBytes(keyBytes)
		if err != nil {
			fmt.Println("load private key file  failed,err", err)
			return nil, err
		}
		return privKey, nil
	} else if signType == ed_25519 {
		return nil, errors.New("not support")
	} else {
		return nil, errors.New("sign type not support")
	}

}

// CreateTxWithCert 构造携带证书的交易
func CreateTxWithCert(signType string, privateKey crypto.PrivKey, hexTx string, certByte []byte) (string, error) {
	data, _ := common.FromHex(hexTx)
	var tx types.Transaction
	err := types.Decode(data, &tx)
	if err != nil {
		fmt.Println("decode tx failed", err)
		return "", err
	}
	signature := privateKey.Sign(data)
	if signType == sm_2 {
		sign := &types.Signature{
			Ty:        258,
			Pubkey:    privateKey.PubKey().Bytes(),
			Signature: signature.Bytes(),
		}
		tx.Signature = sign
		tx.Signature.Signature = utils.EncodeCertToSignature(signature.Bytes(), certByte, default_uid)
	} else {
		return "", errors.New("not support")
	}
	return common.ToHex(types.Encode(&tx)), nil
}

// GetChainConfig get system config parameters
func GetChainConfig(rpcAddr string) (*rpctypes.ChainConfigInfo, error) {
	rpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		return nil, errors.Wrap(err, "new json client")
	}
	var res rpctypes.ChainConfigInfo
	err = rpc.Call("Chain33.GetChainConfig", nil, &res)
	if err != nil {
		return nil, errors.Wrapf(err, "get chain config")
	}
	return &res, nil
}

// SendCreateTxRPC create transaction json rpc
func SendCreateTxRPC(cmd *cobra.Command, exec, actionName string, req types.Message) {
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	paraName, _ := cmd.Flags().GetString("paraName")
	payLoad := types.MustPBToJSON(req)
	pm := &rpctypes.CreateTxIn{
		Execer:     types.GetExecName(exec, paraName),
		ActionName: actionName,
		Payload:    payLoad,
	}

	var res string
	ctx := jsonclient.NewRPCCtx(rpcAddr, "Chain33.CreateTransaction", pm, &res)
	ctx.RunWithoutMarshal()
}
