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

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/common/version"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func main() {
	clog.SetLogLevel("eror")
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
	case "version", "-v", "--version":
		GetVersion()
	case "lock": //锁定
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		Lock()
	case "unlock": //解锁
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		UnLock(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
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
		SendToAddress(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4], false, "", false)
	case "withdrawfromexec":
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		execaddr := GetExecAddr(argsWithoutProg[2], false)
		if execaddr == "" {
			return
		}
		SendToAddress(argsWithoutProg[1], execaddr, argsWithoutProg[3], argsWithoutProg[4], false, "", true)
	case "transfertoken":
		if len(argsWithoutProg) != 6 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SendToAddress(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[5], true, argsWithoutProg[4], false)
	case "withdrawtoken":
		if len(argsWithoutProg) != 6 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		execaddr := GetExecAddr(argsWithoutProg[2], false)
		if execaddr == "" {
			return
		}
		SendToAddress(argsWithoutProg[1], execaddr, argsWithoutProg[3], argsWithoutProg[5], true, argsWithoutProg[4], true)
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
		GetWalletStatus(false)
	case "getbalance":
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetBalance(argsWithoutProg[1], argsWithoutProg[2])
	case "gettokenbalance":
		argsCnt := len(argsWithoutProg)
		if argsCnt < 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		var addresses []string
		for _, args := range argsWithoutProg[3:] {
			addresses = append(addresses, args)
		}

		GetTokenBalance(addresses, argsWithoutProg[1], argsWithoutProg[2])
	case "gettokenassets":
		if len(argsWithoutProg) < 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		exec := "token"
		if len(argsWithoutProg) == 3 {
			if argsWithoutProg[2] == "token" || argsWithoutProg[2] == "trade" {
				exec = argsWithoutProg[2]
			}
		}
		GetTokenAssets(argsWithoutProg[1], exec)
	case "getexecaddr":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetExecAddr(argsWithoutProg[1], true)
	case "bindminer":
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		BindMiner(argsWithoutProg[1], argsWithoutProg[2])
	case "setautomining":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SetAutoMining(argsWithoutProg[1])
	case "getrawtx":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetTxHexByHash(argsWithoutProg[1])
	case "getticketcount":
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetTicketCount()
	case "dumpprivkey": //导出私钥
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		DumpPrivkey(argsWithoutProg[1])
	case "decodetx":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		DecodeTx(argsWithoutProg[1])
	case "getcoldaddrbyminer":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetColdAddrByMiner(argsWithoutProg[1])
	case "closetickets":
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		CloseTickets()
	case "createrawsendtx":
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		CreateRawSendTx(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4], false, false, "")
	case "createrawwithdrawtx":
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		execaddr := GetExecAddr(argsWithoutProg[2], false)
		if execaddr == "" {
			return
		}
		CreateRawSendTx(argsWithoutProg[1], execaddr, argsWithoutProg[3], argsWithoutProg[4], true, false, "")
	case "createrawtokentransfer":
		if len(argsWithoutProg) != 6 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		CreateRawSendTx(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4], false, true, argsWithoutProg[5])
	case "createrawtokenwithdraw":
		if len(argsWithoutProg) != 6 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		execaddr := GetExecAddr(argsWithoutProg[2], false)
		if execaddr == "" {
			return
		}
		CreateRawSendTx(argsWithoutProg[1], execaddr, argsWithoutProg[3], argsWithoutProg[4], true, true, argsWithoutProg[5])
	case "issync":
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		IsSync()
	case "gettokensprecreated":
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetTokensPrecreated()
	case "gettokensfinishcreated":
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetTokensFinishCreated()
	case "gettokeninfo":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetTokenInfo(argsWithoutProg[1])
	case "precreatetoken":
		if len(argsWithoutProg) != 8 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		PreCreateToken(argsWithoutProg[1:])
	case "finishcreatetoken":
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		FinishCreateToken(argsWithoutProg[1:])
	case "selltoken":
		if len(argsWithoutProg) != 7 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}

		SellToken(argsWithoutProg[1:], "0", "0", false)
	case "selltokencrowdfund":
		if len(argsWithoutProg) != 9 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SellToken(argsWithoutProg[1:6], argsWithoutProg[6], argsWithoutProg[7], true)
	case "buytoken":
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		BuyToken(argsWithoutProg[1:])
	case "revokeselltoken":
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		RevokeSellToken(argsWithoutProg[1], argsWithoutProg[2])
	case "showonesselltokenorder":
		if len(argsWithoutProg) < 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		var tokens []string
		if len(argsWithoutProg) > 2 {
			tokens = append(tokens, argsWithoutProg[2:]...)
		}
		ShowOnesSellTokenOrders(argsWithoutProg[1], tokens)
	case "showsellorderwithstatus":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		ShowSellOrderWithStatus(argsWithoutProg[1])
	case "showonesbuyorder":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		var tokens []string
		ShowOnesBuyOrder(argsWithoutProg[1], tokens)
	case "showonesbuytokenorder":
		if len(argsWithoutProg) < 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		ShowOnesBuyOrder(argsWithoutProg[1], argsWithoutProg[2:])
	case "showtokensellorder":
		if len(argsWithoutProg) != 4 && len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		ShowTokenSellOrder(argsWithoutProg[1:])
	case "revokecreatetoken":
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		RevokeCreateToken(argsWithoutProg[1:])
	case "configtransaction":
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		ManageConfigTransactioin(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	case "queryconfig": //查询配置
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		QueryConfigItem(argsWithoutProg[1])
	case "isntpclocksync":
		if len(argsWithoutProg) != 1 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		IsNtpClockSync()
	case "gettotalcoins": //查询代币总数
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		GetTotalCoins(argsWithoutProg[1], argsWithoutProg[2])
	default:
		fmt.Print("指令错误")
	}
}

func LoadHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("lock []                                                     : 锁定")
	fmt.Println("unlock [password, walletorticket, timeout]                  : 解锁钱包或者买票,walletorticket:1,只解锁买票,0：解锁整个钱包")
	fmt.Println("setpasswd [oldpassword, newpassword]                        : 设置密码")
	fmt.Println("setlabl [address, label]                                    : 设置标签")
	fmt.Println("newaccount [labelname]                                      : 新建账户")
	fmt.Println("getaccounts []                                              : 获取账户列表")
	fmt.Println("mergebalance [to]                                           : 合并余额")
	fmt.Println("settxfee [amount]                                           : 设置交易费")
	fmt.Println("sendtoaddress [from, to, amount, note]                      : 发送交易到地址")
	fmt.Println("withdrawfromexec [addr, exec, amount, note]                 : 从合约地址提币")
	fmt.Println("transfertoken [from, to, amount, tokensymbol, note]         : 转账token")
	fmt.Println("withdrawtoken [addr, token, amount, symbol, note]           : 提取token")
	fmt.Println("createrawsendtx [privkey, to, amount, note]                 : 创建交易")
	fmt.Println("createrawwithdrawtx [privkey, exec, amount, note]           : 创建提币交易")
	fmt.Println("createtokentransfer [privkey, to, amount, note]             : 创建token转账交易")
	fmt.Println("createtokenwithdraw [privkey, token, amount, note, symbol]  : 创建token提币交易")
	fmt.Println("importprivkey [privkey, label]                              : 引入私钥")
	fmt.Println("dumpprivkey [addr]                                          : 导出私钥")
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
	fmt.Println("gettokenbalance [token execer addr0 [addr1 addr2]]          : 查询多个地址在token的余额")
	fmt.Println("gettokenassets [addr execer]                                : 查询地址下的token/trace合约下的token资产")
	fmt.Println("getexecaddr [execer]                                        : 获取执行器地址")
	fmt.Println("bindminer [mineraddr, privkey]                              : 绑定挖矿地址")
	fmt.Println("setautomining [flag]                                        : 设置自动挖矿")
	fmt.Println("getrawtx [hash]                                             : 通过哈希获取交易十六进制字符串")
	fmt.Println("getticketcount []                                           : 获取票数")
	fmt.Println("decodetx [data]                                             : 解析交易")
	fmt.Println("getcoldaddrbyminer [address]                                : 获取miner冷钱包地址")
	fmt.Println("closetickets []                                             : 关闭挖矿票")
	fmt.Println("issync []                                                   : 获取同步状态")
	fmt.Println("precreatetoken [creator_address, name, symbol, introduction, owner_address, total, price]")
	fmt.Println("                                                            : 预创建token")
	fmt.Println("finishcreatetoken [finish_address, symbol, owner_address]   : 完成创建token")
	fmt.Println("revokecreatetoken [creator_address, symbol, owner_address]  : 取消创建token")
	fmt.Println("gettokensprecreated                                         : 获取所有预创建的token")
	fmt.Println("gettokensfinishcreated                                      : 获取所有完成创建的token")
	fmt.Println("gettokeninfo                                                : 获取token信息")
	fmt.Println("selltoken [owner, token, Amountpbl, minbl, pricepbl, totalpbl] : 卖出token")
	fmt.Println("buytoken [buyer, sellid, countboardlot]                        : 买入token")
	fmt.Println("revokeselltoken [seller, sellid]                               : 撤销token卖单")
	fmt.Println("showonesselltokenorder [seller, [token0, token1, token2]]      : 显示一个用户下的token卖单")
	fmt.Println("showtokensellorder [token, count, direction, fromSellId]       : 分页显示token的卖单")
	fmt.Println("showsellorderwithstatus [onsale | soldout | revoked]           : 显示指定状态下的所有卖单")
	fmt.Println("showonesbuyorder [buyer]                                       : 显示指定用户下所有token成交的购买单")
	fmt.Println("showonesbuytokenorder [buyer, token0, [token1, token2]]        : 显示指定用户下指定token成交的购买单")
	fmt.Println("sellcrowdfund [owner, token, Amountpbl, minbl, pricepbl, totalpbl, start, stop]              : 卖出众筹")
	fmt.Println("configtransaction [configKey, operate, value, privkey]         : 修改配置")
	fmt.Println("queryconfig [Key]                                              : 查询配置")
	fmt.Println("isntpclocksync []                                           : 获取网络时间同步状态")
	fmt.Println("gettotalcoins [symbol, height]                              : 查询代币总数")

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

type TokenAccountResult struct {
	Token    string `json:"Token,omitempty"`
	Currency int32  `json:"currency,omitempty"`
	Balance  string `json:"balance,omitempty"`
	Frozen   string `json:"frozen,omitempty"`
	Addr     string `json:"addr,omitempty"`
}

type TxListResult struct {
	Txs []*TxResult `json:"txs"`
}

type TxResult struct {
	Execer     string             `json:"execer"`
	Payload    interface{}        `json:"payload"`
	RawPayload string             `json:"rawpayload"`
	Signature  *jsonrpc.Signature `json:"signature"`
	Fee        string             `json:"fee"`
	Expire     int64              `json:"expire"`
	Nonce      int64              `json:"nonce"`
	To         string             `json:"to"`
	Amount     string             `json:"amount,omitempty"`
	From       string             `json:"from,omitempty"`
}

type ReceiptData struct {
	Ty     int32         `json:"ty"`
	TyName string        `json:"tyname"`
	Logs   []*ReceiptLog `json:"logs"`
}

type ReceiptLog struct {
	Ty     int32       `json:"ty"`
	TyName string      `json:"tyname"`
	Log    interface{} `json:"log"`
	RawLog string      `json:"rawlog"`
}

