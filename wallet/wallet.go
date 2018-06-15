package wallet

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/wallet/bipwallet"
)

var (
	minFee            int64 = types.MinFee
	maxTxNumPerBlock  int64 = types.MaxTxsPerBlock
	MaxTxHashsPerTime int64 = 100
	walletlog               = log.New("module", "wallet")
	// 1；secp256k1，2：ed25519，3：sm2
	SignType    = 1
	accountdb   = account.NewCoinsAccount()
	accTokenMap = make(map[string]*account.DB)
)

const (
	AddTx = iota
	DelTx
)

type Wallet struct {
	client queue.Client
	// 模块间通信的操作接口,建议用api代替client调用
	api              client.QueueProtocolAPI
	mtx              sync.Mutex
	timeout          *time.Timer
	minertimeout     *time.Timer
	isclosed         int32
	isWalletLocked   int32
	isTicketLocked   int32
	lastHeight       int64
	autoMinerFlag    int32
	fatalFailureFlag int32
	Password         string
	FeeAmount        int64
	EncryptFlag      int64
	miningTicket     *time.Ticker
	wg               *sync.WaitGroup
	walletStore      *Store
	random           *rand.Rand
	cfg              *types.Wallet
	done             chan struct{}
	privacyActive    map[string]map[string]*walletUTXOs //不同token类型对应的公开地址拥有的隐私存款记录，map[token]map[addr]
	//privacyFrozen    map[string]string                  //[交易hash]sender
}

type walletUTXOs struct {
	outs []*txOutputInfo
}

type txOutputInfo struct {
	amount           int64
	utxoGlobalIndex  *types.UTXOGlobalIndex
	txPublicKeyR     []byte
	onetimePublicKey []byte
}

type addrAndprivacy struct {
	PrivacyKeyPair *privacy.Privacy
	Addr           *string
}

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	walletlog.SetHandler(log.DiscardHandler())
	storelog.SetHandler(log.DiscardHandler())
}

func New(cfg *types.Wallet) *Wallet {
	//walletStore
	walletStoreDB := dbm.NewDB("wallet", cfg.Driver, cfg.DbPath, cfg.DbCache)
	walletStore := NewStore(walletStoreDB)
	minFee = cfg.MinFee
	signType, exist := types.MapSignName2Type[cfg.SignType]
	if !exist {
		signType = types.SECP256K1
	}
	SignType = signType

	wallet := &Wallet{
		walletStore:      walletStore,
		isWalletLocked:   1,
		isTicketLocked:   1,
		autoMinerFlag:    0,
		fatalFailureFlag: 0,
		wg:               &sync.WaitGroup{},
		FeeAmount:        walletStore.GetFeeAmount(),
		EncryptFlag:      walletStore.GetEncryptionFlag(),
		miningTicket:     time.NewTicker(2 * time.Minute),
		done:             make(chan struct{}),
		cfg:              cfg,
	}
	value, _ := walletStore.db.Get([]byte("WalletAutoMiner"))
	if value != nil && string(value) == "1" {
		wallet.autoMinerFlag = 1
	}
	wallet.random = rand.New(rand.NewSource(time.Now().UnixNano()))
	return wallet
}

func (wallet *Wallet) setAutoMining(flag int32) {
	atomic.StoreInt32(&wallet.autoMinerFlag, flag)
}

func (wallet *Wallet) isAutoMining() bool {
	return atomic.LoadInt32(&wallet.autoMinerFlag) == 1
}

func (wallet *Wallet) Close() {
	//等待所有的子线程退出
	//set close flag to isclosed == 1
	atomic.StoreInt32(&wallet.isclosed, 1)
	wallet.miningTicket.Stop()
	close(wallet.done)
	wallet.client.Close()
	wallet.wg.Wait()
	//关闭数据库
	wallet.walletStore.db.Close()
	walletlog.Info("wallet module closed")
}

//返回钱包锁的状态
func (wallet *Wallet) IsWalletLocked() bool {
	if atomic.LoadInt32(&wallet.isWalletLocked) == 0 {
		return false
	} else {
		return true
	}
}

//返回挖矿买票锁的状态
func (wallet *Wallet) IsTicketLocked() bool {
	if atomic.LoadInt32(&wallet.isTicketLocked) == 0 {
		return false
	} else {
		return true
	}
}

func (wallet *Wallet) SetQueueClient(cli queue.Client) {
	wallet.client = cli
	wallet.client.Sub("wallet")
	wallet.api, _ = client.New(cli, nil)
	wallet.wg.Add(3)
	go wallet.ProcRecvMsg()
	go wallet.autoMining()

	//开启检查FTXO的协程
	go wallet.CheckFtxo()
}

func (wallet *Wallet) CheckFtxo() {
	defer wallet.wg.Done()

	//默认10s对Ftxo进行检查
	timecount := 10
	checkFtxoTicker := time.NewTicker(time.Duration(timecount) * time.Second)
	newbatch := wallet.walletStore.NewBatch(true)

	var lastHeight int64

	for {
		select {
		case <-checkFtxoTicker.C:
			curFTXOTxs, curKeys, err := wallet.walletStore.GetWalletFTXO()
			if nil != err {
				return
			}

			height := wallet.GetHeight()
			if lastHeight >= height {
				break
			}
			lastHeight = height

			for i, curFTXOTx := range curFTXOTxs {
				curKey := curKeys[i]
				// 说明该交易还处于FTXO，交易超时，将FTXO转化为UTXO
				str := strings.Split(curKey, ":")
				if len(str) < 2 {
					return
				}
				txhash := str[1]
				if curFTXOTx.TimeoutSec <= 0 {
					wallet.walletStore.unmoveUTXO2FTXO("", "", txhash, newbatch)
				} else {
					wallet.walletStore.updateFTXOTimeoutCount(timecount, txhash, newbatch)
				}
				newbatch.Write()
			}

		case <-wallet.done:
			return
		}
	}
}

//检查周期 --> 10分
//开启挖矿：
//1. 自动把成熟的ticket关闭
//2. 查找超过1万余额的账户，自动购买ticket
//3. 查找mineraddress 和他对应的 账户的余额（不在1中），余额超过1万的自动购买ticket 挖矿
//
//停止挖矿：
//1. 自动把成熟的ticket关闭
//2. 查找ticket 可取的余额
//3. 取出ticket 里面的钱
func (wallet *Wallet) autoMining() {
	defer wallet.wg.Done()
	for {
		select {
		case <-wallet.miningTicket.C:
			if wallet.cfg.GetMinerdisable() {
				break
			}
			if !(wallet.IsCaughtUp() || wallet.cfg.GetForceMining()) {
				walletlog.Error("wallet IsCaughtUp false")
				break
			}
			//判断高度是否增长
			height := wallet.GetHeight()
			if height <= wallet.lastHeight {
				walletlog.Error("wallet Height not inc")
				break
			}
			wallet.lastHeight = height
			walletlog.Info("BEG miningTicket")
			if wallet.isAutoMining() {
				n1, err := wallet.closeTicket(wallet.lastHeight + 1)
				if err != nil {
					walletlog.Error("closeTicket", "err", err)
				}
				err = wallet.processFees()
				if err != nil {
					walletlog.Error("processFees", "err", err)
				}
				hashes1, n2, err := wallet.buyTicket(wallet.lastHeight + 1)
				if err != nil {
					walletlog.Error("buyTicket", "err", err)
				}
				hashes2, n3, err := wallet.buyMinerAddrTicket(wallet.lastHeight + 1)
				if err != nil {
					walletlog.Error("buyMinerAddrTicket", "err", err)
				}
				hashes := append(hashes1, hashes2...)
				if len(hashes) > 0 {
					wallet.waitTxs(hashes)
				}
				if n1+n2+n3 > 0 {
					wallet.flushTicket()
				}
			} else {
				n1, err := wallet.closeTicket(wallet.lastHeight + 1)
				if err != nil {
					walletlog.Error("closeTicket", "err", err)
				}
				err = wallet.processFees()
				if err != nil {
					walletlog.Error("processFees", "err", err)
				}
				hashes, err := wallet.withdrawFromTicket()
				if err != nil {
					walletlog.Error("withdrawFromTicket", "err", err)
				}
				if len(hashes) > 0 {
					wallet.waitTxs(hashes)
				}
				if n1 > 0 {
					wallet.flushTicket()
				}
			}
			walletlog.Info("END miningTicket")
		case <-wallet.done:
			return
		}
	}
}

func (wallet *Wallet) buyTicket(height int64) ([][]byte, int, error) {
	privs, err := wallet.getAllPrivKeys()
	if err != nil {
		walletlog.Error("buyTicket.getAllPrivKeys", "err", err)
		return nil, 0, err
	}
	count := 0
	var hashes [][]byte
	for _, priv := range privs {
		hash, n, err := wallet.buyTicketOne(height, priv)
		if err != nil {
			walletlog.Error("buyTicketOne", "err", err)
			continue
		}
		count += n
		if hash != nil {
			hashes = append(hashes, hash)
		}
	}
	return hashes, count, nil
}

func (wallet *Wallet) buyMinerAddrTicket(height int64) ([][]byte, int, error) {
	privs, err := wallet.getAllPrivKeys()
	if err != nil {
		walletlog.Error("buyMinerAddrTicket.getAllPrivKeys", "err", err)
		return nil, 0, err
	}
	count := 0
	var hashes [][]byte
	for _, priv := range privs {
		hashlist, n, err := wallet.buyMinerAddrTicketOne(height, priv)
		if err != nil {
			if err != types.ErrNotFound {
				walletlog.Error("buyMinerAddrTicketOne", "err", err)
			}
			continue
		}
		count += n
		if hashlist != nil {
			hashes = append(hashes, hashlist...)
		}
	}
	return hashes, count, nil
}

func (wallet *Wallet) withdrawFromTicket() (hashes [][]byte, err error) {
	privs, err := wallet.getAllPrivKeys()
	if err != nil {
		walletlog.Error("withdrawFromTicket.getAllPrivKeys", "err", err)
		return nil, err
	}
	for _, priv := range privs {
		hash, err := wallet.withdrawFromTicketOne(priv)
		if err != nil {
			walletlog.Error("withdrawFromTicketOne", "err", err)
			continue
		}
		if hash != nil {
			hashes = append(hashes, hash)
		}
	}
	return hashes, nil
}

