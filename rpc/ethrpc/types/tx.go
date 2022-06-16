package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	rpctypes "github.com/33cn/chain33/rpc/types"

	"github.com/ethereum/go-ethereum/common"

	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

//makeDERSigToRSV der sig data to rsv
func MakeDERSigToRSV(eipSigner etypes.EIP155Signer, sig []byte) (r, s, v *big.Int, err error) {
	//fmt.Println("len:",len(sig),"sig",hexutil.Encode(sig))
	if len(sig) == 65 {
		r = new(big.Int).SetBytes(sig[:32])
		s = new(big.Int).SetBytes(sig[32:64])
		v = new(big.Int).SetBytes([]byte{sig[64]})
		return
	}
	//return

	rb, sb, err := paraseDERCode(sig)
	if err != nil {
		fmt.Println("makeDERSigToRSV", "err", err.Error(), "sig", hexutil.Encode(sig))
		return nil, nil, nil, err
	}
	var signature []byte
	signature = append(signature, rb...)
	signature = append(signature, sb...)
	signature = append(signature, 0x00)
	r, s, v = decodeSignature(signature)
	if eipSigner.ChainID().Sign() != 0 {
		v = big.NewInt(int64(signature[64] + 35))
		v.Add(v, new(big.Int).Mul(eipSigner.ChainID(), big.NewInt(2)))
	}
	return r, s, v, nil
}