type ReceiptAccountTransfer struct {
	Prev    *AccountResult `protobuf:"bytes,1,opt,name=prev" json:"prev,omitempty"`
	Current *AccountResult `protobuf:"bytes,2,opt,name=current" json:"current,omitempty"`
}

type ReceiptExecAccountTransfer struct {
	ExecAddr string         `protobuf:"bytes,1,opt,name=execAddr" json:"execAddr,omitempty"`
	Prev     *AccountResult `protobuf:"bytes,2,opt,name=prev" json:"prev,omitempty"`
	Current  *AccountResult `protobuf:"bytes,3,opt,name=current" json:"current,omitempty"`
}

type TxDetailResult struct {
	Tx         *TxResult    `json:"tx"`
	Receipt    *ReceiptData `json:"receipt"`
	Proofs     []string     `json:"proofs,omitempty"`
	Height     int64        `json:"height"`
	Index      int64        `json:"index"`
	Blocktime  int64        `json:"blocktime"`
	Amount     string       `json:"amount"`
	Fromaddr   string       `json:"fromaddr"`
	ActionName string       `json:"actionname"`
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
	Block    *BlockResult   `json:"block"`
	Receipts []*ReceiptData `json:"receipts"`
}

type BlockDetailsResult struct {
	Items []*BlockDetailResult `json:"items"`
}

type WalletTxDetailsResult struct {
	TxDetails []*WalletTxDetailResult `json:"txDetails"`
}

type WalletTxDetailResult struct {
	Tx         *TxResult    `json:"tx"`
	Receipt    *ReceiptData `json:"receipt"`
	Height     int64        `json:"height"`
	Index      int64        `json:"index"`
	Blocktime  int64        `json:"blocktime"`
	Amount     string       `json:"amount"`
	Fromaddr   string       `json:"fromaddr"`
	Txhash     string       `json:"txhash"`
	ActionName string       `json:"actionname"`
}

type AddrOverviewResult struct {
	Reciver string `json:"reciver"`
	Balance string `json:"balance"`
	TxCount int64  `json:"txCount"`
}

type SellOrder2Show struct {
	Tokensymbol       string `json:"tokensymbol"`
	Seller            string `json:"address"`
	Amountperboardlot string `json:"amountperboardlot"`
	Minboardlot       int64  `json:"minboardlot"`
	Priceperboardlot  string `json:"priceperboardlot"`
	Totalboardlot     int64  `json:"totalboardlot"`
	Soldboardlot      int64  `json:"soldboardlot"`
	Starttime         int64  `json:"starttime"`
	Stoptime          int64  `json:"stoptime"`
	Crowdfund         bool   `json:"crowdfund"`
	SellID            string `json:"sellid"`
	Status            string `json:"status"`
	Height            int64  `json:"height"`
}

type GetTotalCoinsResult struct {
	TxCount          int64  `json:"txCount"`
	AccountCount     int64  `json:"accountCount"`
	ExpectedAmount   string `json:"expectedAmount"`
	ActualAmount     string `json:"actualAmount"`
	DifferenceAmount string `json:"differenceAmount"`
}