func (wallet *Wallet) closeTicket(height int64) (int, error) {
	return wallet.closeAllTickets(height)
}

func (wallet *Wallet) forceCloseTicket(height int64) (*types.ReplyHashes, error) {
	return wallet.forceCloseAllTicket(height)
}

func (wallet *Wallet) flushTicket() {
	walletlog.Info("wallet FLUSH TICKET")
	hashList := wallet.client.NewMessage("consensus", types.EventFlushTicket, nil)
	wallet.client.Send(hashList, false)
}

func (wallet *Wallet) ProcRecvMsg() {
	defer wallet.wg.Done()
	for msg := range wallet.client.Recv() {
		walletlog.Debug("wallet recv", "msg", msg.Id)
		msgtype := msg.Ty
		switch msgtype {

		case types.EventWalletGetAccountList:
			WalletAccounts, err := wallet.ProcGetAccountList()
			if err != nil {
				walletlog.Error("ProcGetAccountList", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccountList, err))
			} else {
				walletlog.Debug("process WalletAccounts OK")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccountList, WalletAccounts))
			}

		case types.EventWalletAutoMiner:
			flag := msg.GetData().(*types.MinerFlag).Flag
			if flag == 1 {
				wallet.walletStore.db.Set([]byte("WalletAutoMiner"), []byte("1"))
			} else {
				wallet.walletStore.db.Set([]byte("WalletAutoMiner"), []byte("0"))
			}
			wallet.setAutoMining(flag)
			wallet.flushTicket()
			msg.ReplyErr("WalletSetAutoMiner", nil)

		case types.EventWalletGetTickets:
			tickets, privs, err := wallet.GetTickets(1)
			if err != nil {
				walletlog.Error("GetTickets", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("consensus", types.EventWalletTickets, err))
			} else {
				tks := &types.ReplyWalletTickets{tickets, privs}
				walletlog.Debug("process GetTickets OK")
				msg.Reply(wallet.client.NewMessage("consensus", types.EventWalletTickets, tks))
			}

		case types.EventNewAccount:
			NewAccount := msg.Data.(*types.ReqNewAccount)
			WalletAccount, err := wallet.ProcCreateNewAccount(NewAccount)
			if err != nil {
				walletlog.Error("ProcCreateNewAccount", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, WalletAccount))
			}

		case types.EventWalletTransactionList:
			WalletTxList := msg.Data.(*types.ReqWalletTransactionList)
			TransactionDetails, err := wallet.ProcWalletTxList(WalletTxList)
			if err != nil {
				walletlog.Error("ProcWalletTxList", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventTransactionDetails, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventTransactionDetails, TransactionDetails))
			}

		case types.EventWalletImportprivkey:
			ImportPrivKey := msg.Data.(*types.ReqWalletImportPrivKey)
			WalletAccount, err := wallet.ProcImportPrivKey(ImportPrivKey)
			if err != nil {
				walletlog.Error("ProcImportPrivKey", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, WalletAccount))
			}
			wallet.flushTicket()

		case types.EventWalletSendToAddress:
			SendToAddress := msg.Data.(*types.ReqWalletSendToAddress)
			ReplyHash, err := wallet.ProcSendToAddress(SendToAddress)
			if err != nil {
				walletlog.Error("ProcSendToAddress", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, ReplyHash))
			}

		case types.EventWalletSetFee:
			WalletSetFee := msg.Data.(*types.ReqWalletSetFee)

			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletSetFee(WalletSetFee)
			if err != nil {
				walletlog.Error("ProcWalletSetFee", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReply, &reply))

		case types.EventWalletSetLabel:
			WalletSetLabel := msg.Data.(*types.ReqWalletSetLabel)
			WalletAccount, err := wallet.ProcWalletSetLabel(WalletSetLabel)

			if err != nil {
				walletlog.Error("ProcWalletSetLabel", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventWalletAccount, WalletAccount))
			}

		case types.EventWalletMergeBalance:
			MergeBalance := msg.Data.(*types.ReqWalletMergeBalance)
			ReplyHashes, err := wallet.ProcMergeBalance(MergeBalance)
			if err != nil {
				walletlog.Error("ProcMergeBalance", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, ReplyHashes))
			}

		case types.EventWalletSetPasswd:
			SetPasswd := msg.Data.(*types.ReqWalletSetPasswd)

			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletSetPasswd(SetPasswd)
			if err != nil {
				walletlog.Error("ProcWalletSetPasswd", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReply, &reply))

		case types.EventWalletLock:
			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletLock()
			if err != nil {
				walletlog.Error("ProcWalletLock", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReply, &reply))

		case types.EventWalletUnLock:
			WalletUnLock := msg.Data.(*types.WalletUnLock)
			var reply types.Reply
			reply.IsOk = true
			err := wallet.ProcWalletUnLock(WalletUnLock)
			if err != nil {
				walletlog.Error("ProcWalletUnLock", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReply, &reply))
			wallet.flushTicket()

		case types.EventAddBlock:
			block := msg.Data.(*types.BlockDetail)
			wallet.ProcWalletAddBlock(block)
			walletlog.Debug("wallet add block --->", "height", block.Block.GetHeight())

		case types.EventDelBlock:
			block := msg.Data.(*types.BlockDetail)
			wallet.ProcWalletDelBlock(block)
			walletlog.Debug("wallet del block --->", "height", block.Block.GetHeight())

		//seed
		case types.EventGenSeed:
			genSeedLang := msg.Data.(*types.GenSeedLang)
			replySeed, err := wallet.genSeed(genSeedLang.Lang)
			if err != nil {
				walletlog.Error("genSeed", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyGenSeed, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyGenSeed, replySeed))
			}

		case types.EventGetSeed:
			Pw := msg.Data.(*types.GetSeedByPw)
			seed, err := wallet.getSeed(Pw.Passwd)
			if err != nil {
				walletlog.Error("getSeed", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyGetSeed, err))
			} else {
				var replySeed types.ReplySeed
				replySeed.Seed = seed
				//walletlog.Error("EventGetSeed", "seed", seed)
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyGetSeed, &replySeed))
			}

		case types.EventSaveSeed:
			saveseed := msg.Data.(*types.SaveSeedByPw)
			var reply types.Reply
			reply.IsOk = true
			ok, err := wallet.saveSeed(saveseed.Passwd, saveseed.Seed)
			if !ok {
				walletlog.Error("saveSeed", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReply, &reply))

		case types.EventGetWalletStatus:
			s := wallet.GetWalletStatus()
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyWalletStatus, s))

		case types.EventDumpPrivkey:
			addr := msg.Data.(*types.ReqStr)
			privkey, err := wallet.ProcDumpPrivkey(addr.ReqStr)
			if err != nil {
				walletlog.Error("ProcDumpPrivkey", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivkey, err))
			} else {
				var replyStr types.ReplyStr
				replyStr.Replystr = privkey
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivkey, &replyStr))
			}

		case types.EventCloseTickets:
			hashes, err := wallet.forceCloseTicket(wallet.GetHeight() + 1)
			if err != nil {
				walletlog.Error("closeTicket", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyHashes, hashes))
				go func() {
					if len(hashes.Hashes) > 0 {
						wallet.waitTxs(hashes.Hashes)
						wallet.flushTicket()
					}
				}()
			}

		case types.EventSignRawTx:
			unsigned := msg.GetData().(*types.ReqSignRawTx)
			txHex, err := wallet.ProcSignRawTx(unsigned)
			if err != nil {
				walletlog.Error("EventSignRawTx", "err", err)
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplySignRawTx, err))
			} else {
				walletlog.Info("Reply EventSignRawTx", "msg", msg)
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplySignRawTx, &types.ReplySignRawTx{TxHex: txHex}))
			}
		case types.EventErrToFront: //收到系统发生致命性错误事件
			reportErrEvent := msg.Data.(*types.ReportErrEvent)
			wallet.setFatalFailure(reportErrEvent)
			walletlog.Debug("EventErrToFront")

		case types.EventFatalFailure: //定时查询是否有致命性故障产生
			fatalFailure := wallet.getFatalFailure()
			msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyFatalFailure, &types.Int32{Data: fatalFailure}))

		case types.EventShowPrivacyBalance:
			reqPrivBal4AddrToken := msg.Data.(*types.ReqPrivBal4AddrToken)
			accout, err := wallet.showPrivacyBalance(reqPrivBal4AddrToken)
			if err != nil {
				walletlog.Error("showPrivacyAccount", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyBalance, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyBalance, accout))
			}

		case types.EventShowPrivacyAccount:
			reqPrivBal4AddrToken := msg.Data.(*types.ReqPrivBal4AddrToken)
			UTXOs, err := wallet.showPrivacyAccounts(reqPrivBal4AddrToken)
			if err != nil {
				walletlog.Error("showPrivacyAccount", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyAccount, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyAccount, UTXOs))
			}
		case types.EventShowPrivacyAccountSpend:
			reqPrivBal4AddrToken := msg.Data.(*types.ReqPrivBal4AddrToken)
			UTXOs, err := wallet.showPrivacyAccountsSpend(reqPrivBal4AddrToken)
			if err != nil {
				walletlog.Error("showPrivacyAccountSpend", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyAccountSpend, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyAccountSpend, UTXOs))
			}
		case types.EventShowPrivacyPK:
			reqAddr := msg.Data.(*types.ReqStr)
			replyPrivacyPair, err := wallet.showPrivacyPkPair(reqAddr)
			if err != nil {
				walletlog.Error("showPrivacyPkPair", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyPK, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyPK, replyPrivacyPair))
			}
		case types.EventPublic2privacy:
			reqPub2Pri := msg.Data.(*types.ReqPub2Pri)
			replyHash, err := wallet.procPublic2PrivacyV2(reqPub2Pri)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procPublic2Privacy", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPublic2privacy, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procPublic2Privacy", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPublic2privacy, &reply))
			}

		case types.EventPrivacy2privacy:
			reqPri2Pri := msg.Data.(*types.ReqPri2Pri)
			replyHash, err := wallet.procPrivacy2PrivacyV2(reqPri2Pri)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procPrivacy2Privacy", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivacy2privacy, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procPrivacy2Privacy", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivacy2privacy, &reply))
			}
		case types.EventPrivacy2public:
			reqPri2Pub := msg.Data.(*types.ReqPri2Pub)
			replyHash, err := wallet.procPrivacy2PublicV2(reqPri2Pub)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procPrivacy2Public", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivacy2public, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procPrivacy2Public", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivacy2public, &reply))
			}
		case types.EventCreateUTXOs:
			reqCreateUTXOs := msg.Data.(*types.ReqCreateUTXOs)
			replyHash, err := wallet.procCreateUTXOs(reqCreateUTXOs)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procCreateUTXOs", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyCreateUTXOs, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procCreateUTXOs", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyCreateUTXOs, &reply))
			}
		default:
			walletlog.Info("ProcRecvMsg unknow msg", "msgtype", msgtype)
		}
		walletlog.Debug("end process", "msg.id", msg.Id)
	}
}