//CaculateV  return v
func CaculateV(v *big.Int, chainID uint64, txTyppe uint8) (byte, error) {

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
		return 0, errors.New(fmt.Sprintf("no support this tx type:%v", txTyppe))
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

func MakeDERsignature(rb, sb []byte) []byte {
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

//TxsToEthTxs chain33 txs format transfer to eth txs format
func TxsToEthTxs(blocknum int64, ctxs []*ctypes.Transaction, cfg *ctypes.Chain33Config, full bool) (txs []interface{}, fee int64, err error) {
	eipSigner := etypes.NewEIP155Signer(big.NewInt(int64(cfg.GetChainID())))
	for _, itx := range ctxs {
		var tx Transaction
		tx.Hash = common.BytesToHash(itx.Hash())
		if !full {
			txs = append(txs, tx.Hash)
			continue
		}
		fee += itx.Fee

		tx.Type = etypes.LegacyTxType
		from := common.HexToAddress(itx.From())
		to := common.HexToAddress(itx.To)
		tx.From = &from
		tx.To = &to
		amount, err := itx.Amount()
		if err != nil {
			log.Error("TxsToEthTxs", "err", err)
			return nil, fee, err
		}
		tx.Value = (*hexutil.Big)(big.NewInt(amount))
		tx.Input = &hexutil.Bytes{0}
		if strings.HasSuffix(string(itx.Execer), "evm") {
			var action ctypes.EVMContractAction4Chain33
			err := ctypes.Decode(itx.GetPayload(), &action)
			if err != nil {
				log.Error("TxDetailsToEthTx", "err", err)
				continue
			}
			data := (hexutil.Bytes)(action.Para)
			tx.Input = &data
			//如果是EVM交易，to为合约交易
			caddr := common.HexToAddress(action.ContractAddr)
			tx.To = &caddr
		}

		r, s, v, err := MakeDERSigToRSV(eipSigner, itx.Signature.GetSignature())
		if err != nil {
			log.Error("makeDERSigToRSV", "err", err)
			txs = append(txs, &tx)
			continue
		}

		tx.V = (*hexutil.Big)(v)
		tx.R = (*hexutil.Big)(r)
		tx.S = (*hexutil.Big)(s)

		var nonce = (uint64(itx.Nonce))
		var gas = uint64(itx.Fee)
		tx.Nonce = (*hexutil.Uint64)(&nonce)
		tx.GasPrice = (*hexutil.Big)(big.NewInt(10e9))
		tx.Gas = (*hexutil.Uint64)(&gas)
		tx.MaxFeePerGas = (*hexutil.Big)(big.NewInt(10e9))
		tx.MaxPriorityFeePerGas = (*hexutil.Big)(big.NewInt(10e9))
		tx.BlockNumber = (*hexutil.Big)(big.NewInt(blocknum))

		txs = append(txs, &tx)
	}
	return txs, fee, nil
}

//TxDetailsToEthTx chain33 txdetails transfer to eth tx
func TxDetailsToEthTx(txdetails *ctypes.TransactionDetails, cfg *ctypes.Chain33Config) (txs Transactions, receipts []*Receipt, err error) {
	for _, detail := range txdetails.GetTxs() {
		var tx Transaction
		var receipt Receipt
		//处理 execer=EVM
		to := common.HexToAddress(detail.GetTx().GetTo())
		tx.To = &to
		//receipt.To = tx.To
		//var data []byte

		if strings.HasSuffix(string(detail.GetTx().Execer), "evm") {
			var action ctypes.EVMContractAction4Chain33
			err := ctypes.Decode(detail.GetTx().GetPayload(), &action)
			if err != nil {
				log.Error("TxDetailsToEthTx", "err", err)
				continue
			}
			var decodeData []byte
			if len(action.Para) != 0 {
				decodeData = action.Para
			} else {
				//合约部署
				decodeData = action.Code
			}
			hdata := (hexutil.Bytes)(decodeData)
			tx.Input = &hdata
			//如果是EVM交易，to为合约交易
			if action.Para != nil {
				caddr := common.HexToAddress(action.ContractAddr)
				tx.To = &caddr
				receipt.ContractAddress = &caddr
			} else {
				tx.To = nil

			}

		}
		from := common.HexToAddress(detail.Fromaddr)
		tx.From = &from
		receipt.From = tx.From
		tx.Value = (*hexutil.Big)(big.NewInt(detail.GetAmount()))       //fmt.Sprintf("0x%x",detail.GetAmount())/
		tx.BlockNumber = (*hexutil.Big)(big.NewInt(detail.GetHeight())) //fmt.Sprintf("0x%x",detail.Height)
		//tx.TransactionIndex = hexutil.Uint(uint(detail.GetIndex()))   //fmt.Sprintf("0x%x",detail.GetIndex())
		eipSigner := etypes.NewEIP155Signer(big.NewInt(int64(cfg.GetChainID())))
		r, s, v, err := MakeDERSigToRSV(eipSigner, detail.Tx.GetSignature().GetSignature())
		if err != nil {
			return nil, nil, err
		}
		tx.V = (*hexutil.Big)(v)
		tx.R = (*hexutil.Big)(r)
		tx.S = (*hexutil.Big)(s)
		tx.Hash = common.BytesToHash(detail.Tx.Hash())
		tx.ChainID = (*hexutil.Big)(big.NewInt(int64(detail.Tx.ChainID)))
		tx.Type = 2
		receipt.From = tx.From
		if detail.Receipt.Ty == 2 { //success
			receipt.Status = 1
		} else {
			receipt.Status = 0
		}
		var gas uint64
		receipt.Logs, receipt.ContractAddress, gas = receiptLogs2EvmLog(detail.Receipt.Logs, nil)

		receipt.GasUsed = hexutil.Uint64(gas)
		if receipt.GasUsed == 0 {
			receipt.GasUsed = hexutil.Uint64(detail.Tx.Fee)

		}
		receipt.CumulativeGasUsed = receipt.GasUsed
		receipt.Bloom = CreateBloom([]*Receipt{&receipt})
		receipt.TxHash = common.BytesToHash(detail.GetTx().Hash())
		receipt.BlockNumber = (*hexutil.Big)(big.NewInt(detail.Height))
		receipt.TransactionIndex = hexutil.Uint(uint64(detail.GetIndex()))
		tx.Gas = &receipt.GasUsed
		tx.GasPrice = (*hexutil.Big)(big.NewInt(10e9))
		txs = append(txs, &tx)

		receipts = append(receipts, &receipt)
	}
	return
}

func receiptLogs2EvmLog(logs []*ctypes.ReceiptLog, option *SubLogs) (elogs []*EvmLog, contractorAddr *common.Address, gasused uint64) {
	var filterTopics = make(map[string]bool)
	if option != nil {
		for _, topic := range option.Topics {
			filterTopics[topic] = true
		}
	}
	var index int
	for _, lg := range logs {
		if lg.Ty != 605 && lg.Ty != 603 { //evm event
			continue
		}

		var evmLog ctypes.EVMLog
		err := ctypes.Decode(lg.Log, &evmLog)
		if nil != err {
			continue
		}
		if lg.Ty == 603 { //获取消费的GAS

			var recp rpctypes.ReceiptData
			recp.Ty = 2
			recp.Logs = append(recp.Logs, &rpctypes.ReceiptLog{Ty: lg.Ty, Log: common.Bytes2Hex(lg.Log)})
			recpResult, err := rpctypes.DecodeLog([]byte("evm"), &recp)
			if err != nil {
				log.Error("GetTxByHashes", "Failed to DecodeLog for type", err)
				continue
			}
			var receiptEVMContract struct {
				Caller       string ` json:"caller,omitempty"`
				ContractName string ` json:"contractName,omitempty"`
				ContractAddr string ` json:"contractAddr,omitempty"`
				UsedGas      string `json:"usedGas,omitempty"`
				// 创建合约返回的代码
				Ret string `json:"ret,omitempty"`
				// json格式化后的返回值
				JsonRet string ` json:"jsonRet,omitempty"`
			}

			jlg, _ := json.Marshal(recpResult.Logs[0].Log)
			log.Info("receiptLogs2EvmLog", "jlg:", string(jlg))
			err = json.Unmarshal(jlg, &receiptEVMContract)
			if nil == err {
				log.Info("receiptLogs2EvmLog", "gasused:", receiptEVMContract.UsedGas)
				bn, ok := big.NewInt(1).SetString(receiptEVMContract.UsedGas, 10)
				if ok {
					gasused = bn.Uint64()
				}
				cadr := common.HexToAddress(receiptEVMContract.ContractAddr)
				contractorAddr = &cadr

				//log.Info("receiptLogs2EvmLog", "gasused:", gasused)
			}
			continue
		}
		var log EvmLog
		log.Data = evmLog.Data
		log.Index = hexutil.Uint(index)
		for _, topic := range evmLog.Topic {
			if option != nil {
				if _, ok := filterTopics[hexutil.Encode(topic)]; !ok {
					continue
				}
			}

			log.Topics = append(log.Topics, common.BytesToHash(topic))
		}
		index++
		elogs = append(elogs, &log)
	}
	return
}

//FilterEvmLogs filter evm logs by option
func FilterEvmLogs(logs *ctypes.EVMTxLogPerBlk, option *SubLogs) (evmlogs []*EvmLogInfo) {
	var address string
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
		address = option.Address
	}
	for i, txlog := range logs.TxAndLogs {
		var info EvmLogInfo
		if txlog.GetTx().GetTo() == address {
			for j, tlog := range txlog.GetLogsPerTx().GetLogs() {
				var topics []string
				if _, ok := filterTopics[hexutil.Encode(tlog.Topic[0])]; ok || len(filterTopics) == 0 {
					topics = append(topics, hexutil.Encode(tlog.Topic[0]))
				}

				if len(topics) != 0 {
					info.LogIndex = hexutil.EncodeUint64(uint64(j))
					info.Topics = topics
				}
				info.Address = address
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

func ParaseEthSigData(sig []byte, chainID int32) ([]byte, error) {
	if len(sig) != 65 {
		return nil, errors.New("invalid pub key byte,must be 65 bytes")
	}

	if sig[64] != 0 && sig[64] != 1 {
		if sig[64] == 27 || sig[64] == 28 {
			sig[64] = sig[64] - 27
			return sig, nil

		} else {
			//salt := 1
			//if chainID == 0 {
			//	salt = 0
			//}
			chainIDMul := 2 * chainID
			sig[64] = sig[64] - byte(chainIDMul) - 35
			return sig, nil
		}

	}

	return sig, nil

}

// CreateBloom creates a bloom filter out of the give Receipts (+Logs)
func CreateBloom(receipts []*Receipt) etypes.Bloom {
	var bin etypes.Bloom
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			if log.Address.Bytes() != nil {
				bin.Add(log.Address.Bytes())
			}

			for _, b := range log.Topics {
				bin.Add(b[:])
			}
		}
	}
	return bin
}