func GetVersion() {
	fmt.Println(version.GetVersion())
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

func UnLock(passwd string, walletOrTicket string, timeout string) {
	timeoutInt64, err := strconv.ParseInt(timeout, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	walletOrTicketBool, err := strconv.ParseBool(walletOrTicket)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	params := types.WalletUnLock{Passwd: passwd, WalletOrTicket: walletOrTicketBool, Timeout: timeoutInt64}
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

	accResult := decodeAccount(res.GetAcc(), types.Coin)
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

	accResult := decodeAccount(res.GetAcc(), types.Coin)
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
		balanceResult := strconv.FormatFloat(float64(r.Acc.Balance)/float64(types.Coin), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(r.Acc.Frozen)/float64(types.Coin), 'f', 4, 64)
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

func SendToAddress(from string, to string, amount string, note string, isToken bool, tokenSymbol string, isWithdraw bool) {
	amountFloat64, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64 := int64(amountFloat64*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	if isWithdraw {
		amountInt64 = -amountInt64
	}
	params := types.ReqWalletSendToAddress{From: from, To: to, Amount: amountInt64, Note: note}
	if !isToken {
		params.Istoken = false
	} else {
		params.Istoken = true
		params.TokenSymbol = tokenSymbol
	}
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

	accResult := decodeAccount(res.GetAcc(), types.Coin)
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
		amountResult := strconv.FormatFloat(float64(v.Amount)/float64(types.Coin), 'f', 4, 64)
		wtxd := &WalletTxDetailResult{
			Tx:         decodeTransaction(v.Tx),
			Receipt:    decodeLog(*(v.Receipt)),
			Height:     v.Height,
			Index:      v.Index,
			Blocktime:  v.Blocktime,
			Amount:     amountResult,
			Fromaddr:   v.Fromaddr,
			Txhash:     v.Txhash,
			ActionName: v.ActionName,
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
		result.Txs = append(result.Txs, decodeTransaction(v))
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

	fmt.Println("hash:", res)
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
		amountResult := strconv.FormatFloat(float64(v.Amount)/float64(types.Coin), 'f', 4, 64)
		if err != nil {
			continue
		}
		td := TxDetailResult{
			Tx:         decodeTransaction(v.Tx),
			Receipt:    decodeLog(*(v.Receipt)),
			Proofs:     v.Proofs,
			Height:     v.Height,
			Index:      v.Index,
			Blocktime:  v.Blocktime,
			Amount:     amountResult,
			Fromaddr:   v.Fromaddr,
			ActionName: v.ActionName,
		}
		result.Txs = append(result.Txs, &td)
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
			b.Txs = append(b.Txs, decodeTransaction(vTx))
		}
		var rpt []*ReceiptData
		for _, vR := range vItem.Receipts {
			rpt = append(rpt, decodeLog(*vR))
		}
		bd := &BlockDetailResult{Block: b, Receipts: rpt}
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
		result.Txs = append(result.Txs, decodeTransaction(v))
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

	Balance := strconv.FormatFloat(float64(res.GetBalance())/float64(types.Coin), 'f', 4, 64)
	Reciver := strconv.FormatFloat(float64(res.GetReciver())/float64(types.Coin), 'f', 4, 64)

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

func GetWalletStatus(isCloseTickets bool) (interface{}, error) {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	var res jsonrpc.WalletStatus
	err = rpc.Call("Chain33.GetWalletStatus", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	if !isCloseTickets {
		fmt.Println(string(data))
	}
	return res, nil
}

func GetBalance(address string, execer string) {
	isExecer := false
	for _, e := range [7]string{"none", "coins", "hashlock", "retrieve", "ticket", "token", "trade"} {
		if e == execer {
			isExecer = true
			break
		}
	}
	if !isExecer {
		fmt.Println("only none, coins, hashlock, retrieve, ticket, token, trade supported")
		return
	}
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

	balanceResult := strconv.FormatFloat(float64(res[0].Balance)/float64(types.Coin), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(res[0].Frozen)/float64(types.Coin), 'f', 4, 64)
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

func GetTokenBalance(addresses []string, tokenSymbol string, execer string) {
	//var addrs []string
	//addrs = append(addrs, address)
	params := types.ReqTokenBalance{Addresses: addresses, TokenSymbol: tokenSymbol, Execer: execer}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res []*jsonrpc.Account
	err = rpc.Call("Chain33.GetTokenBalance", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for _, result := range res {
		balanceResult := strconv.FormatFloat(float64(result.Balance)/float64(types.TokenPrecision), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(result.Frozen)/float64(types.TokenPrecision), 'f', 4, 64)
		result := &TokenAccountResult{
			Token:    tokenSymbol,
			Addr:     result.Addr,
			Currency: result.Currency,
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

}

func GetTokenAssets(addr, execer string) {
	req := types.ReqAccountTokenAssets{Address: addr, Execer: execer}
	var params jsonrpc.Query4Cli
	params.Execer = "token"
	params.FuncName = "GetAccountTokenAssets"
	params.Payload = req
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res *types.ReplyAccountTokenAssets
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for _, result := range res.TokenAssets {
		balanceResult := strconv.FormatFloat(float64(result.Account.Balance)/float64(types.TokenPrecision), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(result.Account.Frozen)/float64(types.TokenPrecision), 'f', 4, 64)
		result := &TokenAccountResult{
			Token:    result.Symbol,
			Addr:     result.Account.Addr,
			Currency: result.Account.Currency,
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

}

func GetExecAddr(exec string, needPrint bool) string {
	var addr string
	switch exec {
	case "none", "coins", "hashlock", "retrieve", "ticket", "token", "trade":
		addrResult := account.ExecAddress(exec)
		result := addrResult.String()
		data, err := json.MarshalIndent(result, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return ""
		}
		addr = result
		if needPrint {
			fmt.Println(string(data))
		}
	default:
		fmt.Println("only none, coins, hashlock, retrieve, ticket, token, trade supported")
		addr = ""
	}
	return addr
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
	tx := &types.Transaction{Execer: execer, Payload: types.Encode(ta), To: to}
	var random *rand.Rand
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	var err error
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	tx.Fee += types.MinFee
	tx.Sign(types.SECP256K1, privKey)
	txHex := types.Encode(tx)
	fmt.Println(hex.EncodeToString(txHex))
}

func CreateRawSendTx(priv string, to string, amount string, note string, withdraw bool, isToken bool, tokenSymbol string) {
	amountFloat64, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64 := int64(amountFloat64*1e4) * 1e4

	c, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	a, err := common.FromHex(priv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	privKey, err := c.PrivKeyFromBytes(a)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	// addrFrom := account.PubKeyToAddress(privKey.PubKey().Bytes()).String()
	var tx *types.Transaction
	if !isToken {
		transfer := &types.CoinsAction{}
		if !withdraw {
			v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionTransfer
		} else {
			v := &types.CoinsAction_Withdraw{&types.CoinsWithdraw{Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionWithdraw
		}
		tx = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), To: to}
	} else {
		transfer := &types.TokenAction{}
		if !withdraw {
			v := &types.TokenAction_Transfer{&types.CoinsTransfer{Cointoken: tokenSymbol, Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		} else {
			v := &types.TokenAction_Withdraw{&types.CoinsWithdraw{Cointoken: tokenSymbol, Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.ActionWithdraw
		}
		tx = &types.Transaction{Execer: []byte("token"), Payload: types.Encode(transfer), To: to}
	}

	tx.Fee, err = tx.GetRealFee(types.MinBalanceTransfer)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	tx.Fee += types.MinBalanceTransfer
	var random *rand.Rand
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	tx.Sign(types.SECP256K1, privKey)
	txHex := types.Encode(tx)
	fmt.Println(hex.EncodeToString(txHex))
}

func SetAutoMining(flag string) {
	flagInt32, err := strconv.ParseInt(flag, 10, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.MinerFlag{Flag: int32(flagInt32)}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.Reply
	err = rpc.Call("Chain33.SetAutoMining", params, &res)
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

func GetTxHexByHash(hash string) {
	params := jsonrpc.QueryParm{Hash: hash}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res string
	err = rpc.Call("Chain33.GetHexTxByHash", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println("raw tx:", res)
}

func GetTicketCount() {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res int64
	err = rpc.Call("Chain33.GetTicketCount", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println("ticket count:", res)
}

func DumpPrivkey(addr string) {
	params := types.ReqStr{Reqstr: addr}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplyStr
	err = rpc.Call("Chain33.DumpPrivkey", params, &res)
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

func DecodeTx(tran string) {
	var tx types.Transaction
	txHex, err := common.FromHex(tran)
	err = types.Decode(txHex, &tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	res, err := jsonrpc.DecodeTx(&tx)
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

	amountResult := strconv.FormatFloat(float64(res.Amount)/float64(types.Coin), 'f', 4, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	result := TxDetailResult{
		Tx:         decodeTransaction(res.Tx),
		Receipt:    decodeLog(*(res.Receipt)),
		Proofs:     res.Proofs,
		Height:     res.Height,
		Index:      res.Index,
		Blocktime:  res.Blocktime,
		Amount:     amountResult,
		Fromaddr:   res.Fromaddr,
		ActionName: res.ActionName,
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func GetColdAddrByMiner(addr string) {
	reqaddr := &types.ReqString{addr}
	var params jsonrpc.Query4Cli
	params.Execer = "ticket"
	params.FuncName = "MinerSourceList"
	params.Payload = reqaddr
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.Message
	err = rpc.Call("Chain33.Query", params, &res)
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

func CloseTickets() {
	status, err := GetWalletStatus(true)
	if err != nil {
		return
	}
	isAutoMining := status.(jsonrpc.WalletStatus).IsAutoMining
	if isAutoMining {
		fmt.Fprintln(os.Stderr, types.ErrMinerNotClosed)
		return
	}

	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplyHashes
	err = rpc.Call("Chain33.CloseTickets", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if len(res.Hashes) == 0 {
		fmt.Println("no ticket to be close")
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func IsSync() {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res bool
	err = rpc.Call("Chain33.IsSync", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println("sync completed:", res)
}

func decodeTransaction(tx *jsonrpc.Transaction) *TxResult {
	feeResult := strconv.FormatFloat(float64(tx.Fee)/float64(types.Coin), 'f', 4, 64)
	amountResult := ""
	if tx.Amount != 0 {
		amountResult = strconv.FormatFloat(float64(tx.Amount)/float64(types.Coin), 'f', 4, 64)
	}
	result := &TxResult{
		Execer:     tx.Execer,
		Payload:    tx.Payload,
		RawPayload: tx.RawPayload,
		Signature:  tx.Signature,
		Fee:        feeResult,
		Expire:     tx.Expire,
		Nonce:      tx.Nonce,
		To:         tx.To,
	}
	if plValue, ok := tx.Payload.(map[string]interface{})["Value"]; ok {
		payloadValue := plValue.(map[string]interface{})
		for _, e := range [4]string{"Transfer", "Withdraw", "Genesis", "Hlock"} {
			if _, ok := payloadValue[e]; ok {
				if amtValue, ok := result.Payload.(map[string]interface{})["Value"].(map[string]interface{})[e].(map[string]interface{})["amount"]; ok {
					amt := amtValue.(float64) / float64(types.Coin)
					amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
					result.Payload.(map[string]interface{})["Value"].(map[string]interface{})[e].(map[string]interface{})["amount"] = amtResult
					break
				}
			}
		}
		if _, ok := payloadValue["Miner"]; ok {
			if rwdValue, ok := result.Payload.(map[string]interface{})["Value"].(map[string]interface{})["Miner"].(map[string]interface{})["reward"]; ok {
				rwd := rwdValue.(float64) / float64(types.Coin)
				rwdResult := strconv.FormatFloat(rwd, 'f', 4, 64)
				result.Payload.(map[string]interface{})["Value"].(map[string]interface{})["Miner"].(map[string]interface{})["reward"] = rwdResult
			}
		}
	}
	if tx.Amount != 0 {
		result.Amount = amountResult
	}
	if tx.From != "" {
		result.From = tx.From
	}
	return result
}

func decodeAccount(acc *types.Account, precision int64) *AccountResult {
	balanceResult := strconv.FormatFloat(float64(acc.GetBalance())/float64(precision), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(acc.GetFrozen())/float64(precision), 'f', 4, 64)
	accResult := &AccountResult{
		Addr:     acc.GetAddr(),
		Currency: acc.GetCurrency(),
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	return accResult
}

func constructAccFromLog(l *jsonrpc.ReceiptLogResult, key string) *types.Account {
	var cur int32
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["currency"]; ok {
		cur = int32(tmp.(float32))
	}
	var bal int64
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["balance"]; ok {
		bal = int64(tmp.(float64))
	}
	var fro int64
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["frozen"]; ok {
		fro = int64(tmp.(float64))
	}
	var ad string
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["addr"]; ok {
		ad = tmp.(string)
	}
	return &types.Account{
		Currency: cur,
		Balance:  bal,
		Frozen:   fro,
		Addr:     ad,
	}
}

func GetTokensPrecreated() {
	var reqtokens types.ReqTokens
	reqtokens.Status = types.TokenStatusPreCreated
	reqtokens.Queryall = true
	var params jsonrpc.Query4Cli
	params.Execer = "token"
	params.FuncName = "GetTokens"
	params.Payload = reqtokens
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplyTokens
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for i, preCreatedToken := range res.Tokens {
		preCreatedToken.Price = preCreatedToken.Price / types.Coin
		preCreatedToken.Total = preCreatedToken.Total / types.TokenPrecision

		fmt.Printf("---The %dth precreated token is below--------------------\n", i)
		data, err := json.MarshalIndent(preCreatedToken, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		fmt.Println(string(data))
	}

}

func GetTokensFinishCreated() {
	var reqtokens types.ReqTokens
	reqtokens.Status = types.TokenStatusCreated
	reqtokens.Queryall = true
	var params jsonrpc.Query4Cli
	params.Execer = "token"
	params.FuncName = "GetTokens"
	params.Payload = reqtokens

	fmt.Fprintln(os.Stderr, "Payload is: ", params.Payload)
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplyTokens
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for i, createdToken := range res.Tokens {
		createdToken.Price = createdToken.Price / types.Coin
		createdToken.Total = createdToken.Total / types.TokenPrecision

		fmt.Printf("---The %dth Finish Created token is below--------------------\n", i)
		data, err := json.MarshalIndent(createdToken, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		fmt.Println(string(data))
	}
}

//获取Token信息
func GetTokenInfo(symbol string) {
	var req types.ReqString
	req.Data = symbol
	var params jsonrpc.Query4Cli
	params.Execer = "token"
	params.FuncName = "GetTokenInfo"
	params.Payload = req
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.Token
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	res.Price = res.Price / types.Coin
	res.Total = res.Total / types.TokenPrecision

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func PreCreateToken(args []string) {
	// creator, name, symbol, introduction, owner, totalStr, priceStr string) {
	creator := args[0]
	name := args[1]
	symbol := args[2]
	introduction := args[3]
	owner := args[4]
	total, err := strconv.ParseInt(args[5], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	price, err := strconv.ParseInt(args[6], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.ReqTokenPreCreate{CreatorAddr: creator, Name: name, Symbol: symbol, Introduction: introduction, OwnerAddr: owner, Total: total * types.TokenPrecision, Price: price * types.Coin}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyHash
	err = rpc.Call("Chain33.TokenPreCreate", params, &res)
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

func SellToken(args []string, starttime string, stoptime string, isCrowfund bool) {
	owner := args[0]
	sell := types.TradeForSell{}
	params := &types.ReqSellToken{&sell, owner}

	sell.Tokensymbol = args[1]
	var err error
	amountperboardlot, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64 := int64(amountperboardlot*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	sell.Amountperboardlot = amountInt64

	sell.Minboardlot, err = strconv.ParseInt(args[3], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	price, err := strconv.ParseFloat(args[4], 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	sell.Priceperboardlot = int64(price*types.InputPrecision) * types.Multiple1E4

	sell.Totalboardlot, err = strconv.ParseInt(args[5], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	sell.Starttime, err = strconv.ParseInt(starttime, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	sell.Stoptime, err = strconv.ParseInt(starttime, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	sell.Crowdfund = isCrowfund
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyHash

	err = rpc.Call("Chain33.SellToken", params, &res)
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

func FinishCreateToken(args []string) {
	// finisher, symbol, owner, string) {
	finisher := args[0]
	symbol := args[1]
	owner := args[2]

	params := types.ReqTokenFinishCreate{FinisherAddr: finisher, Symbol: symbol, OwnerAddr: owner}

	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyHash

	err = rpc.Call("Chain33.TokenFinishCreate", params, &res)
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

//buytoken [0-owner, 1-sellid, 2-countboardlot]
func BuyToken(args []string) {
	cntBoardlot, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	buy := &types.TradeForBuy{args[1], cntBoardlot}
	params := &types.ReqBuyToken{buy, args[0]}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyHash
	err = rpc.Call("Chain33.BuyToken", params, &res)
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

func RevokeCreateToken(args []string) {
	// revoker, symbol, owner, string) {
	revoker := args[0]
	symbol := args[1]
	owner := args[2]

	params := types.ReqTokenRevokeCreate{RevokerAddr: revoker, Symbol: symbol, OwnerAddr: owner}
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyHash
	err = rpc.Call("Chain33.TokenRevokeCreate", params, &res)
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

//revokeselltoken [seller, sellid]
func RevokeSellToken(seller string, sellid string) {
	revoke := &types.TradeForRevokeSell{sellid}
	params := &types.ReqRevokeSell{revoke, seller}

	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res jsonrpc.ReplyHash
	err = rpc.Call("Chain33.RevokeSellToken", params, &res)
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

//获取并显示一个指定用户下的所有token的卖单或者是指定token的卖单
func ShowOnesSellTokenOrders(seller string, tokens []string) {
	var reqAddrtokens types.ReqAddrTokens
	reqAddrtokens.Status = types.OnSale
	reqAddrtokens.Addr = seller
	if 0 != len(tokens) {
		reqAddrtokens.Token = append(reqAddrtokens.Token, tokens...)
	}
	var params jsonrpc.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetOnesSellOrder"
	params.Payload = reqAddrtokens
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplySellOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for i, sellorder := range res.Selloders {
		var sellOrders2show SellOrder2Show
		sellOrders2show.Tokensymbol = sellorder.Tokensymbol
		sellOrders2show.Seller = sellorder.Address
		sellOrders2show.Amountperboardlot = strconv.FormatFloat(float64(sellorder.Amountperboardlot)/float64(types.TokenPrecision), 'f', 4, 64)
		sellOrders2show.Minboardlot = sellorder.Minboardlot
		sellOrders2show.Priceperboardlot = strconv.FormatFloat(float64(sellorder.Priceperboardlot)/float64(types.Coin), 'f', 8, 64)
		sellOrders2show.Totalboardlot = sellorder.Totalboardlot
		sellOrders2show.Soldboardlot = sellorder.Soldboardlot
		sellOrders2show.Starttime = sellorder.Starttime
		sellOrders2show.Stoptime = sellorder.Stoptime
		sellOrders2show.Soldboardlot = sellorder.Soldboardlot
		sellOrders2show.Crowdfund = sellorder.Crowdfund
		sellOrders2show.SellID = sellorder.Sellid
		sellOrders2show.Status = types.SellOrderStatus[sellorder.Status]
		sellOrders2show.Height = sellorder.Height

		data, err := json.MarshalIndent(sellOrders2show, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Printf("---The %dth sellorder is below--------------------\n", i)
		fmt.Println(string(data))
	}
}

func QueryConfigItem(key string) {
	req := &types.ReqString{key}
	var params jsonrpc.Query4Cli
	params.Execer = "manage"
	params.FuncName = "GetConfigItem"
	params.Payload = req
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	var res types.ReplyConfig
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	fmt.Println(string(data))
}

func ShowSellOrderWithStatus(status string) {
	statusInt, ok := types.MapSellOrderStatusStr2Int[status]
	if !ok {
		fmt.Print(errors.New("参数错误\n").Error())
		return
	}
	var reqAddrtokens types.ReqAddrTokens
	reqAddrtokens.Status = statusInt

	var params jsonrpc.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetAllSellOrdersWithStatus"
	params.Payload = reqAddrtokens
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res types.ReplySellOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for i, sellorder := range res.Selloders {
		var sellOrders2show SellOrder2Show
		sellOrders2show.Tokensymbol = sellorder.Tokensymbol
		sellOrders2show.Seller = sellorder.Address
		sellOrders2show.Amountperboardlot = strconv.FormatFloat(float64(sellorder.Amountperboardlot)/float64(types.TokenPrecision), 'f', 4, 64)
		sellOrders2show.Minboardlot = sellorder.Minboardlot
		sellOrders2show.Priceperboardlot = strconv.FormatFloat(float64(sellorder.Priceperboardlot)/float64(types.Coin), 'f', 8, 64)
		sellOrders2show.Totalboardlot = sellorder.Totalboardlot
		sellOrders2show.Soldboardlot = sellorder.Soldboardlot
		sellOrders2show.Starttime = sellorder.Starttime
		sellOrders2show.Stoptime = sellorder.Stoptime
		sellOrders2show.Soldboardlot = sellorder.Soldboardlot
		sellOrders2show.Crowdfund = sellorder.Crowdfund
		sellOrders2show.SellID = sellorder.Sellid
		sellOrders2show.Status = types.SellOrderStatus[sellorder.Status]
		sellOrders2show.Height = sellorder.Height

		data, err := json.MarshalIndent(sellOrders2show, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Printf("---The %dth sellorder is below--------------------\n", i)
		fmt.Println(string(data))
	}
}

func ShowOnesBuyOrder(buyer string, tokens []string) {

	var reqAddrtokens types.ReqAddrTokens
	reqAddrtokens.Addr = buyer
	reqAddrtokens.Token = tokens

	var params jsonrpc.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetOnesBuyOrder"
	params.Payload = reqAddrtokens
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplyTradeBuyOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for i, buy := range res.Tradebuydones {
		data, err := json.MarshalIndent(buy, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Printf("---The %dth buyorder is below--------------------\n", i)
		fmt.Println(string(data))
	}
}

func ShowTokenSellOrder(args []string) {
	var req types.ReqTokenSellOrder
	req.TokenSymbol = args[0]
	count, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	direction, err := strconv.ParseInt(args[2], 10, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if direction != 0 && direction != 1 {
		fmt.Fprintln(os.Stderr, "direction must be 0 (previous-page) or 1(next-page)")
	}
	req.Count = int32(count)
	req.Direction = int32(direction)
	if len(args) == 4 {
		req.FromSellId = args[3]
	} else {
		req.FromSellId = ""
	}

	var params jsonrpc.Query4Cli
	params.Execer = "trade"
	params.FuncName = "GetTokenSellOrderByStatus"
	params.Payload = req
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res types.ReplySellOrders
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for i, sellorder := range res.Selloders {
		var sellOrders2show SellOrder2Show
		sellOrders2show.Tokensymbol = sellorder.Tokensymbol
		sellOrders2show.Seller = sellorder.Address
		sellOrders2show.Amountperboardlot = strconv.FormatFloat(float64(sellorder.Amountperboardlot)/float64(types.TokenPrecision), 'f', 4, 64)
		sellOrders2show.Minboardlot = sellorder.Minboardlot
		sellOrders2show.Priceperboardlot = strconv.FormatFloat(float64(sellorder.Priceperboardlot)/float64(types.Coin), 'f', 8, 64)
		sellOrders2show.Totalboardlot = sellorder.Totalboardlot
		sellOrders2show.Soldboardlot = sellorder.Soldboardlot
		sellOrders2show.Starttime = sellorder.Starttime
		sellOrders2show.Stoptime = sellorder.Stoptime
		sellOrders2show.Soldboardlot = sellorder.Soldboardlot
		sellOrders2show.Crowdfund = sellorder.Crowdfund
		sellOrders2show.SellID = sellorder.Sellid
		sellOrders2show.Status = types.SellOrderStatus[sellorder.Status]
		sellOrders2show.Height = sellorder.Height

		data, err := json.MarshalIndent(sellOrders2show, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Printf("---The %dth sellorder is below--------------------\n", i)
		fmt.Println(string(data))
	}
}

func decodeLog(rlog jsonrpc.ReceiptDataResult) *ReceiptData {
	rd := &ReceiptData{Ty: rlog.Ty, TyName: rlog.TyName}

	for _, l := range rlog.Logs {
		rl := &ReceiptLog{Ty: l.Ty, TyName: l.TyName, RawLog: l.RawLog}
		switch l.Ty {
		//case 1, 4, 111, 112, 113, 114:
		case types.TyLogErr, types.TyLogGenesis, types.TyLogNewTicket, types.TyLogCloseTicket, types.TyLogMinerTicket,
			types.TyLogTicketBind, types.TyLogPreCreateToken, types.TyLogFinishCreateToken, types.TyLogRevokeCreateToken,
			types.TyLogTradeSell, types.TyLogTradeBuy, types.TyLogTradeRevoke:
			rl.Log = l.Log
		//case 2, 3, 5, 11:
		case types.TyLogFee, types.TyLogTransfer, types.TyLogDeposit, types.TyLogGenesisTransfer,
			types.TyLogTokenTransfer, types.TyLogTokenDeposit:
			rl.Log = &ReceiptAccountTransfer{
				Prev:    decodeAccount(constructAccFromLog(l, "prev"), types.Coin),
				Current: decodeAccount(constructAccFromLog(l, "current"), types.Coin),
			}
		//case 6, 7, 8, 9, 10, 12:
		case types.TyLogExecTransfer, types.TyLogExecWithdraw, types.TyLogExecDeposit, types.TyLogExecFrozen, types.TyLogExecActive, types.TyLogGenesisDeposit:
			var execaddr string
			if tmp, ok := l.Log.(map[string]interface{})["execaddr"].(string); ok {
				execaddr = tmp
			}
			rl.Log = &ReceiptExecAccountTransfer{
				ExecAddr: execaddr,
				Prev:     decodeAccount(constructAccFromLog(l, "prev"), types.Coin),
				Current:  decodeAccount(constructAccFromLog(l, "current"), types.Coin),
			}
		case types.TyLogTokenExecTransfer, types.TyLogTokenExecWithdraw, types.TyLogTokenExecDeposit, types.TyLogTokenExecFrozen, types.TyLogTokenExecActive,
			types.TyLogTokenGenesisTransfer, types.TyLogTokenGenesisDeposit:
			var execaddr string
			if tmp, ok := l.Log.(map[string]interface{})["execaddr"].(string); ok {
				execaddr = tmp
			}
			rl.Log = &ReceiptExecAccountTransfer{
				ExecAddr: execaddr,
				Prev:     decodeAccount(constructAccFromLog(l, "prev"), types.TokenPrecision),
				Current:  decodeAccount(constructAccFromLog(l, "current"), types.TokenPrecision),
			}
		default:
			// fmt.Printf("---The log with vlaue:%d is not decoded --------------------\n", l.Ty)
			rl.Log = nil
		}
		rd.Logs = append(rd.Logs, rl)
	}
	return rd
}

func IsNtpClockSync() {
	rpc, err := jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var res bool
	err = rpc.Call("Chain33.IsNtpClockSync", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println("ntpclocksync status:", res)
}

func ManageConfigTransactioin(key, op, opAddr, priv string) {
	c, _ := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	a, _ := common.FromHex(priv)
	privKey, _ := c.PrivKeyFromBytes(a)
	originaddr := account.PubKeyToAddress(privKey.PubKey().Bytes()).String()

	v := &types.ModifyConfig{Key: key, Op: op, Value: opAddr, Addr: originaddr}
	modify := &types.ManageAction{
		Ty:    types.ManageActionModifyConfig,
		Value: &types.ManageAction_Modify{v},
	}
	tx := &types.Transaction{Execer: []byte("manage"), Payload: types.Encode(modify)}

	var random *rand.Rand
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()

	var err error
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	tx.Fee += types.MinFee
	tx.Sign(types.SECP256K1, privKey)
	txHex := types.Encode(tx)
	fmt.Println(hex.EncodeToString(txHex))
}

func GetTotalCoins(symbol string, height string) {
	// 获取高度statehash
	heightInt64, err := strconv.ParseInt(height, 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	params := jsonrpc.BlockParam{Start: heightInt64, End: heightInt64, Isdetail: false}
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

	stateHash, err := common.FromHex(res.Items[0].Block.StateHash)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	// 查询高度哈希对应数据
	var expectedAmount int64
	var actualAmount int64
	resp := GetTotalCoinsResult{}

	var startKey []byte
	var count int64
	for count = 1000; count == 1000; {
		params := types.ReqGetTotalCoins{Symbol: symbol, StateHash: stateHash, StartKey: startKey, Count: count}
		var res types.ReplyGetTotalCoins
		err = rpc.Call("Chain33.GetTotalCoins", params, &res)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		count = res.Num
		resp.AccountCount += res.Num
		actualAmount += res.Amount
		startKey = res.NextKey
	}

	if symbol == "bty" {
		//查询高度blockhash
		params := types.ReqInt{heightInt64}
		var res1 jsonrpc.ReplyHash
		err = rpc.Call("Chain33.GetBlockHash", params, &res1)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		blockHash, err := common.FromHex(res1.Hash)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		//查询手续费
		params2 := types.ReqHash{Hash: blockHash}
		var res2 types.TotalFee
		err = rpc.Call("Chain33.QueryTotalFee", params2, &res2)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		resp.TxCount = res2.TxCount
		expectedAmount = (3e+8+30000+30*heightInt64)*types.Coin - res2.Fee
		resp.ExpectedAmount = strconv.FormatFloat(float64(expectedAmount)/float64(types.Coin), 'f', 4, 64)
		resp.ActualAmount = strconv.FormatFloat(float64(actualAmount)/float64(types.Coin), 'f', 4, 64)
		resp.DifferenceAmount = strconv.FormatFloat(float64(expectedAmount-actualAmount)/float64(types.Coin), 'f', 4, 64)
	} else {
		//查询Token总量
		var req types.ReqString
		req.Data = symbol
		var params jsonrpc.Query4Cli
		params.Execer = "token"
		params.FuncName = "GetTokenInfo"
		params.Payload = req
		var res types.Token
		err = rpc.Call("Chain33.Query", params, &res)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		expectedAmount = res.Total
		resp.ExpectedAmount = strconv.FormatFloat(float64(expectedAmount)/float64(types.TokenPrecision), 'f', 4, 64)
		resp.ActualAmount = strconv.FormatFloat(float64(actualAmount)/float64(types.TokenPrecision), 'f', 4, 64)
		resp.DifferenceAmount = strconv.FormatFloat(float64(expectedAmount-actualAmount)/float64(types.TokenPrecision), 'f', 4, 64)
	}

	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}