//input:
//type ReqSignRawTx struct {
//	Addr    string
//	Privkey string
//	TxHex   string
//	Expire  string
//}
//output:
//string
//签名交易
func (wallet *Wallet) ProcSignRawTx(unsigned *types.ReqSignRawTx) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	index := unsigned.Index
	if unsigned.GetAddr() == "" {
		return "", types.ErrNoPrivKeyOrAddr
	}

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return "", err
	}
	key, err := wallet.getPrivKeyByAddr(unsigned.GetAddr())
	if err != nil {
		return "", err
	}

	var tx types.Transaction
	bytes, err := common.FromHex(unsigned.GetTxHex())
	if err != nil {
		return "", err
	}
	err = types.Decode(bytes, &tx)
	if err != nil {
		return "", err
	}
	expire, err := time.ParseDuration(unsigned.GetExpire())
	if err != nil {
		return "", err
	}
	tx.SetExpire(expire)
	group, err := tx.GetTxGroup()
	if err != nil {
		return "", err
	}
	if group == nil {
		tx.Sign(int32(SignType), key)
		txHex := types.Encode(&tx)
		signedTx := hex.EncodeToString(txHex)
		return signedTx, nil
	}
	if int(index) > len(group.GetTxs()) {
		return "", types.ErrIndex
	}
	if index <= 0 {
		for i := range group.Txs {
			group.SignN(i, int32(SignType), key)
		}
		grouptx := group.Tx()
		txHex := types.Encode(grouptx)
		signedTx := hex.EncodeToString(txHex)
		return signedTx, nil
	} else {
		index -= 1
		group.SignN(int(index), int32(SignType), key)
		grouptx := group.Tx()
		txHex := types.Encode(grouptx)
		signedTx := hex.EncodeToString(txHex)
		return signedTx, nil
	}
	return "", nil
}

//output:
//type WalletAccounts struct {
//	Wallets []*WalletAccount
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//获取钱包的地址列表
func (wallet *Wallet) ProcGetAccountList() (*types.WalletAccounts, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	//通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Info("ProcGetAccountList", "GetAccountByPrefix:err", err)
		return nil, err
	}

	addrs := make([]string, len(WalletAccStores))
	for index, AccStore := range WalletAccStores {
		if len(AccStore.Addr) != 0 {
			addrs[index] = AccStore.Addr
		}
		//walletlog.Debug("ProcGetAccountList", "all AccStore", AccStore.String())
	}
	//获取所有地址对应的账户详细信息从account模块
	accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcGetAccountList", "LoadAccounts:err", err)
		return nil, err
	}

	//异常打印信息
	if len(WalletAccStores) != len(accounts) {
		walletlog.Error("ProcGetAccountList err!", "AccStores)", len(WalletAccStores), "accounts", len(accounts))
	}

	var WalletAccounts types.WalletAccounts
	WalletAccounts.Wallets = make([]*types.WalletAccount, len(WalletAccStores))

	for index, Account := range accounts {
		var WalletAccount types.WalletAccount
		//此账户还没有参与交易所在account模块没有记录
		if len(Account.Addr) == 0 {
			Account.Addr = addrs[index]
		}
		WalletAccount.Acc = Account
		WalletAccount.Label = WalletAccStores[index].GetLabel()
		WalletAccounts.Wallets[index] = &WalletAccount

		//walletlog.Info("ProcGetAccountList", "LoadAccounts:account", Account.String())
	}
	return &WalletAccounts, nil
}

//input:
//type ReqNewAccount struct {
//	Label string
//output:
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//type Account struct {
//	Currency int32
//	Balance  int64
//	Frozen   int64
//	Addr     string
//创建一个新的账户
func (wallet *Wallet) ProcCreateNewAccount(Label *types.ReqNewAccount) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}

	if Label == nil || len(Label.GetLabel()) == 0 {
		walletlog.Error("ProcCreateNewAccount Label is nil")
		return nil, types.ErrInputPara
	}

	//首先校验label是否已被使用
	WalletAccStores, err := wallet.walletStore.GetAccountByLabel(Label.GetLabel())
	if WalletAccStores != nil {
		walletlog.Error("ProcCreateNewAccount Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
	}

	var Account types.Account
	var walletAccount types.WalletAccount
	var WalletAccStore types.WalletAccountStore
	var cointype uint32
	if SignType == 1 {
		cointype = bipwallet.TypeBty
	} else if SignType == 2 {
		cointype = bipwallet.TypeYcc
	} else {
		cointype = bipwallet.TypeBty
	}

	//通过seed获取私钥, 首先通过钱包密码解锁seed然后通过seed生成私钥
	seed, err := wallet.getSeed(wallet.Password)
	if err != nil {
		walletlog.Error("ProcCreateNewAccount", "getSeed err", err)
		return nil, err
	}
	privkeyhex, err := GetPrivkeyBySeed(wallet.walletStore.db, seed)
	if err != nil {
		walletlog.Error("ProcCreateNewAccount", "GetPrivkeyBySeed err", err)
		return nil, err
	}
	privkeybyte, err := common.FromHex(privkeyhex)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("ProcCreateNewAccount", "FromHex err", err)
		return nil, err
	}

	pub, err := bipwallet.PrivkeyToPub(cointype, privkeybyte)
	if err != nil {
		seedlog.Error("ProcCreateNewAccount PrivkeyToPub", "err", err)
		return nil, types.ErrPrivkeyToPub
	}
	addr, err := bipwallet.PubToAddress(cointype, pub)
	if err != nil {
		seedlog.Error("ProcCreateNewAccount PubToAddress", "err", err)
		return nil, types.ErrPrivkeyToPub
	}

	Account.Addr = addr
	Account.Currency = 0
	Account.Balance = 0
	Account.Frozen = 0

	walletAccount.Acc = &Account
	walletAccount.Label = Label.GetLabel()

	//使用钱包的password对私钥加密 aes cbc
	Encrypted := CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	WalletAccStore.Privkey = common.ToHex(Encrypted)
	WalletAccStore.Label = Label.GetLabel()
	WalletAccStore.Addr = addr

	//存储账户信息到wallet数据库中
	err = wallet.walletStore.SetWalletAccount(false, Account.Addr, &WalletAccStore)
	if err != nil {
		return nil, err
	}

	//获取地址对应的账户信息从account模块
	addrs := make([]string, 1)
	addrs[0] = addr
	accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil {
		walletlog.Error("ProcCreateNewAccount", "LoadAccounts err", err)
		return nil, err
	}
	// 本账户是首次创建
	if len(accounts[0].Addr) == 0 {
		accounts[0].Addr = addr
	}
	walletAccount.Acc = accounts[0]

	//从blockchain模块同步Account.Addr对应的所有交易详细信息
	wallet.wg.Add(1)
	go wallet.ReqTxDetailByAddr(addr)
	wallet.wg.Add(1)
	go wallet.ReqPrivacyTxByAddr(addr)

	return &walletAccount, nil
}

//input:
//type ReqWalletTransactionList struct {
//	FromTx []byte
//	Count  int32
//output:
//type WalletTxDetails struct {
//	TxDetails []*WalletTxDetail
//type WalletTxDetail struct {
//	Tx      *Transaction
//	Receipt *ReceiptData
//	Height  int64
//	Index   int64
//获取所有钱包的交易记录
func (wallet *Wallet) ProcWalletTxList(TxList *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if TxList == nil {
		walletlog.Error("ProcWalletTxList TxList is nil!")
		return nil, types.ErrInputPara
	}
	if TxList.GetDirection() != 0 && TxList.GetDirection() != 1 {
		walletlog.Error("ProcWalletTxList Direction err!")
		return nil, types.ErrInputPara
	}
	WalletTxDetails, err := wallet.walletStore.GetTxDetailByIter(TxList)
	if err != nil {
		walletlog.Error("ProcWalletTxList", "GetTxDetailByIter err", err)
		return nil, err
	}
	return WalletTxDetails, nil
}

