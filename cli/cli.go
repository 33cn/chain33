package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	jsonrpc "code.aliyun.com/chain33/chain33/rpc"
	"code.aliyun.com/chain33/chain33/types"
)

func main() {
	common.SetLogLevel("eror")
	//	argsWithProg := os.Args
	if len(os.Args) == 1 {
		LoadHelp()
		return
	}
	argsWithoutProg := os.Args[1:]
	switch argsWithoutProg[0] {
	case "-h": //使用帮助
		LoadHelp()
	case "--help": //使用帮助
		LoadHelp()
	case "help": //使用帮助
		LoadHelp()
	case "lock": //锁定
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		Lock()
	case "unlock": //解锁
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		UnLock(argsWithoutProg[1], argsWithoutProg[2])
	case "setpasswd": //重设密码
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SetPasswd(argsWithoutProg[1], argsWithoutProg[2])
	case "setlabl": //设置标签
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SetLabl(argsWithoutProg[1], argsWithoutProg[2])
	case "newaccount": //新建账户
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NewAccount(argsWithoutProg[1])
	case "getaccounts": //获取账户列表
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetAccounts()
	case "mergebalance": //合并余额
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		MergeBalance(argsWithoutProg[1])
	case "settxfee": //设置交易费
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SetTxFee(argsWithoutProg[1])
	case "sendtoaddress": //发送到地址
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SendToAddress(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	case "importprivkey": //引入私钥
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		ImportPrivKey(argsWithoutProg[1], argsWithoutProg[2])
	case "wallettxlist": //钱包交易列表
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		WalletTransactionList(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	case "getmempool": //获取Mempool
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetMemPool()
	case "sendtransaction": //发送交易
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SendTransaction(argsWithoutProg[1])
	case "querytransaction": //查询交易
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		if len(argsWithoutProg[1]) != 66 {
			fmt.Print(errors.New("哈希错误").Error())
			return
		}
		QueryTransaction(argsWithoutProg[1])
	case "gettxbyaddr": //根据地址获取交易
		if len(argsWithoutProg) != 7 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetTransactionByAddr(argsWithoutProg[1], argsWithoutProg[2],
			argsWithoutProg[3], argsWithoutProg[4],
			argsWithoutProg[5], argsWithoutProg[6])
	case "gettxbyhashes": //根据哈希数组获取交易
		if len(argsWithoutProg) < 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		for _, v := range argsWithoutProg[1:] {
			if len(v) != 66 {
				fmt.Print(errors.New("哈希错误").Error())
				return
			}
		}
		GetTransactionByHashes(argsWithoutProg[1:])
	case "getblocks": //获取区块
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetBlocks(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	case "getlastheader": //获取上一区块头
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetLastHeader()
	case "getheaders":
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetHeaders(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	case "getpeerinfo": //获取对等点信息
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetPeerInfo()
	case "getlastmempool":
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetLastMempool()
	case "getblockoverview":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetBlockOverview(argsWithoutProg[1])
	case "getaddroverview":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetAddrOverview(argsWithoutProg[1])
	case "getblockhash":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetBlockHash(argsWithoutProg[1])
	//seed
	case "genseed":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GenSeed(argsWithoutProg[1])
	case "saveseed":
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SaveSeed(argsWithoutProg[1], argsWithoutProg[2])
	case "getseed":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetSeed(argsWithoutProg[1])

	case "getwalletstatus": //获取钱包的状态
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetWalletStatus()
	case "getbalance":
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetBalance(argsWithoutProg[1], argsWithoutProg[2])
	case "getexecaddr":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetExecAddr(argsWithoutProg[1])
	case "bindminer":
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		BindMiner(argsWithoutProg[1], argsWithoutProg[2])
	default:
		fmt.Print("指令错误")
	}
}

func LoadHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("lock []                                                     : 锁定")
	fmt.Println("unlock [password, timeout]                                  : 解锁")
	fmt.Println("setpasswd [oldpassword, newpassword]                        : 设置密码")
	fmt.Println("setlabl [address, label]                                    : 设置标签")
	fmt.Println("newaccount [labelname]                                      : 新建账户")
	fmt.Println("getaccounts []                                              : 获取账户列表")
	fmt.Println("mergebalance [to]                                           : 合并余额")
	fmt.Println("settxfee [amount]                                           : 设置交易费")
	fmt.Println("sendtoaddress [from, to, amount, note]                      : 发送交易到地址")
	fmt.Println("importprivkey [privkey, label]                              : 引入私钥")
	fmt.Println("wallettxlist [from, count, direction]                       : 钱包交易列表")
	fmt.Println("getmempool []                                               : 获取内存池")
	fmt.Println("sendtransaction [data]                                      : 发送交易")
	fmt.Println("querytransaction [hash]                                     : 按哈希查询交易")
	fmt.Println("gettxbyaddr [address, flag, count, direction, height, index]: 按地址获取交易")
	fmt.Println("gettxbyhashes [hashes...]                                   : 按哈希列表获取交易")
	fmt.Println("getblocks [start, end, isdetail]                            : 获取指定区间区块")
	fmt.Println("getlastheader []                                            : 获取最新区块头")
	fmt.Println("getheaders [start, end, isdetail]                           : 获取指定区间区块头")
	fmt.Println("getpeerinfo []                                              : 获取远程节点信息")
	fmt.Println("getlastmempool []                                           : 获取最新加入到内存池中的十条交易")
	fmt.Println("getblockoverview [hash]                                     : 获取区块信息")
	fmt.Println("getaddroverview [address]                                   : 获取地址交易信息")
	fmt.Println("getblockhash [height]                                       : 获取区块哈希值")
	//seed
	fmt.Println("genseed [lang]                                              : 生成随机的种子,lang=0:英语，lang=1:简体汉字")
	fmt.Println("saveseed [seed,password]                                    : 保存种子并用密码加密,种子要求15个单词或者汉字,参考genseed输出格式")
	fmt.Println("getseed [password]                                          : 通过密码获取种子")
	fmt.Println("getwalletstatus []                                          : 获取钱包的状态")
	fmt.Println("getbalance [address, execer]                                : 查询地址余额")
	fmt.Println("getexecaddr [execer]                                        : 获取执行器地址")
	fmt.Println("bindminer [mineraddr, privkey]                              : 绑定挖矿地址")
}

type AccountsResult struct {
	Wallets []*WalletResult `json:"wallets"`
}

type WalletResult struct {
	Acc   *AccountResult `json:"acc,omitempty"`
	Label string         `json:"label,omitempty"`
}

type AccountResult struct {
	Currency int32  `json:"currency,omitempty"`
	Balance  string `json:"balance,omitempty"`
	Frozen   string `json:"frozen,omitempty"`
	Addr     string `json:"addr,omitempty"`
}

type TxListResult struct {
	Txs []*TxResult `json:"txs"`
}

type TxResult struct {
	Execer    string             `json:"execer"`
	Payload   string             `json:"payload"`
	Signature *jsonrpc.Signature `json:"signature"`
	Fee       string             `json:"fee"`
	Expire    int64              `json:"expire"`
	Nonce     int64              `json:"nonce"`
	To        string             `json:"to"`
}

type TxDetailResult struct {
	Tx        *TxResult            `json:"tx"`
	Receipt   *jsonrpc.ReceiptData `json:"receipt"`
	Proofs    []string             `json:"proofs"`
	Height    int64                `json:"height"`
	Index     int64                `json:"index"`
	Blocktime int64                `json:"blocktime"`
	Amount    string               `json:"amount"`
	Fromaddr  string               `json:"fromaddr"`
}

type TxDetailsResult struct {
	Txs []*TxDetailResult `json:"txs"`
}

type BlockResult struct {
	Version    int64       `json:"version"`
	ParentHash string      `json:"parenthash"`
	TxHash     string      `json:"txhash"`
	StateHash  string      `json:"statehash"`
	Height     int64       `json:"height"`
	BlockTime  int64       `json:"blocktime"`
	Txs        []*TxResult `json:"txs"`
}

type BlockDetailResult struct {
	Block    *BlockResult           `json:"block"`
	Receipts []*jsonrpc.ReceiptData `json:"receipts"`
}

type BlockDetailsResult struct {
	Items []*BlockDetailResult `json:"items"`
}

type WalletTxDetailsResult struct {
	TxDetails []*WalletTxDetailResult `json:"txDetails"`
}

type WalletTxDetailResult struct {
	Tx        *TxResult            `json:"tx"`
	Receipt   *jsonrpc.ReceiptData `json:"receipt"`
	Height    int64                `json:"height"`
	Index     int64                `json:"index"`
	Blocktime int64                `json:"blocktime"`
	Amount    string               `json:"amount"`
	Fromaddr  string               `json:"fromaddr"`
	Txhash    string               `json:"txhash"`
}

type AddrOverviewResult struct {
	Reciver string `json:"reciver"`
	Balance string `json:"balance"`
	TxCount int64  `json:"txCount"`
}

func Lock() {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.Reply
	err = rpc.Call("Chain33.Lock", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func UnLock(passwd string, timeout string) {
	timeoutInt64, err := strconv.ParseInt(timeout, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.WalletUnLock{Passwd: passwd, Timeout: timeoutInt64}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.Reply
	err = rpc.Call("Chain33.UnLock", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func SetPasswd(oldpass string, newpass string) {
	params := types.ReqWalletSetPasswd{Oldpass: oldpass, Newpass: newpass}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.Reply
	err = rpc.Call("Chain33.SetPasswd", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func SetLabl(addr string, label string) {
	params := types.ReqWalletSetLabel{Addr: addr, Label: label}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.WalletAccount
	err = rpc.Call("Chain33.SetLabl", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	balanceResult := strconv.FormatFloat(float64(res.GetAcc().GetBalance())/float64(1e8), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(res.GetAcc().GetFrozen())/float64(1e8), 'f', 4, 64)
	accResult := &AccountResult{
		Addr:     res.GetAcc().GetAddr(),
		Currency: res.GetAcc().GetCurrency(),
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	result := WalletResult{Acc: accResult, Label: res.GetLabel()}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func NewAccount(lb string) {
	params := types.ReqNewAccount{Label: lb}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.WalletAccount
	err = rpc.Call("Chain33.NewAccount", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	balanceResult := strconv.FormatFloat(float64(res.GetAcc().GetBalance())/float64(1e8), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(res.GetAcc().GetFrozen())/float64(1e8), 'f', 4, 64)
	accResult := &AccountResult{
		Addr:     res.GetAcc().GetAddr(),
		Currency: res.GetAcc().GetCurrency(),
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	result := WalletResult{Acc: accResult, Label: res.GetLabel()}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetAccounts() {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.WalletAccounts
	err = rpc.Call("Chain33.GetAccounts", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var result AccountsResult

	for _, r := range res.Wallets {
		balanceResult := strconv.FormatFloat(float64(r.Acc.Balance)/float64(1e8), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(r.Acc.Frozen)/float64(1e8), 'f', 4, 64)
		accResult := &AccountResult{
			Currency: r.Acc.Currency,
			Addr:     r.Acc.Addr,
			Balance:  balanceResult,
			Frozen:   frozenResult,
		}
		result.Wallets = append(result.Wallets, &WalletResult{Acc: accResult, Label: r.Label})
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func MergeBalance(to string) {
	params := types.ReqWalletMergeBalance{To: to}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyHashes
	err = rpc.Call("Chain33.MergeBalance", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func SetTxFee(amount string) {
	amountFloat64, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64 := int64(amountFloat64 * 1e4)
	params := types.ReqWalletSetFee{Amount: amountInt64 * 1e4}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.Reply
	err = rpc.Call("Chain33.SetTxFee", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func SendToAddress(from string, to string, amount string, note string) {
	amountFloat64, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64 := int64(amountFloat64 * 1e4)
	params := types.ReqWalletSendToAddress{From: from, To: to, Amount: amountInt64 * 1e4, Note: note}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyHash
	err = rpc.Call("Chain33.SendToAddress", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func ImportPrivKey(privkey string, label string) {
	params := types.ReqWalletImportPrivKey{Privkey: privkey, Label: label}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.WalletAccount
	err = rpc.Call("Chain33.ImportPrivkey", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	balanceResult := strconv.FormatFloat(float64(res.GetAcc().GetBalance())/float64(1e8), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(res.GetAcc().GetFrozen())/float64(1e8), 'f', 4, 64)
	accResult := &AccountResult{
		Addr:     res.GetAcc().GetAddr(),
		Currency: res.GetAcc().GetCurrency(),
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	result := WalletResult{Acc: accResult, Label: res.GetLabel()}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func WalletTransactionList(fromTx string, count string, direction string) {
	countInt32, err := strconv.ParseInt(count, 10, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	directionInt32, err := strconv.ParseInt(direction, 10, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := jsonrpc.ReqWalletTransactionList{
		FromTx:    fromTx,
		Count:     int32(countInt32),
		Direction: int32(directionInt32),
	}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.WalletTxDetails
	err = rpc.Call("Chain33.WalletTxList", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var result WalletTxDetailsResult
	for _, v := range res.TxDetails {
		feeResult := strconv.FormatFloat(float64(v.Tx.Fee)/float64(1e8), 'f', 4, 64)
		t := &TxResult{
			Execer:    v.Tx.Execer,
			Payload:   v.Tx.Payload,
			Signature: v.Tx.Signature,
			Expire:    v.Tx.Expire,
			Nonce:     v.Tx.Nonce,
			To:        v.Tx.To,
			Fee:       feeResult,
		}
		amountResult := strconv.FormatFloat(float64(v.Amount)/float64(1e8), 'f', 4, 64)
		wtxd := &WalletTxDetailResult{
			Tx:        t,
			Receipt:   v.Receipt,
			Height:    v.Height,
			Index:     v.Index,
			Blocktime: v.Blocktime,
			Amount:    amountResult,
			Fromaddr:  v.Fromaddr,
			Txhash:    v.Txhash,
		}
		result.TxDetails = append(result.TxDetails, wtxd)
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetMemPool() {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyTxList
	err = rpc.Call("Chain33.GetMempool", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var result TxListResult
	for _, v := range res.Txs {
		feeResult := strconv.FormatFloat(float64(v.Fee)/float64(1e8), 'f', 4, 64)
		t := &TxResult{
			Execer:    v.Execer,
			Payload:   v.Payload,
			Signature: v.Signature,
			Expire:    v.Expire,
			Nonce:     v.Nonce,
			To:        v.To,
			Fee:       feeResult,
		}
		result.Txs = append(result.Txs, t)
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func SendTransaction(tran string) {
	params := jsonrpc.RawParm{Data: tran}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res string
	err = rpc.Call("Chain33.SendTransaction", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func QueryTransaction(h string) {
	params := jsonrpc.QueryParm{Hash: h}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.TransactionDetail
	err = rpc.Call("Chain33.QueryTransaction", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	feeResult := strconv.FormatFloat(float64(res.Tx.Fee)/float64(1e8), 'f', 4, 64)
	t := &TxResult{
		Execer:    res.Tx.Execer,
		Payload:   res.Tx.Payload,
		Signature: res.Tx.Signature,
		Expire:    res.Tx.Expire,
		Nonce:     res.Tx.Nonce,
		To:        res.Tx.To,
		Fee:       feeResult,
	}
	amountResult := strconv.FormatFloat(float64(res.Amount)/float64(1e8), 'f', 4, 64)
	result := TxDetailResult{
		Tx:        t,
		Receipt:   res.Receipt,
		Proofs:    res.Proofs,
		Height:    res.Height,
		Index:     res.Index,
		Blocktime: res.Blocktime,
		Amount:    amountResult,
		Fromaddr:  res.Fromaddr,
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetTransactionByAddr(addr string, flag string, count string, direction string, height string, index string) {
	flagInt32, err := strconv.ParseInt(flag, 10, 32)
	countInt32, err := strconv.ParseInt(count, 10, 32)
	directionInt32, err := strconv.ParseInt(direction, 10, 32)
	heightInt64, err := strconv.ParseInt(height, 10, 64)
	indexInt64, err := strconv.ParseInt(index, 10, 64)
	params := types.ReqAddr{
		Addr:      addr,
		Flag:      int32(flagInt32),
		Count:     int32(countInt32),
		Direction: int32(directionInt32),
		Height:    heightInt64,
		Index:     indexInt64,
	}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyTxInfos
	err = rpc.Call("Chain33.GetTxByAddr", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetTransactionByHashes(hashes []string) {
	params := jsonrpc.ReqHashes{Hashes: hashes}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.TransactionDetails
	err = rpc.Call("Chain33.GetTxByHashes", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var result TxDetailsResult
	for _, v := range res.Txs {
		feeResult := strconv.FormatFloat(float64(v.Tx.Fee)/float64(1e8), 'f', 4, 64)
		t := &TxResult{
			Execer:    v.Tx.Execer,
			Payload:   v.Tx.Payload,
			Signature: v.Tx.Signature,
			Expire:    v.Tx.Expire,
			Nonce:     v.Tx.Nonce,
			To:        v.Tx.To,
			Fee:       feeResult,
		}
		amountResult := strconv.FormatFloat(float64(v.Amount)/float64(1e8), 'f', 4, 64)
		td := &TxDetailResult{
			Tx:        t,
			Receipt:   v.Receipt,
			Proofs:    v.Proofs,
			Height:    v.Height,
			Index:     v.Index,
			Blocktime: v.Blocktime,
			Amount:    amountResult,
			Fromaddr:  v.Fromaddr,
		}
		result.Txs = append(result.Txs, td)
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetBlocks(start string, end string, detail string) {
	startInt64, err := strconv.ParseInt(start, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	endInt64, err := strconv.ParseInt(end, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	detailBool, err := strconv.ParseBool(detail)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := jsonrpc.BlockParam{Start: startInt64, End: endInt64, Isdetail: detailBool}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.BlockDetails
	err = rpc.Call("Chain33.GetBlocks", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var result BlockDetailsResult
	for _, vItem := range res.Items {
		b := &BlockResult{
			Version:    vItem.Block.Version,
			ParentHash: vItem.Block.ParentHash,
			TxHash:     vItem.Block.TxHash,
			StateHash:  vItem.Block.StateHash,
			Height:     vItem.Block.Height,
			BlockTime:  vItem.Block.BlockTime,
		}
		for _, vTx := range vItem.Block.Txs {
			feeResult := strconv.FormatFloat(float64(vTx.Fee)/float64(1e8), 'f', 4, 64)
			t := &TxResult{
				Execer:    vTx.Execer,
				Payload:   vTx.Payload,
				Signature: vTx.Signature,
				Expire:    vTx.Expire,
				Nonce:     vTx.Nonce,
				To:        vTx.To,
				Fee:       feeResult,
			}
			b.Txs = append(b.Txs, t)
		}
		bd := &BlockDetailResult{Block: b, Receipts: vItem.Receipts}
		result.Items = append(result.Items, bd)
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetLastHeader() {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.Header
	err = rpc.Call("Chain33.GetLastHeader", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetHeaders(start string, end string, detail string) {
	startInt64, err := strconv.ParseInt(start, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	endInt64, err := strconv.ParseInt(end, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	detailBool, err := strconv.ParseBool(detail)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.ReqBlocks{Start: startInt64, End: endInt64, Isdetail: detailBool}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.Headers
	err = rpc.Call("Chain33.GetHeaders", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetPeerInfo() {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.PeerList
	err = rpc.Call("Chain33.GetPeerInfo", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetLastMempool() {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyTxList
	err = rpc.Call("Chain33.GetLastMemPool", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var result TxListResult
	for _, v := range res.Txs {
		feeResult := strconv.FormatFloat(float64(v.Fee)/float64(1e8), 'f', 4, 64)
		t := &TxResult{
			Execer:    v.Execer,
			Payload:   v.Payload,
			Signature: v.Signature,
			Expire:    v.Expire,
			Nonce:     v.Nonce,
			To:        v.To,
			Fee:       feeResult,
		}
		result.Txs = append(result.Txs, t)
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetBlockOverview(hash string) {
	params := jsonrpc.QueryParm{Hash: hash}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.BlockOverview
	err = rpc.Call("Chain33.GetBlockOverview", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetAddrOverview(addr string) {
	params := types.ReqAddr{Addr: addr}

	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.AddrOverview
	err = rpc.Call("Chain33.GetAddrOverview", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	Balance := strconv.FormatFloat(float64(res.GetBalance())/float64(1e8), 'f', 4, 64)
	Reciver := strconv.FormatFloat(float64(res.GetReciver())/float64(1e8), 'f', 4, 64)

	addrOverview := &AddrOverviewResult{
		Balance: Balance,
		Reciver: Reciver,
		TxCount: res.GetTxCount(),
	}

	data, err := json.MarshalIndent(addrOverview, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetBlockHash(height string) {

	heightInt64, err := strconv.ParseInt(height, 10, 64)
	params := types.ReqInt{Height: heightInt64}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyHash
	err = rpc.Call("Chain33.GetBlockHash", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

//seed
func GenSeed(lang string) {

	langInt32, err := strconv.ParseInt(lang, 10, 32)

	params := types.GenSeedLang{Lang: int32(langInt32)}

	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplySeed
	err = rpc.Call("Chain33.GenSeed", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(string(data))
}

func SaveSeed(seed string, password string) {
	params := types.SaveSeedByPw{Seed: seed, Passwd: password}

	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.Reply
	err = rpc.Call("Chain33.SaveSeed", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(string(data))
}

func GetSeed(passwd string) {
	params := types.GetSeedByPw{Passwd: passwd}

	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplySeed
	err = rpc.Call("Chain33.GetSeed", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(string(data))
}

func GetWalletStatus() {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.Reply
	err = rpc.Call("Chain33.GetWalletStatus", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetBalance(address string, execer string) {
	var addrs []string
	addrs = append(addrs, address)
	params := types.ReqBalance{Addresses: addrs, Execer: execer}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res []*jsonrpc.Account
	err = rpc.Call("Chain33.GetBalance", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	balanceResult := strconv.FormatFloat(float64(res[0].Balance)/float64(1e8), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(res[0].Frozen)/float64(1e8), 'f', 4, 64)
	result := &AccountResult{
		Addr:     res[0].Addr,
		Currency: res[0].Currency,
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetExecAddr(exec string) {
	addrResult := account.ExecAddress(exec)
	result := addrResult.String()
	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func BindMiner(mineraddr string, priv string) {
	c, _ := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	a, _ := common.FromHex(priv)
	privKey, _ := c.PrivKeyFromBytes(a)
	originaddr := account.PubKeyToAddress(privKey.PubKey().Bytes()).String()
	ta := &types.TicketAction{}
	tbind := &types.TicketBind{MinerAddress: mineraddr, ReturnAddress: originaddr}
	ta.Value = &types.TicketAction_Tbind{tbind}
	ta.Ty = types.TicketActionBind
	execer := []byte("ticket")
	to := account.ExecAddress(string(execer)).String()
	tx := &types.Transaction{Execer: execer, Payload: types.Encode(ta), Fee: 1e6, To: to}
	var random *rand.Rand
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	var err error
	tx.Fee, err = tx.GetRealFee()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	tx.Fee += types.MinFee
	tx.Sign(types.SECP256K1, privKey)
	txHex := types.Encode(tx)
	fmt.Println(hex.EncodeToString(txHex))
	//	SendTransaction(hex.EncodeToString(txHex))
}
