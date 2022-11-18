package types

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/33cn/chain33/system/crypto/secp256k1eth"

	"github.com/33cn/chain33/common/address"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/system/address/eth"
	dtypes "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/ethereum/go-ethereum/common"

	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

// newRPCTransaction returns a transaction that will serialize to the RPC
func newRPCTransaction(tx *etypes.Transaction, blockHash common.Hash, blockNumber uint64, index uint64) *Transaction {
	var signer etypes.Signer
	if tx.Protected() {
		signer = etypes.LatestSignerForChainID(tx.ChainId())
	} else {
		signer = etypes.HomesteadSigner{}
	}
	from, _ := etypes.Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()

	result := &Transaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}

	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}

	switch tx.Type() {
	case etypes.AccessListTxType, etypes.DynamicFeeTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())

	}

	return result
}

//makeDERSigToRSV der sig data to rsv
func makeDERSigToRSV(eipSigner etypes.Signer, sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) == 65 {
		r = new(big.Int).SetBytes(sig[:32])
		s = new(big.Int).SetBytes(sig[32:64])
		v = new(big.Int).SetBytes([]byte{sig[64]})
		return
	}

	rb, sb, err := paraseDERCode(sig)
	if err != nil {
		log.Error("makeDERSigToRSV", "paraseDERCode err", err.Error(), "sig", hexutil.Encode(sig))
		return nil, nil, nil, err
	}

	signature := make([]byte, 65)
	copy(signature[32-len(rb):32], rb)
	copy(signature[64-len(sb):64], sb)
	signature[64] = 0x00
	r, s, v = decodeSignature(signature)
	if eipSigner.ChainID().Sign() != 0 {
		v = big.NewInt(int64(signature[64] + 35))
		v.Add(v, new(big.Int).Mul(eipSigner.ChainID(), big.NewInt(2)))
	}
	return r, s, v, nil
}

//CaculateRealV  return v
func CaculateRealV(v *big.Int, chainID uint64, txTyppe uint8) (byte, error) {

	switch txTyppe {
	case etypes.LegacyTxType:
		if chainID != 0 {
			chainIDMul := 2 * chainID
			return byte(v.Uint64() - chainIDMul - 35), nil // -8-27
		}
		return byte(v.Uint64() - 27), nil

	case etypes.DynamicFeeTxType:
		return byte(v.Uint64()), nil

	case etypes.AccessListTxType:
		return byte(v.Uint64() - 27), nil
	default:
		return 0, fmt.Errorf(fmt.Sprintf("no support this tx type:%v", txTyppe))
	}

}

func decodeSignature(sig []byte) (r, s, v *big.Int) {
	if len(sig) != crypto.SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v
}

func paraseDERCode(sig []byte) (r, s []byte, err error) {
	//0x30 [total-length] 0x02 [R-length] [R] 0x02 [S-length] [S]

	if len(sig) < 70 || len(sig) > 71 {
		return nil, nil, fmt.Errorf("wrong sig data size:%v,must beyound length 70 bytes", len(sig))
	}
	//3045022100af5778b81ae8817c6ae29fad8c1113d501e521c885a65c2c4d71763c4963984b022020687b73f5c90243dc16c99427d6593a711c52c8bf09ca6331cdd42c66edee74
	if sig[0] == 0x30 && sig[2] == 0x02 {
		r = sig[4 : int(sig[3])+4]
		if r[0] == 0x0 {
			r = r[1:]
		}
	}
	if sig[int(sig[3])+4] == 0x02 { //&&sig[int(sig[3])+5]==0x20
		s = sig[int(sig[3])+6 : int(sig[3])+6+int(sig[int(sig[3])+5])]
	}

	return
}