//input:
//type ReqWalletImportPrivKey struct {
//	Privkey string
//	Label   string
//output:
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//导入私钥，并且同时会导入交易
func (wallet *Wallet) ProcImportPrivKey(PrivKey *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}

	if PrivKey == nil || len(PrivKey.GetLabel()) == 0 || len(PrivKey.GetPrivkey()) == 0 {
		walletlog.Error("ProcImportPrivKey input parameter is nil!")
		return nil, types.ErrInputPara
	}

	//校验label是否已经被使用
	Account, err := wallet.walletStore.GetAccountByLabel(PrivKey.GetLabel())
	if Account != nil {
		walletlog.Error("ProcImportPrivKey Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
	}

	var cointype uint32
	if SignType == 1 {
		cointype = bipwallet.TypeBty
	} else if SignType == 2 {
		cointype = bipwallet.TypeYcc
	} else {
		cointype = bipwallet.TypeBty
	}

	privkeybyte, err := common.FromHex(PrivKey.Privkey)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("ProcImportPrivKey", "FromHex err", err)
		return nil, types.ErrFromHex
	}

	pub, err := bipwallet.PrivkeyToPub(cointype, privkeybyte)
	if err != nil {
		seedlog.Error("ProcImportPrivKey PrivkeyToPub", "err", err)
		return nil, types.ErrPrivkeyToPub
	}
	addr, err := bipwallet.PubToAddress(cointype, pub)
	if err != nil {
		seedlog.Error("ProcImportPrivKey PrivkeyToPub", "err", err)
		return nil, types.ErrPrivkeyToPub
	}

	//对私钥加密
	Encryptered := CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	Encrypteredstr := common.ToHex(Encryptered)
	//校验PrivKey对应的addr是否已经存在钱包中
	Account, err = wallet.walletStore.GetAccountByAddr(addr)
	if Account != nil {
		if Account.Privkey == Encrypteredstr {
			walletlog.Error("ProcImportPrivKey Privkey is exist in wallet!")
			return nil, types.ErrPrivkeyExist
		} else {
			walletlog.Error("ProcImportPrivKey!", "Account.Privkey", Account.Privkey, "input Privkey", PrivKey.Privkey)
			return nil, types.ErrPrivkey
		}
	}

	var walletaccount types.WalletAccount
	var WalletAccStore types.WalletAccountStore
	WalletAccStore.Privkey = Encrypteredstr //存储加密后的私钥
	WalletAccStore.Label = PrivKey.GetLabel()
	WalletAccStore.Addr = addr
	//存储Addr:label+privkey+addr到数据库
	err = wallet.walletStore.SetWalletAccount(false, addr, &WalletAccStore)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "SetWalletAccount err", err)
		return nil, err
	}

	//获取地址对应的账户信息从account模块
	addrs := make([]string, 1)
	addrs[0] = addr
	accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "LoadAccounts err", err)
		return nil, err
	}
	// 本账户是首次创建
	if len(accounts[0].Addr) == 0 {
		accounts[0].Addr = addr
	}
	walletaccount.Acc = accounts[0]
	walletaccount.Label = PrivKey.Label

	//从blockchain模块同步Account.Addr对应的所有交易详细信息
	wallet.wg.Add(1)
	go wallet.ReqTxDetailByAddr(addr)
	wallet.wg.Add(1)
	go wallet.ReqPrivacyTxByAddr(addr)

	return &walletaccount, nil
}

//input:
//type ReqWalletSendToAddress struct {
//	From   string
//	To     string
//	Amount int64
//	Note   string
//output:
//type ReplyHash struct {
//	Hashe []byte
//发送一笔交易给对方地址，返回交易hash
func (wallet *Wallet) ProcSendToAddress(SendToAddress *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if SendToAddress == nil {
		walletlog.Error("ProcSendToAddress input para is nil")
		return nil, types.ErrInputPara
	}
	if len(SendToAddress.From) == 0 || len(SendToAddress.To) == 0 {
		walletlog.Error("ProcSendToAddress input para From or To is nil!")
		return nil, types.ErrInputPara
	}

	ok, err := wallet.IsTransfer(SendToAddress.GetTo())
	if !ok {
		return nil, err
	}

	//获取from账户的余额从account模块，校验余额是否充足
	addrs := make([]string, 1)
	addrs[0] = SendToAddress.GetFrom()
	var accounts []*types.Account
	var tokenAccounts []*types.Account
	accounts, err = accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcSendToAddress", "LoadAccounts err", err)
		return nil, err
	}
	Balance := accounts[0].Balance
	amount := SendToAddress.GetAmount()
	if !SendToAddress.IsToken {
		if Balance < amount+wallet.FeeAmount {
			return nil, types.ErrInsufficientBalance
		}
	} else {
		//如果是token转账，一方面需要保证coin的余额满足fee，另一方面则需要保证token的余额满足转账操作
		if Balance < wallet.FeeAmount {
			return nil, types.ErrInsufficientBalance
		}

		if nil == accTokenMap[SendToAddress.TokenSymbol] {
			tokenAccDB, err := account.NewAccountDB("token", SendToAddress.TokenSymbol, nil)
			if err != nil {
				return nil, err
			}
			accTokenMap[SendToAddress.TokenSymbol] = tokenAccDB
		}
		tokenAccDB := accTokenMap[SendToAddress.TokenSymbol]
		tokenAccounts, err = tokenAccDB.LoadAccounts(wallet.api, addrs)
		if err != nil || len(tokenAccounts) == 0 {
			walletlog.Error("ProcSendToAddress", "Load Token Accounts err", err)
			return nil, err
		}
		tokenBalance := tokenAccounts[0].Balance
		if tokenBalance < amount {
			return nil, types.ErrInsufficientTokenBal
		}
	}
	addrto := SendToAddress.GetTo()
	note := SendToAddress.GetNote()
	priv, err := wallet.getPrivKeyByAddr(addrs[0])
	if err != nil {
		return nil, err
	}
	return wallet.sendToAddress(priv, addrto, amount, note, SendToAddress.IsToken, SendToAddress.TokenSymbol)
}

func (wallet *Wallet) getPrivKeyByAddr(addr string) (crypto.PrivKey, error) {
	//获取指定地址在钱包里的账户信息
	Accountstor, err := wallet.walletStore.GetAccountByAddr(addr)
	if err != nil {
		walletlog.Error("ProcSendToAddress", "GetAccountByAddr err:", err)
		return nil, err
	}

	//通过password解密存储的私钥
	prikeybyte, err := common.FromHex(Accountstor.GetPrivkey())
	if err != nil || len(prikeybyte) == 0 {
		walletlog.Error("ProcSendToAddress", "FromHex err", err)
		return nil, err
	}

	privkey := CBCDecrypterPrivkey([]byte(wallet.Password), prikeybyte)
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(SignType))
	if err != nil {
		walletlog.Error("ProcSendToAddress", "err", err)
		return nil, err
	}
	priv, err := cr.PrivKeyFromBytes(privkey)
	if err != nil {
		walletlog.Error("ProcSendToAddress", "PrivKeyFromBytes err", err)
		return nil, err
	}
	return priv, nil
}

//type ReqWalletSetFee struct {
//	Amount int64
//设置钱包默认的手续费
func (wallet *Wallet) ProcWalletSetFee(WalletSetFee *types.ReqWalletSetFee) error {
	if WalletSetFee.Amount < minFee {
		walletlog.Error("ProcWalletSetFee err!", "Amount", WalletSetFee.Amount, "MinFee", minFee)
		return types.ErrInputPara
	}
	err := wallet.walletStore.SetFeeAmount(WalletSetFee.Amount)
	if err == nil {
		walletlog.Debug("ProcWalletSetFee success!")
		wallet.mtx.Lock()
		wallet.FeeAmount = WalletSetFee.Amount
		wallet.mtx.Unlock()
	}
	return err
}

//外部已经加了lock
func (wallet *Wallet) getFee() int64 {
	return wallet.FeeAmount
}

//input:
//type ReqWalletSetLabel struct {
//	Addr  string
//	Label string
//output:
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//设置某个账户的标签
func (wallet *Wallet) ProcWalletSetLabel(SetLabel *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if SetLabel == nil || len(SetLabel.Addr) == 0 || len(SetLabel.Label) == 0 {
		walletlog.Error("ProcWalletSetLabel input parameter is nil!")
		return nil, types.ErrInputPara
	}
	//校验label是否已经被使用
	Account, err := wallet.walletStore.GetAccountByLabel(SetLabel.GetLabel())
	if Account != nil {
		walletlog.Error("ProcWalletSetLabel Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
	}
	//获取地址对应的账户信息从钱包中,然后修改label
	Account, err = wallet.walletStore.GetAccountByAddr(SetLabel.Addr)
	if err == nil && Account != nil {
		oldLabel := Account.Label
		Account.Label = SetLabel.GetLabel()
		err := wallet.walletStore.SetWalletAccount(true, SetLabel.Addr, Account)
		if err == nil {
			//新的label设置成功之后需要删除旧的label在db的数据
			wallet.walletStore.DelAccountByLabel(oldLabel)

			//获取地址对应的账户详细信息从account模块
			addrs := make([]string, 1)
			addrs[0] = SetLabel.Addr
			accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
			if err != nil || len(accounts) == 0 {
				walletlog.Error("ProcWalletSetLabel", "LoadAccounts err", err)
				return nil, err
			}
			var walletAccount types.WalletAccount
			walletAccount.Acc = accounts[0]
			walletAccount.Label = SetLabel.GetLabel()
			return &walletAccount, err
		}
	}
	return nil, err
}

//input:
//type ReqWalletMergeBalance struct {
//	To string
//output:
//type ReplyHashes struct {
//	Hashes [][]byte
//合并所有的balance 到一个地址
func (wallet *Wallet) ProcMergeBalance(MergeBalance *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}

	if len(MergeBalance.GetTo()) == 0 {
		walletlog.Error("ProcMergeBalance input para is nil!")
		return nil, types.ErrInputPara
	}

	//获取钱包上的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Error("ProcMergeBalance", "GetAccountByPrefix err", err)
		return nil, err
	}

	addrs := make([]string, len(WalletAccStores))
	for index, AccStore := range WalletAccStores {
		if len(AccStore.Addr) != 0 {
			addrs[index] = AccStore.Addr
		}
	}
	//获取所有地址对应的账户信息从account模块
	accounts, err := accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcMergeBalance", "LoadAccounts err", err)
		return nil, err
	}

	//异常信息记录
	if len(WalletAccStores) != len(accounts) {
		walletlog.Error("ProcMergeBalance", "AccStores", len(WalletAccStores), "accounts", len(accounts))
	}
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(SignType))
	if err != nil {
		walletlog.Error("ProcMergeBalance", "err", err)
		return nil, err
	}

	addrto := MergeBalance.GetTo()
	note := "MergeBalance"

	var ReplyHashes types.ReplyHashes

	for index, Account := range accounts {
		Privkey := WalletAccStores[index].Privkey
		//解密存储的私钥
		prikeybyte, err := common.FromHex(Privkey)
		if err != nil || len(prikeybyte) == 0 {
			walletlog.Error("ProcMergeBalance", "FromHex err", err, "index", index)
			continue
		}

		privkey := CBCDecrypterPrivkey([]byte(wallet.Password), prikeybyte)
		priv, err := cr.PrivKeyFromBytes(privkey)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "PrivKeyFromBytes err", err, "index", index)
			continue
		}
		//过滤掉to地址
		if Account.Addr == addrto {
			continue
		}
		//获取账户的余额，过滤掉余额不足的地址
		amount := Account.GetBalance()
		if amount < wallet.FeeAmount {
			continue
		}
		amount = amount - wallet.FeeAmount
		v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount, Note: note}}
		transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
		//初始化随机数
		//r := rand.New(rand.NewSource(time.Now().UnixNano()))
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: wallet.FeeAmount, To: addrto, Nonce: wallet.random.Int63()}
		tx.SetExpire(time.Second * 120)
		tx.Sign(int32(SignType), priv)

		//发送交易信息给mempool模块
		msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
		wallet.client.Send(msg, true)
		resp, err := wallet.client.Wait(msg)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "Send tx err", err, "index", index)
			continue
		}
		//如果交易在mempool校验失败，不记录此交易
		reply := resp.GetData().(*types.Reply)
		if !reply.GetIsOk() {
			walletlog.Error("ProcMergeBalance", "Send tx err", string(reply.GetMsg()), "index", index)
			continue
		}
		ReplyHashes.Hashes = append(ReplyHashes.Hashes, tx.Hash())
	}
	return &ReplyHashes, nil
}

