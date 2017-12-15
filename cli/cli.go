package main

import (
	//	"bytes"
	"errors"
	"fmt"
	//	"io/ioutil"
	//	"net/http"
	"net/rpc/jsonrpc"
	"os"
	"strconv"

	pram "code.aliyun.com/chain33/chain33/rpc"
	"code.aliyun.com/chain33/chain33/types"
)

func main() {

	//	argsWithProg := os.Args
	argsWithoutProg := os.Args[1:]
	switch argsWithoutProg[0] {
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
	//case "Status":
	//	Status() //获取是否为锁定状态
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
	case "wallettransactionlist": //钱包交易列表
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
	case "gettransactionbyaddr": //根据地址获取交易
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetTransactionByAddr(argsWithoutProg[1])
	case "gettransactionbyhashes": //根据哈希数组获取交易
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
	case "getlastheader": //获取上一去块头
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetLastHeader()
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
	fmt.Println("lock []                                       : 锁定")
	fmt.Println("unlock [password]                             : 解锁")
	fmt.Println("setpasswd [oldpassword, newpassword]          : 设置密码")
	fmt.Println("setlabl [address, label]                      : 设置标签")
	fmt.Println("newaccount [labelname]                        : 新建账户")
	fmt.Println("getaccounts []                                : 获取账户列表")
	fmt.Println("mergebalance [to]                             : 合并余额")
	fmt.Println("settxfee [amount]                             : 设置交易费")
	fmt.Println("sendtoaddress [from, to, amount, note]        : 发送到地址")
	fmt.Println("importprivkey [privkey, label]                : 引入私钥")
	fmt.Println("wallettransactionlist [from, count, direction]: 钱包交易列表")
	fmt.Println("getmempool []                                 : 获取内存池")
	fmt.Println("sendtransaction [data]                        : 发送交易")
	fmt.Println("querytransaction [hash]                       : 查询交易")
	fmt.Println("gettransactionbyaddr [address]                : 按地址获取交易")
	fmt.Println("gettransactionbyhashes [hashes...]            : 按哈希列表获取交易")
	fmt.Println("getblocks [start, end, isdetail]              : 获取区块")
	fmt.Println("getlastheader []                              : 获取上一区块头")
	fmt.Println("getpeerinfo []                                : 获取对等点信息")
}

func Lock() {
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *types.Reply
	err = rpc.Call("Chain33.Lock", nil, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.Lock","params":[]}`)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func UnLock(passwd string, timeout string) {
	timeoutInt64, err := strconv.ParseInt(timeout, 10, 64)
	if err != nil {
		panic(err)
		return
	}
	prams := types.WalletUnLock{Passwd: passwd, Timeout: timeoutInt64}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *types.Reply
	err = rpc.Call("Chain33.UnLock", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	timeoutInt64, err := strconv.ParseInt(timeout, 10, 64)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.UnLock",
	//		"params":[{"passwd":"%s","timeout":%d}]}`, passwd, timeoutInt64)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func SetPasswd(oldpass string, newpass string) {
	prams := types.ReqWalletSetPasswd{Oldpass: oldpass, Newpass: newpass}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *types.Reply
	err = rpc.Call("Chain33.SetPasswd", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.SetPasswd",
	//		"params":[{"oldpass":"%s","newpass":"%s"}]}`, oldpass, newpass)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func SetLabl(addr string, label string) {
	prams := types.ReqWalletSetLabel{Addr: addr, Label: label}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *types.WalletAccount
	err = rpc.Call("Chain33.SetLabl", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.SetLabl",
	//		"params":[{"addr":"%s","label":"%s"}]}`, addr, label)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func NewAccount(lb string) {
	prams := types.ReqNewAccount{Label: lb}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *types.WalletAccount
	err = rpc.Call("Chain33.NewAccount", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.NewAccount",
	//		"params":[{"label":"%s"}]}`, lb)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func GetAccounts() {
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *types.WalletAccounts
	err = rpc.Call("Chain33.GetAccounts", nil, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.GetAccounts","params":[]}`)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func MergeBalance(to string) {
	prams := types.ReqWalletMergeBalance{To: to}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *types.ReplyHashes
	err = rpc.Call("Chain33.MergeBalance", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.MergeBalance",
	//		"params":[{"to":"%s"}]}`, to)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func SetTxFee(amount string) {
	amountInt64, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		panic(err)
		return
	}
	prams := types.ReqWalletSetFee{Amount: amountInt64}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *types.Reply
	err = rpc.Call("Chain33.SetTxFee", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.SetTxFee",
	//		"params":[{"amount":%d}]}`, amountInt64)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func SendToAddress(from string, to string, amount string, note string) {
	amountInt64, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		panic(err)
		return
	}
	prams := types.ReqWalletSendToAddress{From: from, To: to, Amount: amountInt64, Note: note}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *types.ReplyHash
	err = rpc.Call("Chain33.SendToAddress", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.SendToAddress",
	//		"params":[{"from":"%s","to":"%s","amount":%d,"note":"%s"}]}`, from, to, amountInt64, note)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func ImportPrivKey(privkey string, label string) {
	prams := types.ReqWalletImportPrivKey{Privkey: privkey, Label: label}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *types.WalletAccount
	err = rpc.Call("Chain33.ImportPrivKey", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.ImportPrivKey",
	//		"params":[{"privkey":"%s","label":"%s"}]}`, privkey, label)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func WalletTransactionList(fromTx string, count string, direction string) {
	countInt32, err := strconv.ParseInt(count, 10, 32)
	if err != nil {
		panic(err)
		return
	}
	directionInt32, err := strconv.ParseInt(direction, 10, 32)
	if err != nil {
		panic(err)
		return
	}
	prams := pram.ReqWalletTransactionList{FromTx: fromTx, Count: int32(countInt32), Direction: int32(directionInt32)}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *pram.TransactionDetails
	err = rpc.Call("Chain33.WalletTxList", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	countInt32, err := strconv.ParseInt(count, 10, 32)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	directionInt32, err := strconv.ParseInt(direction, 10, 32)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.WalletTxList",
	//		"params":[{"fromTx":"%s","count":%d,"direction":&d}]}`, fromTx, countInt32, directionInt32)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func GetMemPool() {
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *pram.ReplyTxList
	err = rpc.Call("Chain33.GetMemPool", nil, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.GetMempool","params":[]}`)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func SendTransaction(data string) {
	prams := pram.RawParm{Data: data}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *string
	err = rpc.Call("Chain33.SendTransaction", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.SendTransaction",
	//		"params":[{"data":"%s"}]}`, data)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func QueryTransaction(h string) {
	prams := pram.QueryParm{Hash: h}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *pram.TransactionDetail
	err = rpc.Call("Chain33.QueryTransaction", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.QueryTransaction",
	//		"params":[{"hash":"%s"}]}`, hash)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func GetTransactionByAddr(addr string) {
	prams := pram.ReqAddr{Addr: addr}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *pram.ReplyTxInfos
	err = rpc.Call("Chain33.GetTxByAddr", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.GetTxByAddr",
	//		"params":[{"addr":"%s"}]}`, addr)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func GetTransactionByHashes(hashes []string) {
	prams := pram.ReqHashes{Hashes: hashes}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *pram.TransactionDetails
	err = rpc.Call("Chain33.GetTxByHashes", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	var buf bytes.Buffer
	//	buf.WriteString("[")
	//	for h := range hashes {
	//		buf.WriteString(hashes[h] + ",")
	//	}
	//	strHashes := buf.String()
	//	length := len(strHashes)
	//	if length > 1 {
	//		strHashes = strHashes[:length-1]
	//	}
	//	strHashes += "]"
	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.GetTxByHashes",
	//		"params":[{"addr":"%s"}]}`, strHashes)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func GetBlocks(start string, end string, detail string) {
	startInt64, err := strconv.ParseInt(start, 10, 64)
	if err != nil {
		panic(err)
		return
	}
	endInt64, err := strconv.ParseInt(end, 10, 64)
	if err != nil {
		panic(err)
		return
	}
	detailBool, err := strconv.ParseBool(detail)
	if err != nil {
		panic(err)
		return
	}
	prams := pram.BlockParam{Start: startInt64, End: endInt64, Isdetail: detailBool}
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *pram.BlockDetails
	err = rpc.Call("Chain33.GetBlocks", prams, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	startInt64, err := strconv.ParseInt(start, 10, 64)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	endInt64, err := strconv.ParseInt(end, 10, 64)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	detailBool, err := strconv.ParseBool(detail)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.GetBlocks",
	//		"params":[{"start":%d,"end":%d,"isdetail":%t}]}`, startInt64, endInt64, detailBool)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func GetLastHeader() {
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *pram.Header
	err = rpc.Call("Chain33.GetLastHeader", nil, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.GetLastHeader","params":[]}`)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}

func GetPeerInfo() {
	rpc, err := jsonrpc.Dial("jsonrpc", "localhost:8801")
	if err != nil {
		panic(err)
		return
	}
	var res *pram.PeerList
	err = rpc.Call("Chain33.GetPeerInfo", nil, res)
	if err != nil {
		panic(err)
	}

	fmt.Print(res)

	//	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":0,"method":"Chain33.GetPeerInfo","params":[]}`)
	//	resp, err := http.Post("http://localhost:8801", "application/json", bytes.NewBufferString(poststr))
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	defer resp.Body.Close()
	//	b, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		panic(err)
	//		return
	//	}
	//	fmt.Print(string(b))
}