//makeDERsignature ...
func makeDERsignature(rb, sb []byte) []byte {
	if rb[0] > 0x7F {
		rb = append([]byte{0}, rb...)
	}

	if sb[0] > 0x7F {
		sb = append([]byte{0}, sb...)
	}
	// total length of returned signature is 1 byte for each magic and
	// length (6 total), plus lengths of r and s
	length := 6 + len(rb) + len(sb)
	b := make([]byte, length)

	b[0] = 0x30
	b[1] = byte(length - 2)
	b[2] = 0x02
	b[3] = byte(len(rb))
	offset := copy(b[4:], rb) + 4
	b[offset] = 0x02
	b[offset+1] = byte(len(sb))
	copy(b[offset+2:], sb)
	return b
}
func paraseChain33Tx(itx *ctypes.Transaction, blockHash common.Hash, blockNum int64, index uint64, cfg *ctypes.Chain33Config) *Transaction {
	eipSigner := etypes.LatestSignerForChainID(big.NewInt(secp256k1eth.GetEvmChainID()))
	var tx Transaction
	tx.Hash = common.BytesToHash(itx.Hash())
	tx.BlockHash = &blockHash
	tx.Type = etypes.LegacyTxType
	from := common.HexToAddress(itx.From())
	to := common.HexToAddress(itx.To)
	tx.From = from
	tx.To = &to

	tx.TransactionIndex = (*hexutil.Uint64)(&index)
	amount, err := itx.Amount()
	if err != nil {
		log.Error("paraseChain33Tx", "err", err)
		return nil
	}
	bamount := big.NewInt(amount)
	eamount := bamount.Mul(bamount, big.NewInt(1e10))
	tx.Value = (*hexutil.Big)(eamount)
	tx.Input = hexutil.Bytes{0}
	if strings.HasSuffix(string(itx.Execer), "evm") {
		var action ctypes.EVMContractAction4Chain33
		err := ctypes.Decode(itx.GetPayload(), &action)
		if err != nil {
			log.Error("paraseChain33Tx", "err", err)
			return nil
		}
		data := (hexutil.Bytes)(action.Para)
		tx.Input = data
		caddr := common.HexToAddress(action.ContractAddr)
		tx.To = &caddr
		if len(action.Code) != 0 {
			tx.Input = action.Code
			tx.To = &common.Address{}
		}
	} else if strings.HasSuffix(string(itx.Execer), "coins") {
		if cfg.IsPara() {
			var action dtypes.CoinsAction
			err := ctypes.Decode(itx.GetPayload(), &action)
			if err != nil {
				log.Error("paraseChain33Tx", "decode coinsAction err", err)
				return nil
			}
			transfer, ok := action.GetValue().(*dtypes.CoinsAction_Transfer)
			if ok {
				to := common.HexToAddress(transfer.Transfer.GetTo())
				tx.To = &to
			}
		}
	}

	r, s, v, err := makeDERSigToRSV(eipSigner, itx.Signature.GetSignature())
	if err != nil {
		log.Error("makeDERSigToRSV", "err", err)
		return nil
	}

	tx.V = (*hexutil.Big)(v)
	tx.R = (*hexutil.Big)(r)
	tx.S = (*hexutil.Big)(s)

	var nonce = uint64(itx.Nonce)
	var gas = uint64(itx.GetTxFee())
	tx.Nonce = (hexutil.Uint64)(nonce)
	tx.GasPrice = (*hexutil.Big)(big.NewInt(10e9))
	tx.Gas = (hexutil.Uint64)(gas)
	tx.BlockNumber = (*hexutil.Big)(big.NewInt(blockNum))
	return &tx
}

func paraseChain33TxPayload(execer string, payload []byte, blockHash common.Hash, blockNum uint64) *Transaction {
	var note []byte
	if strings.HasSuffix(execer, "evm") {
		var evmaction ctypes.EVMContractAction4Chain33
		err := ctypes.Decode(payload, &evmaction)
		if err == nil {
			if evmaction.GetNote() != "" {
				note = common.FromHex(evmaction.GetNote())
			} else {
				return nil
			}
		}

	} else {
		var coinsaction dtypes.CoinsAction
		err := ctypes.Decode(payload, &coinsaction)
		if err == nil {
			transfer, ok := coinsaction.GetValue().(*dtypes.CoinsAction_Transfer)
			if ok && len(transfer.Transfer.GetNote()) != 0 {
				note = transfer.Transfer.GetNote()
			}
		}
	}

	var etx etypes.Transaction
	err := etx.UnmarshalBinary(note)
	if err == nil {
		return newRPCTransaction(&etx, blockHash, blockNum, 0)
	}
	return nil

}