//input:
//type ReqWalletSetPasswd struct {
//	Oldpass string
//	Newpass string
//设置或者修改密码
func (wallet *Wallet) ProcWalletSetPasswd(Passwd *types.ReqWalletSetPasswd) error {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	isok, err := wallet.CheckWalletStatus()
	if !isok && err == types.ErrSaveSeedFirst {
		return err
	}
	//保存钱包的锁状态，需要暂时的解锁，函数退出时再恢复回去
	tempislock := atomic.LoadInt32(&wallet.isWalletLocked)
	//wallet.isWalletLocked = false
	atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 1, 0)

	defer func() {
		//wallet.isWalletLocked = tempislock
		atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, tempislock)
	}()

	// 钱包已经加密需要验证oldpass的正确性
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		isok := wallet.walletStore.VerifyPasswordHash(Passwd.GetOldpass())
		if !isok {
			walletlog.Error("ProcWalletSetPasswd Verify Oldpasswd fail!")
			return types.ErrVerifyOldpasswdFail
		}
	}

	if len(wallet.Password) != 0 && Passwd.GetOldpass() != wallet.Password {
		walletlog.Error("ProcWalletSetPasswd Oldpass err!")
		return types.ErrVerifyOldpasswdFail
	}

	//使用新的密码生成passwdhash用于下次密码的验证
	err = wallet.walletStore.SetPasswordHash(Passwd.GetNewpass())
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "SetPasswordHash err", err)
		return err
	}
	//设置钱包加密标志位
	err = wallet.walletStore.SetEncryptionFlag()
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "SetEncryptionFlag err", err)
		return err
	}
	//使用old密码解密seed然后用新的钱包密码重新加密seed
	seed, err := wallet.getSeed(Passwd.GetOldpass())
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "getSeed err", err)
		return err
	}
	ok, err := SaveSeed(wallet.walletStore.db, seed, Passwd.GetNewpass())
	if !ok {
		walletlog.Error("ProcWalletSetPasswd", "SaveSeed err", err)
		return err
	}

	//对所有存储的私钥重新使用新的密码加密,通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Error("ProcWalletSetPasswd", "GetAccountByPrefix:err", err)
	}

	for _, AccStore := range WalletAccStores {
		//使用old Password解密存储的私钥
		storekey, err := common.FromHex(AccStore.GetPrivkey())
		if err != nil || len(storekey) == 0 {
			walletlog.Error("ProcWalletSetPasswd", "addr", AccStore.Addr, "FromHex err", err)
			continue
		}
		Decrypter := CBCDecrypterPrivkey([]byte(Passwd.GetOldpass()), storekey)

		//使用新的密码重新加密私钥
		Encrypter := CBCEncrypterPrivkey([]byte(Passwd.GetNewpass()), Decrypter)
		AccStore.Privkey = common.ToHex(Encrypter)
		err = wallet.walletStore.SetWalletAccount(true, AccStore.Addr, AccStore)
		if err != nil {
			walletlog.Error("ProcWalletSetPasswd", "addr", AccStore.Addr, "SetWalletAccount err", err)
		}
	}

	wallet.Password = Passwd.GetNewpass()
	return nil
}

//锁定钱包
func (wallet *Wallet) ProcWalletLock() error {
	//判断钱包是否已保存seed
	has, _ := HasSeed(wallet.walletStore.db)
	if !has {
		return types.ErrSaveSeedFirst
	}

	atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, 1)
	atomic.CompareAndSwapInt32(&wallet.isTicketLocked, 0, 1)
	return nil
}

//input:
//type WalletUnLock struct {
//	Passwd  string
//	Timeout int64
//解锁钱包Timeout时间，超时后继续锁住
func (wallet *Wallet) ProcWalletUnLock(WalletUnLock *types.WalletUnLock) error {
	//判断钱包是否已保存seed
	has, _ := HasSeed(wallet.walletStore.db)
	if !has {
		return types.ErrSaveSeedFirst
	}
	// 钱包已经加密需要验证passwd的正确性
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		isok := wallet.walletStore.VerifyPasswordHash(WalletUnLock.Passwd)
		if !isok {
			walletlog.Error("ProcWalletUnLock Verify Oldpasswd fail!")
			return types.ErrVerifyOldpasswdFail
		}
	}

	//内存中已经记录password时的校验
	if len(wallet.Password) != 0 && WalletUnLock.Passwd != wallet.Password {
		return types.ErrInputPassword
	}
	//本钱包没有设置密码加密过,只需要解锁不需要记录解锁密码
	wallet.Password = WalletUnLock.Passwd

	//walletlog.Error("ProcWalletUnLock !", "WalletOrTicket", WalletUnLock.WalletOrTicket)

	//只解锁挖矿转账
	if WalletUnLock.WalletOrTicket {
		//wallet.isTicketLocked = false
		atomic.CompareAndSwapInt32(&wallet.isTicketLocked, 1, 0)
	} else {
		//wallet.isWalletLocked = false
		atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 1, 0)
	}
	if WalletUnLock.Timeout != 0 {
		wallet.resetTimeout(WalletUnLock.WalletOrTicket, WalletUnLock.Timeout)
	}
	return nil

}

//解锁超时处理，需要区分整个钱包的解锁或者只挖矿的解锁
func (wallet *Wallet) resetTimeout(IsTicket bool, Timeout int64) {
	//只挖矿的解锁超时
	if IsTicket {
		if wallet.minertimeout == nil {
			wallet.minertimeout = time.AfterFunc(time.Second*time.Duration(Timeout), func() {
				//wallet.isTicketLocked = true
				atomic.CompareAndSwapInt32(&wallet.isTicketLocked, 0, 1)
			})
		} else {
			wallet.minertimeout.Reset(time.Second * time.Duration(Timeout))
		}
	} else { //整个钱包的解锁超时
		if wallet.timeout == nil {
			wallet.timeout = time.AfterFunc(time.Second*time.Duration(Timeout), func() {
				//wallet.isWalletLocked = true
				atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, 1)
			})
		} else {
			wallet.timeout.Reset(time.Second * time.Duration(Timeout))
		}
	}
}

//wallet模块收到blockchain广播的addblock消息，需要解析钱包相关的tx并存储到db中
func (wallet *Wallet) ProcWalletAddBlock(block *types.BlockDetail) {
	if block == nil {
		walletlog.Error("ProcWalletAddBlock input para is nil!")
		return
	}
	//walletlog.Error("ProcWalletAddBlock", "height", block.GetBlock().GetHeight())
	txlen := len(block.Block.GetTxs())
	newbatch := wallet.walletStore.NewBatch(true)
	for index := 0; index < txlen; index++ {
		tx := block.Block.Txs[index]

		//check whether the privacy tx belong to current wallet
		if types.PrivacyX != string(tx.Execer) {
			//获取from地址
			pubkey := block.Block.Txs[index].Signature.GetPubkey()
			addr := account.PubKeyToAddress(pubkey)

			//from addr
			fromaddress := addr.String()
			if len(fromaddress) != 0 && wallet.AddrInWallet(fromaddress) {
				wallet.buildAndStoreWalletTxDetail(tx, block, index, newbatch, fromaddress, false, AddTx)
				walletlog.Debug("ProcWalletAddBlock", "fromaddress", fromaddress)
				continue
			}
			//toaddr
			toaddr := tx.GetTo()
			if len(toaddr) != 0 && wallet.AddrInWallet(toaddr) {
				wallet.buildAndStoreWalletTxDetail(tx, block, index, newbatch, fromaddress, false, AddTx)
				walletlog.Debug("ProcWalletAddBlock", "toaddr", toaddr)
				continue
			}
		} else {
			//TODO:当前不会出现扣掉交易费，而实际的交易不执行的情况，因为如果交易费得不到保障，交易将不被执行
			//确认隐私交易是否是ExecOk
			wallet.AddDelPrivacyTxsFromBlock(tx, int32(index), block, newbatch, AddTx)
		}
	}
	newbatch.Write()
}

func (wallet *Wallet) buildAndStoreWalletTxDetail(tx *types.Transaction, block *types.BlockDetail, index int, newbatch dbm.Batch, from string, isprivacy bool, addDelType int32) {
	blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
	heightstr := fmt.Sprintf("%018d", blockheight)
	if AddTx == addDelType {
		var txdetail types.WalletTxDetail
		txdetail.Tx = tx
		txdetail.Height = block.Block.Height
		txdetail.Index = int64(index)
		txdetail.Receipt = block.Receipts[index]
		txdetail.Blocktime = block.Block.BlockTime

		txdetail.ActionName = txdetail.Tx.ActionName()
		txdetail.Amount, _ = tx.Amount()
		txdetail.Fromaddr = from

		txdetailbyte, err := proto.Marshal(&txdetail)
		if err != nil {
			storelog.Error("ProcWalletAddBlock Marshal txdetail err", "Height", block.Block.Height, "index", index)
			return
		}

		newbatch.Set(calcTxKey(heightstr), txdetailbyte)
		//if isprivacy {
		//	newbatch.Set([]byte(calcRecvPrivacyTxKey(heightstr)), txdetailbyte)
		//}
	} else {
		newbatch.Delete(calcTxKey(heightstr))
		//if isprivacy {
		//	newbatch.Delete([]byte(calcRecvPrivacyTxKey(heightstr)))
		//}
	}

	walletlog.Debug("buildAndStoreWalletTxDetail", "heightstr", heightstr, "addDelType", addDelType)
}

