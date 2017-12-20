package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"code.aliyun.com/chain33/chain33/common"
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
		QueryTransaction(argsWithoutProg[1])
	case "gettxbyaddr": //根据地址获取交易
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetTransactionByAddr(argsWithoutProg[1])
	case "gettxbyhashes": //根据哈希数组获取交易
		if len(argsWithoutProg) < 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
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
	default:
		fmt.Print("指令错误")
	}
}

func LoadHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("lock []                                : 锁定")
	fmt.Println("unlock [password, timeout]             : 解锁")
	fmt.Println("setpasswd [oldpassword, newpassword]   : 设置密码")
	fmt.Println("setlabl [address, label]               : 设置标签")
	fmt.Println("newaccount [labelname]                 : 新建账户")
	fmt.Println("getaccounts []                         : 获取账户列表")
	fmt.Println("mergebalance [to]                      : 合并余额")
	fmt.Println("settxfee [amount]                      : 设置交易费")
	fmt.Println("sendtoaddress [from, to, amount, note] : 发送交易到地址")
	fmt.Println("importprivkey [privkey, label]         : 引入私钥")
	fmt.Println("wallettxlist [from, count, direction]  : 钱包交易列表")
	fmt.Println("getmempool []                          : 获取内存池")
	fmt.Println("sendtransaction [data]                 : 发送交易")
	fmt.Println("querytransaction [hash]                : 按哈希查询交易")
	fmt.Println("gettxbyaddr [address]                  : 按地址获取交易")
	fmt.Println("gettxbyhashes [hashes...]              : 按哈希列表获取交易")
	fmt.Println("getblocks [start, end, isdetail]       : 获取指定区间区块")
	fmt.Println("getlastheader []                       : 获取最新区块头")
	fmt.Println("getheaders [start, end, isdetail]      : 获取指定区间区块头")
	fmt.Println("getpeerinfo []                         : 获取远程节点信息")
}

type AccountsResult struct {
	Wallets []WalletResult
}

type WalletResult struct {
	Acc   AccountResult
	Label string
}

type AccountResult struct {
	Currency int32
	Balance  string
	Frozen   string
	Addr     string
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
	accResult := AccountResult{Addr: res.GetAcc().GetAddr(), Currency: res.GetAcc().GetCurrency(), Balance: balanceResult, Frozen: frozenResult}
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
	accResult := AccountResult{Addr: res.GetAcc().GetAddr(), Currency: res.GetAcc().GetCurrency(), Balance: balanceResult, Frozen: frozenResult}
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
		accResult := AccountResult{Currency: r.Acc.Currency, Addr: r.Acc.Addr, Balance: balanceResult, Frozen: frozenResult}
		result.Wallets = append(result.Wallets, WalletResult{Acc: accResult, Label: r.Label})
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
	amountInt64, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.ReqWalletSetFee{Amount: amountInt64 * 1e8}
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
	amountInt64, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.ReqWalletSendToAddress{From: from, To: to, Amount: amountInt64 * 1e8, Note: note}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyHash
	err = rpc.Call("Chain33.SendToAddress", params, &res)
	if err != nil {
		fmt.Println(err.Error())
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
	accResult := AccountResult{Addr: res.GetAcc().GetAddr(), Currency: res.GetAcc().GetCurrency(), Balance: balanceResult, Frozen: frozenResult}
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
	params := jsonrpc.ReqWalletTransactionList{FromTx: fromTx, Count: int32(countInt32), Direction: int32(directionInt32)}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.TransactionDetails
	err = rpc.Call("Chain33.WalletTxList", params, &res)
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

	data, err := json.MarshalIndent(res, "", "    ")
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

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetTransactionByAddr(addr string) {
	params := jsonrpc.ReqAddr{Addr: addr}
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

	data, err := json.MarshalIndent(res, "", "    ")
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

	data, err := json.MarshalIndent(res, "", "    ")
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