//TxsToEthTxs chain33 txs format transfer to eth txs format
func TxsToEthTxs(blockHash common.Hash, blockNum int64, ctxs []*ctypes.Transaction, cfg *ctypes.Chain33Config, full bool) (txs []interface{}, fee int64, err error) {
	for index, itx := range ctxs {
		fee += itx.GetFee()
		if !full {
			txs = append(txs, common.Bytes2Hex(itx.Hash()))
			continue
		}
		if itx == nil {
			continue
		}
		tx := paraseChain33TxPayload(string(itx.GetExecer()), itx.GetPayload(), blockHash, uint64(blockNum))
		//重置交易哈希
		if tx != nil {
			tx.Hash = common.BytesToHash(itx.Hash())
			txs = append(txs, tx)
		} else {

			tx = paraseChain33Tx(itx, blockHash, blockNum, uint64(index), cfg)
			if tx != nil {
				txs = append(txs, tx)
			}
		}

	}
	return txs, fee, nil
}

//TxDetailsToEthReceipts chain33 txdetails transfer to eth tx receipts
func TxDetailsToEthReceipts(txDetails *ctypes.TransactionDetails, blockHash common.Hash, cfg *ctypes.Chain33Config) (txs Transactions, receipts []*Receipt, err error) {
	for index, detail := range txDetails.GetTxs() {
		if detail.GetTx() == nil {
			continue
		}

		tx := paraseChain33TxPayload(string(detail.GetTx().GetExecer()), detail.GetTx().GetPayload(), blockHash, uint64(detail.Height))
		if tx == nil {
			tx = paraseChain33Tx(detail.GetTx(), blockHash, detail.GetHeight(), uint64(index), cfg)
			if tx == nil {
				continue
			}

		}

		tx.Hash = common.BytesToHash(detail.GetTx().Hash())
		txs = append(txs, tx)
		var receipt Receipt
		if tx.Input.String() != "0x" {
			receipt.ContractAddress = tx.To
		}
		receipt.From = &tx.From
		if detail.Receipt.Ty == ctypes.ExecOk { //success
			receipt.Status = 1
		} else {
			receipt.Status = 0
		}
		var gas uint64
		receipt.Logs, receipt.ContractAddress, gas = receiptLogs2EvmLog(detail, blockHash, nil)
		if receipt.Logs == nil {
			receipt.Logs = []*EvmLog{}
		}

		receipt.GasUsed = hexutil.Uint64(gas)
		if receipt.GasUsed == 0 {
			receipt.GasUsed = hexutil.Uint64(detail.Tx.Fee)

		}
		receipt.Type = tx.Type
		receipt.CumulativeGasUsed = receipt.GasUsed
		receipt.Bloom = CreateBloom([]*Receipt{&receipt})
		receipt.TxHash = common.BytesToHash(detail.GetTx().Hash())
		receipt.BlockNumber = (*hexutil.Big)(big.NewInt(detail.Height))
		receipt.TransactionIndex = hexutil.Uint(uint64(detail.GetIndex()))
		receipts = append(receipts, &receipt)
	}
	return
}