func (wallet *Wallet) needFlushTicket(tx *types.Transaction, receipt *types.ReceiptData) bool {
	if receipt.Ty != types.ExecOk || string(tx.Execer) != "ticket" {
		return false
	}
	pubkey := tx.Signature.GetPubkey()
	addr := account.PubKeyToAddress(pubkey)
	return wallet.AddrInWallet(addr.String())
}

//wallet模块收到blockchain广播的delblock消息，需要解析钱包相关的tx并存db中删除
func (wallet *Wallet) ProcWalletDelBlock(block *types.BlockDetail) {
	if block == nil {
		walletlog.Error("ProcWalletDelBlock input para is nil!")
		return
	}
	//walletlog.Error("ProcWalletDelBlock", "height", block.GetBlock().GetHeight())

	txlen := len(block.Block.GetTxs())
	newbatch := wallet.walletStore.NewBatch(true)
	needflush := false
	for index := txlen - 1; index >= 0; index-- {
		blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
		heightstr := fmt.Sprintf("%018d", blockheight)
		tx := block.Block.Txs[index]
		if "ticket" == string(tx.Execer) {
			receipt := block.Receipts[index]
			if wallet.needFlushTicket(tx, receipt) {
				needflush = true
			}
		}

		if types.PrivacyX != string(tx.Execer) {
			//获取from地址
			pubkey := tx.Signature.GetPubkey()
			addr := account.PubKeyToAddress(pubkey)
			fromaddress := addr.String()
			if len(fromaddress) != 0 && wallet.AddrInWallet(fromaddress) {
				newbatch.Delete(calcTxKey(heightstr))
				continue
			}
			//toaddr
			toaddr := tx.GetTo()
			if len(toaddr) != 0 && wallet.AddrInWallet(toaddr) {
				newbatch.Delete(calcTxKey(heightstr))
			}
		} else {
			wallet.AddDelPrivacyTxsFromBlock(tx, int32(index), block, newbatch, DelTx)
		}
	}
	newbatch.Write()
	if needflush {
		wallet.flushTicket()
	}
}

func (wallet *Wallet) AddDelPrivacyTxsFromBlock(tx *types.Transaction, index int32, block *types.BlockDetail, newbatch dbm.Batch, addDelType int32) {
	amount, err := tx.Amount()
	if err != nil {
		walletlog.Error("AddDelPrivacyTxsFromBlock failed to tx.Amount()")
		return
	}

	txExecRes := block.Receipts[index].Ty

	txhashInbytes := tx.Hash()
	txhash := common.Bytes2Hex(txhashInbytes)
	var privateAction types.PrivacyAction
	if err := types.Decode(tx.GetPayload(), &privateAction); err != nil {
		walletlog.Error("AddDelPrivacyTxsFromBlock failed to decode payload")
	}
	var RpubKey []byte
	var privacyOutput *types.PrivacyOutput
	//var privacyInput *types.PrivacyInput
	var tokenname string
	if types.ActionPublic2Privacy == privateAction.Ty {
		RpubKey = privateAction.GetPublic2Privacy().GetOutput().GetRpubKeytx()
		privacyOutput = privateAction.GetPublic2Privacy().GetOutput()
		tokenname = privateAction.GetPublic2Privacy().GetTokenname()
	} else if types.ActionPrivacy2Privacy == privateAction.Ty {
		RpubKey = privateAction.GetPrivacy2Privacy().GetOutput().GetRpubKeytx()
		privacyOutput = privateAction.GetPrivacy2Privacy().GetOutput()
		tokenname = privateAction.GetPrivacy2Privacy().GetTokenname()
		//privacyInput = privateAction.GetPrivacy2Privacy().GetInput()
	} else if types.ActionPrivacy2Public == privateAction.Ty {
		RpubKey = privateAction.GetPrivacy2Public().GetOutput().GetRpubKeytx()
		privacyOutput = privateAction.GetPrivacy2Public().GetOutput()
		tokenname = privateAction.GetPrivacy2Public().GetTokenname()
		//privacyInput = privateAction.GetPrivacy2Public().GetInput()
	}

	totalUtxosLeft := len(privacyOutput.Keyoutput)
	if privacyInfo, err := wallet.getPrivacyKeyPairsOfWallet(); err == nil {
		matchedCount := 0
		utxoProcessed := make([]bool, len(privacyOutput.Keyoutput))
		for _, info := range privacyInfo {
			walletlog.Debug("AddDelPrivacyTxsFromBlock", "individual privacyInfo's addr", *info.Addr)
			privacykeyParirs := info.PrivacyKeyPair
			walletlog.Debug("AddDelPrivacyTxsFromBlock", "individual ViewPubkey", common.Bytes2Hex(privacykeyParirs.ViewPubkey.Bytes()),
				"individual SpendPubkey", common.Bytes2Hex(privacykeyParirs.SpendPubkey.Bytes()))
			matched4addr := false
			for indexoutput, output := range privacyOutput.Keyoutput {
				if utxoProcessed[indexoutput] {
					continue
				}
				priv, err := privacy.RecoverOnetimePriKey(RpubKey, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(indexoutput))
				if err == nil {
					recoverPub := priv.PubKey().Bytes()[:]
					if bytes.Equal(recoverPub, output.Onetimepubkey) {
						//为了避免匹配成功之后不必要的验证计算，需要统计匹配次数
						//因为目前只会往一个隐私账户转账，
						//1.一般情况下，只会匹配一次，如果是往其他钱包账户转账，
						//2.但是如果是往本钱包的其他地址转账，因为可能存在的change，会匹配2次
						matched4addr = true
						totalUtxosLeft--
						utxoProcessed[indexoutput] = true
						walletlog.Debug("AddDelPrivacyTxsFromBlock got privacy tx belong to current wallet",
							"Address", *info.Addr, "tx with hash", txhash, "Amount", amount)
						//只有当该交易执行成功才进行相应的UTXO的处理
						if types.ExecOk == txExecRes {
							if AddTx == addDelType {
								info2store := &types.PrivacyDBStore{
									Txhash:           txhashInbytes,
									Tokenname:        tokenname,
									Amount:           output.Amount,
									OutIndex:         int32(indexoutput),
									TxPublicKeyR:     RpubKey,
									OnetimePublicKey: output.Onetimepubkey,
									Owner:            *info.Addr,
									Height:           block.Block.Height,
									Txindex:          index,
									Blockhash:        block.Block.Hash(),
								}
								wallet.walletStore.setUTXO(info.Addr, &txhash, indexoutput, info2store, newbatch)
							} else {
								wallet.walletStore.unsetUTXO(info.Addr, &txhash, indexoutput, tokenname, newbatch)
							}
						} else {
							//对于执行失败的交易，只需要将该交易记录在钱包就行
							break
						}

					}
				}
			}
			if true == matched4addr {
				matchedCount++
				//匹配次数达到2次，不再对本钱包中的其他地址进行匹配尝试
				walletlog.Debug("AddDelPrivacyTxsFromBlock", "Get matched privacy transfer for address", *info.Addr,
					"totalUtxosLeft", totalUtxosLeft)

				wallet.buildAndStoreWalletTxDetail(tx, block, int(index), newbatch, *info.Addr, true, addDelType)
				if 2 == matchedCount || 0 == totalUtxosLeft || types.ExecOk != txExecRes {
					walletlog.Debug("AddDelPrivacyTxsFromBlock", "Get matched privacy transfer for address", *info.Addr,
						"matchedCount", matchedCount)
					break
				}
			}
		}
	}

	//如果该隐私交易是本钱包中的地址发送出去的，则需要对相应的utxo进行处理
	if AddTx == addDelType {
		ftxosInOneTx, _, err := wallet.walletStore.GetWalletFTXO()
		if err == nil {
			for _, ftxo := range ftxosInOneTx {
				if ftxo.Txhash == txhash {
					if types.ExecOk == txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
						wallet.walletStore.moveFTXO2STXO(txhash, newbatch)
					} else if types.ExecOk != txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
						//如果执行失败
						wallet.walletStore.unmoveUTXO2FTXO(tokenname, ftxo.Sender, txhash, newbatch)
					}
					//该交易正常执行完毕，删除对其的关注
					wallet.buildAndStoreWalletTxDetail(tx, block, int(index), newbatch,  ftxo.Sender, true, addDelType)
				}
			}
		}
	} else {
		//TODO: 区块回撤的问题，还需要仔细梳理逻辑处理, added by hezhengjun
		blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
		heightstr := fmt.Sprintf("%018d", blockheight)
		value, err := wallet.walletStore.db.Get((calcTxKey(heightstr)))
		if err == nil && nil != value {
			var txdetail types.WalletTxDetail
			err := types.Decode(value, &txdetail)
			if err != nil {
				walletlog.Debug("AddDelPrivacyTxsFromBlock failed to decode value for WalletTxDetail")
			}

			if types.ExecOk == txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
				wallet.walletStore.unmoveFTXO2STXO(txhash, newbatch)
				wallet.buildAndStoreWalletTxDetail(tx, block, int(index), newbatch, "", true, addDelType)
			} else if types.ExecOk != txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
				wallet.buildAndStoreWalletTxDetail(tx, block, int(index), newbatch, "", true, addDelType)
			}
		}
	}
}

//地址对应的账户是否属于本钱包
func (wallet *Wallet) AddrInWallet(addr string) bool {
	if len(addr) == 0 {
		return false
	}
	acc, err := wallet.walletStore.GetAccountByAddr(addr)
	if err == nil && acc != nil {
		return true
	}
	return false
}

func (wallet *Wallet) GetTxDetailByHashs(ReqHashes *types.ReqHashes) {
	//通过txhashs获取对应的txdetail
	msg := wallet.client.NewMessage("blockchain", types.EventGetTransactionByHash, ReqHashes)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		walletlog.Error("GetTxDetailByHashs EventGetTransactionByHash", "err", err)
		return
	}
	TxDetails := resp.GetData().(*types.TransactionDetails)
	if TxDetails == nil {
		walletlog.Info("GetTxDetailByHashs TransactionDetails is nil")
		return
	}

	//批量存储地址对应的所有交易的详细信息到wallet db中
	newbatch := wallet.walletStore.NewBatch(true)
	for _, txdetal := range TxDetails.Txs {
		height := txdetal.GetHeight()
		txindex := txdetal.GetIndex()

		blockheight := height*maxTxNumPerBlock + txindex
		heightstr := fmt.Sprintf("%018d", blockheight)
		var txdetail types.WalletTxDetail
		txdetail.Tx = txdetal.GetTx()
		txdetail.Height = txdetal.GetHeight()
		txdetail.Index = txdetal.GetIndex()
		txdetail.Receipt = txdetal.GetReceipt()
		txdetail.Blocktime = txdetal.GetBlocktime()
		txdetail.Amount = txdetal.GetAmount()
		txdetail.Fromaddr = txdetal.GetFromaddr()
		txdetail.ActionName = txdetal.GetTx().ActionName()

		txdetailbyte, err := proto.Marshal(&txdetail)
		if err != nil {
			storelog.Error("GetTxDetailByHashs Marshal txdetail err", "Height", height, "index", txindex)
			return
		}
		newbatch.Set(calcTxKey(heightstr), txdetailbyte)
		walletlog.Debug("GetTxDetailByHashs", "heightstr", heightstr, "txdetail", txdetail.String())
	}
	newbatch.Write()
}

//从blockchain模块同步addr对应隐私地址作为隐私交易接收方的交易
func (wallet *Wallet) ReqPrivacyTxByAddr(addr string) {
	defer wallet.wg.Done()
	if len(addr) == 0 {
		walletlog.Error("ReqPrivacyTxByAddr input addr is nil!")
		return
	}
	var txInfo types.ReplyTxInfo

	//privacyKeyPair, err := wallet.getPrivacykeyPair(addr)
	_, err := wallet.getPrivacykeyPair(addr)
	if err != nil {
		walletlog.Error("ReqPrivacyTxByAddr failed to getPrivacykeyPair!", "address is", addr)
		return
	}

	i := 0
	for {
		//首先从blockchain模块获取地址对应的所有交易hashs列表,从最新的交易开始获取
		var ReqPrivacy types.ReqPrivacy
		ReqPrivacy.Direction = 0
		ReqPrivacy.Count = int32(MaxTxHashsPerTime)
		if i == 0 {
			ReqPrivacy.Height = -1
		} else {
			ReqPrivacy.Height = txInfo.GetHeight()
		}
		i++
		msg := wallet.client.NewMessage("blockchain", types.EventGetPrivacyTransaction, &ReqPrivacy)
		wallet.client.Send(msg, true)
		resp, err := wallet.client.Wait(msg)
		if err != nil {
			walletlog.Error("ReqTxInfosByAddr EventGetTransactionByAddr", "err", err, "addr", addr)
			return
		}

		ReplyTxInfos := resp.GetData().(*types.ReplyTxInfos)
		if ReplyTxInfos == nil {
			walletlog.Info("ReqTxInfosByAddr ReplyTxInfos is nil")
			return
		}
		txcount := len(ReplyTxInfos.TxInfos)

		var ReqHashes types.ReqHashes
		ReqHashes.Hashes = make([][]byte, len(ReplyTxInfos.TxInfos))
		for index, ReplyTxInfo := range ReplyTxInfos.TxInfos {
			ReqHashes.Hashes[index] = ReplyTxInfo.GetHash()
		}
		// TODO: 隐私交易 导入账户时,需要讲该用户相关的交易扫描出来保存
		//wallet.GetPrivacyTxDetailByHashs(&ReqHashes, privacyKeyPair, &addr)
		if txcount < int(MaxTxHashsPerTime) {
			return
		}

		index := len(ReplyTxInfos.TxInfos) - 1
		txInfo.Hash = ReplyTxInfos.TxInfos[index].GetHash()
		txInfo.Height = ReplyTxInfos.TxInfos[index].GetHeight()
		txInfo.Index = ReplyTxInfos.TxInfos[index].GetIndex()
	}
}

//从blockchain模块同步addr参与的所有交易详细信息
func (wallet *Wallet) ReqTxDetailByAddr(addr string) {
	defer wallet.wg.Done()
	if len(addr) == 0 {
		walletlog.Error("ReqTxInfosByAddr input addr is nil!")
		return
	}
	var txInfo types.ReplyTxInfo

	i := 0
	for {
		//首先从blockchain模块获取地址对应的所有交易hashs列表,从最新的交易开始获取
		var ReqAddr types.ReqAddr
		ReqAddr.Addr = addr
		ReqAddr.Flag = 0
		ReqAddr.Direction = 0
		ReqAddr.Count = int32(MaxTxHashsPerTime)
		if i == 0 {
			ReqAddr.Height = -1
			ReqAddr.Index = 0
		} else {
			ReqAddr.Height = txInfo.GetHeight()
			ReqAddr.Index = txInfo.GetIndex()
		}
		i++
		msg := wallet.client.NewMessage("blockchain", types.EventGetTransactionByAddr, &ReqAddr)
		wallet.client.Send(msg, true)
		resp, err := wallet.client.Wait(msg)
		if err != nil {
			walletlog.Error("ReqTxInfosByAddr EventGetTransactionByAddr", "err", err, "addr", addr)
			return
		}

		ReplyTxInfos := resp.GetData().(*types.ReplyTxInfos)
		if ReplyTxInfos == nil {
			walletlog.Info("ReqTxInfosByAddr ReplyTxInfos is nil")
			return
		}
		txcount := len(ReplyTxInfos.TxInfos)

		var ReqHashes types.ReqHashes
		ReqHashes.Hashes = make([][]byte, len(ReplyTxInfos.TxInfos))
		for index, ReplyTxInfo := range ReplyTxInfos.TxInfos {
			ReqHashes.Hashes[index] = ReplyTxInfo.GetHash()
			txInfo.Hash = ReplyTxInfo.GetHash()
			txInfo.Height = ReplyTxInfo.GetHeight()
			txInfo.Index = ReplyTxInfo.GetIndex()
		}
		wallet.GetTxDetailByHashs(&ReqHashes)
		if txcount < int(MaxTxHashsPerTime) {
			return
		}
	}
}

//使用钱包的password对私钥进行aes cbc加密,返回加密后的privkey
func CBCEncrypterPrivkey(password []byte, privkey []byte) []byte {
	key := make([]byte, 32)
	Encrypted := make([]byte, len(privkey))
	if len(password) > 32 {
		key = password[0:32]
	} else {
		copy(key, password)
	}

	block, _ := aes.NewCipher(key)
	iv := key[:block.BlockSize()]
	//walletlog.Info("CBCEncrypterPrivkey", "password", string(key), "Privkey", common.ToHex(privkey))

	encrypter := cipher.NewCBCEncrypter(block, iv)
	encrypter.CryptBlocks(Encrypted, privkey)

	//walletlog.Info("CBCEncrypterPrivkey", "Encrypted", common.ToHex(Encrypted))
	return Encrypted
}

//使用钱包的password对私钥进行aes cbc解密,返回解密后的privkey
func CBCDecrypterPrivkey(password []byte, privkey []byte) []byte {
	key := make([]byte, 32)
	if len(password) > 32 {
		key = password[0:32]
	} else {
		copy(key, password)
	}

	block, _ := aes.NewCipher(key)
	iv := key[:block.BlockSize()]
	decryptered := make([]byte, len(privkey))
	decrypter := cipher.NewCBCDecrypter(block, iv)
	decrypter.CryptBlocks(decryptered, privkey)
	//walletlog.Info("CBCDecrypterPrivkey", "password", string(key), "Encrypted", common.ToHex(privkey), "decryptered", common.ToHex(decryptered))
	return decryptered
}

//生成一个随机的seed种子, 目前支持英文单词和简体中文
func (wallet *Wallet) genSeed(lang int32) (*types.ReplySeed, error) {
	seed, err := CreateSeed("", lang)
	if err != nil {
		walletlog.Error("genSeed", "CreateSeed err", err)
		return nil, err
	}
	var ReplySeed types.ReplySeed
	ReplySeed.Seed = seed
	return &ReplySeed, nil
}

//获取seed种子, 通过钱包密码
func (wallet *Wallet) getSeed(password string) (string, error) {
	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return "", err
	}

	seed, err := GetSeed(wallet.walletStore.db, password)
	if err != nil {
		walletlog.Error("getSeed", "GetSeed err", err)
		return "", err
	}
	return seed, nil
}

//保存seed种子到数据库中, 并通过钱包密码加密, 钱包起来首先要设置seed
func (wallet *Wallet) saveSeed(password string, seed string) (bool, error) {

	//首先需要判断钱包是否已经设置seed，如果已经设置提示不需要再设置，一个钱包只能保存一个seed
	exit, err := HasSeed(wallet.walletStore.db)
	if exit {
		return false, types.ErrSeedExist
	}
	//入参数校验，seed必须是大于等于12个单词或者汉字
	if len(password) == 0 || len(seed) == 0 {
		return false, types.ErrInputPara
	}

	seedarry := strings.Fields(seed)
	curseedlen := len(seedarry)
	if curseedlen < SaveSeedLong {
		walletlog.Error("saveSeed VeriySeedwordnum", "curseedlen", curseedlen, "SaveSeedLong", SaveSeedLong)
		return false, types.ErrSeedWordNum
	}

	var newseed string
	for index, seedstr := range seedarry {
		if index != curseedlen-1 {
			newseed += seedstr + " "
		} else {
			newseed += seedstr
		}
	}

	//校验seed是否能生成钱包结构类型，从而来校验seed的正确性
	have, err := VerifySeed(newseed)
	if !have {
		walletlog.Error("saveSeed VerifySeed", "err", err)
		return false, types.ErrSeedWord
	}

	ok, err := SaveSeed(wallet.walletStore.db, newseed, password)
	//seed保存成功需要更新钱包密码
	if ok {
		var ReqWalletSetPasswd types.ReqWalletSetPasswd
		ReqWalletSetPasswd.Oldpass = password
		ReqWalletSetPasswd.Newpass = password
		Err := wallet.ProcWalletSetPasswd(&ReqWalletSetPasswd)
		if Err != nil {
			walletlog.Error("saveSeed", "ProcWalletSetPasswd err", err)
		}
	}
	return ok, err
}