func receiptLogs2EvmLog(detail *ctypes.TransactionDetail, blockHash common.Hash, option *SubLogs) (elogs []*EvmLog, contractorAddr *common.Address, gasused uint64) {
	var cAddr common.Address
	var filterTopics = make(map[string]bool)
	if option != nil {
		for _, topic := range option.Topics {
			filterTopics[topic] = true
		}
	}
	var index int

	for _, lg := range detail.Receipt.Logs {

		if lg.Ty != 605 && lg.Ty != 603 { //evm event
			continue
		}
		var evmLog ctypes.EVMLog

		if lg.Ty == 605 {
			err := ctypes.Decode(lg.Log, &evmLog)
			if nil != err {
				log.Error("receiptLogs2EvmLog", "decode evmlog", err.Error())
				continue
			}

		}

		if lg.Ty == 603 { //获取消费的GAS
			var recp rpctypes.ReceiptData
			recp.Ty = 2
			recp.Logs = append(recp.Logs, &rpctypes.ReceiptLog{Ty: lg.Ty, Log: common.Bytes2Hex(lg.Log)})
			recpResult, err := rpctypes.DecodeLog([]byte("evm"), &recp)
			if err != nil {
				log.Error("receiptLogs2EvmLog", "Failed to DecodeLog for type", err)
				continue
			}
			var receiptEVMContract struct {
				Caller       string ` json:"caller,omitempty"`
				ContractName string ` json:"contractName,omitempty"`
				ContractAddr string ` json:"contractAddr,omitempty"`
				UsedGas      string `json:"usedGas,omitempty"`
				// 创建合约返回的代码
				Ret string `json:"ret,omitempty"`
				//  json格式化后的返回值
				JSONRet string ` json:"jsonRet,omitempty"`
			}

			jlg, _ := json.Marshal(recpResult.Logs[0].Log)
			err = json.Unmarshal(jlg, &receiptEVMContract)
			if nil == err {
				log.Info("receiptLogs2EvmLog", "gasused:", receiptEVMContract.UsedGas)
				bn, ok := big.NewInt(1).SetString(receiptEVMContract.UsedGas, 10)
				if ok {
					gasused = bn.Uint64()
				}
				cAddr = common.HexToAddress(receiptEVMContract.ContractAddr)
				contractorAddr = &cAddr

			}
		}

		var elog EvmLog
		elog.TxIndex = hexutil.Uint(detail.GetIndex())
		elog.Index = hexutil.Uint(index)
		elog.Address = &cAddr
		elog.TxHash = common.BytesToHash(detail.GetTx().Hash())
		elog.BlockNumber = hexutil.Uint64(detail.Height)
		elog.BlockHash = blockHash
		for _, topic := range evmLog.Topic {
			if option != nil {
				if _, ok := filterTopics[hexutil.Encode(topic)]; !ok {
					continue
				}
			}
			elog.Topics = append(elog.Topics, common.BytesToHash(topic))
			if len(evmLog.Data) != 0 {
				elog.Data = (*hexutil.Bytes)(&evmLog.Data)
			} else {
				elog.Data = nil
			}

		}

		if lg.Ty == 605 {
			elogs = append(elogs, &elog)
			index++
		}

	}

	return
}

//FilterEvmLogs filter evm logs by option
func FilterEvmLogs(logs *ctypes.EVMTxLogPerBlk, option *SubLogs) (evmlogs []*EvmLogInfo) {
	var addr string
	var filterTopics = make(map[string]bool)
	if option != nil {
		for _, topic := range option.Topics {
			if topic == "" {
				continue
			}
			filterTopics[topic] = true
		}
	}

	if option != nil {
		addr = option.Address
	}
	for i, txlog := range logs.TxAndLogs {
		var info EvmLogInfo
		if txlog.GetTx().GetTo() == addr {
			for j, tlog := range txlog.GetLogsPerTx().GetLogs() {
				var topics []string
				if _, ok := filterTopics[hexutil.Encode(tlog.Topic[0])]; ok || len(filterTopics) == 0 {
					topics = append(topics, hexutil.Encode(tlog.Topic[0]))
				}

				if len(topics) != 0 {
					info.LogIndex = hexutil.EncodeUint64(uint64(j))
					info.Topics = topics
				}
				info.Address = addr
				info.TransactionIndex = hexutil.EncodeUint64(uint64(i))
				info.BlockHash = hexutil.Encode(logs.BlockHash)
				info.TransactionHash = hexutil.Encode(txlog.GetTx().Hash())
				info.BlockNumber = hexutil.EncodeUint64(uint64(logs.Height))
				evmlogs = append(evmlogs, &info)
			}
		}
	}

	return
}