//钱包状态检测函数,解锁状态，seed是否已保存
func (wallet *Wallet) CheckWalletStatus() (bool, error) {
	// 钱包锁定，ticket已经解锁，返回只解锁了ticket的错误
	if wallet.IsWalletLocked() && !wallet.IsTicketLocked() {
		return false, types.ErrOnlyTicketUnLocked
	} else if wallet.IsWalletLocked() {
		return false, types.ErrWalletIsLocked
	}

	//判断钱包是否已保存seed
	has, _ := HasSeed(wallet.walletStore.db)
	if !has {
		return false, types.ErrSaveSeedFirst
	}
	return true, nil
}

func (wallet *Wallet) GetWalletStatus() *types.WalletStatus {
	s := &types.WalletStatus{}
	s.IsWalletLock = wallet.IsWalletLocked()
	s.IsHasSeed, _ = HasSeed(wallet.walletStore.db)
	s.IsAutoMining = wallet.isAutoMining()
	s.IsTicketLock = wallet.IsTicketLocked()
	return s
}

//获取地址对应的私钥
func (wallet *Wallet) ProcDumpPrivkey(addr string) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return "", err
	}
	if len(addr) == 0 {
		walletlog.Error("ProcDumpPrivkey input para is nil!")
		return "", types.ErrInputPara
	}

	priv, err := wallet.getPrivKeyByAddr(addr)
	if err != nil {
		return "", err
	}
	return common.ToHex(priv.Bytes()), nil
	//return strings.ToUpper(common.ToHex(priv.Bytes())), nil
}

//检测钱包是否允许转账到指定地址，判断钱包锁和是否有seed以及挖矿锁
func (wallet *Wallet) IsTransfer(addr string) (bool, error) {

	ok, err := wallet.CheckWalletStatus()
	//钱包已经解锁或者错误是ErrSaveSeedFirst直接返回
	if ok || err == types.ErrSaveSeedFirst {
		return ok, err
	}
	//钱包已经锁定，挖矿锁已经解锁,需要判断addr是否是挖矿合约地址
	if !wallet.IsTicketLocked() {
		if addr == account.ExecAddress("ticket") {
			return true, nil
		}
	}
	return ok, err
}

//收到其他模块上报的系统有致命性故障，需要通知前端
func (wallet *Wallet) setFatalFailure(reportErrEvent *types.ReportErrEvent) {

	walletlog.Error("setFatalFailure", "reportErrEvent", reportErrEvent.String())
	if reportErrEvent.Error == "ErrDataBaseDamage" {
		atomic.StoreInt32(&wallet.fatalFailureFlag, 1)
	}
}

func (wallet *Wallet) getFatalFailure() int32 {
	return atomic.LoadInt32(&wallet.fatalFailureFlag)
}

func (wallet *Wallet) getPrivacykeyPair(addr string) (*privacy.Privacy, error) {
	if accPrivacy, _ := wallet.walletStore.GetWalletAccountPrivacy(addr); accPrivacy != nil {
		privacyInfo := &privacy.Privacy{}
		copy(privacyInfo.ViewPubkey[:], accPrivacy.ViewPubkey)
		decrypteredView := CBCDecrypterPrivkey([]byte(wallet.Password), accPrivacy.ViewPrivKey)
		copy(privacyInfo.ViewPrivKey[:], decrypteredView)
		copy(privacyInfo.SpendPubkey[:], accPrivacy.SpendPubkey)
		decrypteredSpend := CBCDecrypterPrivkey([]byte(wallet.Password), accPrivacy.SpendPrivKey)
		copy(privacyInfo.SpendPrivKey[:], decrypteredSpend)

		return privacyInfo, nil
	}
	priv, err := wallet.getPrivKeyByAddr(addr)
	if err != nil {
		return nil, err
	}

	newPrivacy, err := privacy.NewPrivacyWithPrivKey((*[privacy.KeyLen32]byte)(unsafe.Pointer(&priv.Bytes()[0])))
	if err != nil {
		return nil, err
	}

	encrypteredView := CBCEncrypterPrivkey([]byte(wallet.Password), newPrivacy.ViewPrivKey.Bytes())
	encrypteredSpend := CBCEncrypterPrivkey([]byte(wallet.Password), newPrivacy.SpendPrivKey.Bytes())
	walletPrivacy := &types.WalletAccountPrivacy{
		ViewPubkey:   newPrivacy.ViewPubkey[:],
		ViewPrivKey:  encrypteredView,
		SpendPubkey:  newPrivacy.SpendPubkey[:],
		SpendPrivKey: encrypteredSpend,
	}
	//save the privacy created to wallet db
	wallet.walletStore.SetWalletAccountPrivacy(addr, walletPrivacy)
	return newPrivacy, nil
}

func (wallet *Wallet) showPrivacyBalance(req *types.ReqPrivBal4AddrToken) (*types.Account, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	accRes := &types.Account{
		Balance: 0,
	}
	addr := req.GetAddr()
	token := req.GetToken()
	privacyDBStore, err := wallet.walletStore.listAvailableUTXOs(token, addr)
	if err != nil {
		return accRes, err
	}

	if 0 == len(privacyDBStore) {
		return accRes, nil
	}

	balance := int64(0)
	for _, ele := range privacyDBStore {
		balance += ele.Amount
	}

	FTXOsInOneTx, _, err := wallet.walletStore.GetWalletFTXO()
	if err != nil {
		accRes.Balance = balance
		return accRes, nil
	}
	for _, ftxo := range FTXOsInOneTx {
		if ftxo.Sender != addr {
			continue
		}

		for _, utxo := range ftxo.Utxos {
			balance += utxo.Amount
		}
	}

	accRes.Balance = balance
	return accRes, nil
}

func (wallet *Wallet) showPrivacyAccounts(req *types.ReqPrivBal4AddrToken) (*types.UTXOs, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	nilaccRes := make([]*types.UTXO, 0)
	addr := req.GetAddr()
	token := req.GetToken()
	privacyDBStore, err := wallet.walletStore.listAvailableUTXOs(token, addr)
	if err != nil {
		return &types.UTXOs{Utxos: nilaccRes}, err
	}

	if 0 == len(privacyDBStore) {
		return &types.UTXOs{Utxos: nilaccRes}, nil
	}

	accRes := make([]*types.UTXO, len(privacyDBStore))
	for index, ele := range privacyDBStore {
		utxoBasic := &types.UTXOBasic{
			UtxoGlobalIndex: &types.UTXOGlobalIndex{
				Height:   ele.Height,
				Txindex:  ele.Txindex,
				Outindex: ele.OutIndex,
				Txhash:   ele.Txhash,
			},
			OnetimePubkey: ele.OnetimePublicKey,
		}
		utxo := &types.UTXO{
			Amount:    ele.Amount,
			UtxoBasic: utxoBasic,
		}
		accRes[index] = utxo
	}

	return &types.UTXOs{Utxos: nilaccRes}, nil
}

func (wallet *Wallet) showPrivacyAccountsSpend(req *types.ReqPrivBal4AddrToken) ([]*types.UTXOHaveTxHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	addr := req.GetAddr()
	token := req.GetToken()
	utxoHaveTxHash, err := wallet.walletStore.listSpendUTXOs(token, addr)
	if err != nil {
		return nil, err
	}

	if 0 == len(utxoHaveTxHash) {
		return nil, nil
	}

	return utxoHaveTxHash, nil
}

func makeViewSpendPubKeyPairToString(viewPubKey, spendPubKey []byte) string {
	pair := viewPubKey
	pair = append(pair, spendPubKey...)
	return common.Bytes2Hex(pair)
}

func (wallet *Wallet) showPrivacyPkPair(reqAddr *types.ReqStr) (*types.ReplyPrivacyPkPair, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	privacyInfo, err := wallet.getPrivacykeyPair(reqAddr.GetReqStr())
	if err != nil {
		return nil, err
	}

	pair := privacyInfo.ViewPubkey[:]
	pair = append(pair, privacyInfo.SpendPubkey[:]...)

	replyPrivacyPkPair := &types.ReplyPrivacyPkPair{
		ShowSuccessful: true,
		Pubkeypair:     makeViewSpendPubKeyPairToString(privacyInfo.ViewPubkey[:], privacyInfo.SpendPubkey[:]),
	}

	return replyPrivacyPkPair, nil
}

func (wallet *Wallet) getPrivacyKeyPairsOfWallet() ([]addrAndprivacy, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	//通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Info("getPrivacyKeyPairsOfWallet", "GetAccountByPrefix:err", err)
		return nil, err
	}

	infoPriRes := make([]addrAndprivacy, len(WalletAccStores))
	for index, AccStore := range WalletAccStores {
		if len(AccStore.Addr) != 0 {
			if privacyInfo, err := wallet.getPrivacykeyPair(AccStore.Addr); err == nil {
				var priInfo addrAndprivacy
				priInfo.Addr = &AccStore.Addr
				priInfo.PrivacyKeyPair = privacyInfo
				infoPriRes[index] = priInfo
			}
		}
	}
	return infoPriRes, nil
}

func (w *Wallet) getActionMainInfo(action *types.PrivacyAction) (rpubkey []byte, privOutput *types.PrivacyOutput, tokenname string, err error) {
	if types.ActionPublic2Privacy == action.Ty {
		rpubkey = action.GetPublic2Privacy().GetOutput().GetRpubKeytx()
		privOutput = action.GetPublic2Privacy().GetOutput()
		tokenname = action.GetPublic2Privacy().GetTokenname()
	} else if types.ActionPrivacy2Privacy == action.Ty {
		rpubkey = action.GetPrivacy2Privacy().GetOutput().GetRpubKeytx()
		privOutput = action.GetPrivacy2Privacy().GetOutput()
		tokenname = action.GetPrivacy2Privacy().GetTokenname()
	} else if types.ActionPrivacy2Public == action.Ty {
		rpubkey = action.GetPrivacy2Public().GetOutput().GetRpubKeytx()
		privOutput = action.GetPrivacy2Public().GetOutput()
		tokenname = action.GetPrivacy2Public().GetTokenname()
	} else {
		err = errors.New("Do not support action type.")
	}
	return
}