// CreateBloom creates a bloom filter out of the give Receipts (+Logs)
func CreateBloom(receipts []*Receipt) etypes.Bloom {
	var bin etypes.Bloom
	for _, receipt := range receipts {
		for _, rlog := range receipt.Logs {
			if rlog.Address != nil && rlog.Address.Bytes() != nil {
				bin.Add(rlog.Address.Bytes())
			}

			for _, b := range rlog.Topics {
				bin.Add(b[:])
			}
		}
	}
	return bin
}

//AssembleChain33Tx 通过eth tx 组装chain33 tx 全部走evm 通道
func AssembleChain33Tx(etx *etypes.Transaction, sig, pubkey []byte, cfg *ctypes.Chain33Config) *ctypes.Transaction {
	rawData, err := etx.MarshalBinary()
	if err != nil {
		log.Error("AssembleChain33Tx", "tx.MarshalBinary err", err.Error())
		return nil
	}

	var exec = cfg.ExecName("evm")
	var amount int64
	ethUnit := big.NewInt(1e18)
	if ctypes.DefaultCoinPrecision > int64(1e18) {
		log.Error("AssembleChain33Tx", "no support coinsPrecision", ctypes.DefaultCoinPrecision)
		return nil
	}
	if etx.Value() != nil {
		//amount = etx.Value().Div(etx.Value(), big.NewInt(1).SetUint64(1e10)).Int64()
		amount = etx.Value().Div(etx.Value(), ethUnit.Div(ethUnit, big.NewInt(1).SetInt64(ctypes.DefaultCoinPrecision))).Int64()

	}
	action := &ctypes.EVMContractAction4Chain33{
		Amount:       uint64(amount),
		GasLimit:     etx.Gas(),
		GasPrice:     1,
		Code:         nil,
		Para:         nil,
		Alias:        "",
		Note:         common.Bytes2Hex(rawData),
		ContractAddr: "",
	}

	var to string
	if len(etx.Data()) != 0 { //合约操作
		packdata := etx.Data()
		if etx.To() == nil || len(etx.To().Bytes()) == 0 {
			//合约部署
			action.Code = packdata
			to = address.ExecAddress(exec)
		} else {
			//合约操作
			action.Para = packdata
			to = etx.To().String()
		}
		action.ContractAddr = to

	} else { // coins 操作
		to = etx.To().String()
		//coins 转账,para为目的地址
		action.Para = common.FromHex(to)
		//ContractAddr 为执行器地址
		action.ContractAddr = address.ExecAddress(exec)
	}
	payload := ctypes.Encode(action)
	var gas = etx.Gas()
	if gas < 1e5 { //gas 不能低于1e5
		gas = 1e5
	}

	//全部走Evm 通道，exec=evm
	var chain33Tx = &ctypes.Transaction{
		ChainID: cfg.GetChainID(), //与链节点的chainID保持一致
		To:      to,
		Execer:  []byte(exec),
		Payload: payload,
		Fee:     int64(gas),
		Signature: &ctypes.Signature{
			Ty:        ctypes.EncodeSignID(ctypes.SECP256K1ETH, eth.ID),
			Pubkey:    pubkey,
			Signature: sig,
		},
	}
	//为了防止认为设置过高的nonce,挤占mempool空间，允许最大3小时的超时时间
	chain33Tx.SetExpire(cfg, time.Hour*3)
	chain33Tx.Nonce = int64(etx.Nonce())
	if cfg.IsPara() {
		chain33Tx.To = address.ExecAddress(string(chain33Tx.Execer))
	}
	return chain33Tx
}
