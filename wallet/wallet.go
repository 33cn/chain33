package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"bytes"
	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"unsafe"
)

var (
	minFee            int64 = types.MinFee
	maxTxNumPerBlock  int64 = types.MaxTxsPerBlock
	MaxTxHashsPerTime int64 = 100
	walletlog               = log.New("module", "wallet")
	SignType          int   = 1 //1；secp256k1，2：ed25519，3：sm2
	accountdb               = account.NewCoinsAccount()
	accTokenMap             = make(map[string]*account.AccountDB)
)

type Wallet struct {
	client         queue.Client
	mtx            sync.Mutex
	timeout        *time.Timer
	minertimeout   *time.Timer
	isclosed       int32
	isWalletLocked bool
	isTicketLocked bool
	lastHeight     int64
	autoMinerFlag  int32
	Password       string
	FeeAmount      int64
	EncryptFlag    int64
	miningTicket   *time.Ticker
	wg             *sync.WaitGroup
	walletStore    *WalletStore
	random         *rand.Rand
	done           chan struct{}
	privacyActive  map[string]map[string]*walletOuts //不同token类型对应的公开地址拥有的隐私存款记录，map【token】map【addr】
	//privacyActive  map[string]*walletOuts //公开地址对应的隐私存款记录
	privacyFrozen map[string]*walletOuts //公开地址对应的被冻结的隐私存款记录，需要确认该笔隐私存款被花费
}

type walletOuts struct {
	outs []*txOutputInfo
}

type txOutputInfo struct {
	// output info
	amount            int64
	//globalOutputIndex int
	//indexInTx         int
	utxoGlobalIndex *types.UTXOGlobalIndex
	// transaction info
	txHash           common.Hash
	txPublicKeyR     privacy.PubKeyPrivacy
	onetimePublicKey privacy.PubKeyPrivacy
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
	walletStoreDB := dbm.NewDB("wallet", cfg.Driver, cfg.DbPath, 16)
	walletStore := NewWalletStore(walletStoreDB)
	minFee = cfg.MinFee
	signType, exist := crypto.MapSignName2Type[cfg.SignType]
	if !exist {
		signType = crypto.SignTypeSecp256k1
	}
	SignType = signType

	wallet := &Wallet{
		walletStore:    walletStore,
		isWalletLocked: true,
		isTicketLocked: true,
		autoMinerFlag:  0,
		wg:             &sync.WaitGroup{},
		FeeAmount:      walletStore.GetFeeAmount(),
		EncryptFlag:    walletStore.GetEncryptionFlag(),
		miningTicket:   time.NewTicker(2 * time.Minute),
		done:           make(chan struct{}),
	}
	value := walletStore.db.Get([]byte("WalletAutoMiner"))
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

func (wallet *Wallet) IsWalletLocked() bool {
	return wallet.isWalletLocked
}

func (wallet *Wallet) SetQueueClient(client queue.Client) {
	wallet.client = client
	wallet.client.Sub("wallet")
	wallet.wg.Add(2)
	go wallet.ProcRecvMsg()
	go wallet.autoMining()
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
			if !wallet.IsCaughtUp() {
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
		walletlog.Debug("wallet recv", "msg", msg)
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
			WalletAccount, err := wallet.ProcCreatNewAccount(NewAccount)
			if err != nil {
				walletlog.Error("ProcCreatNewAccount", "err", err.Error())
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
				walletlog.Error("EventGetSeed", "seed", seed)
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
			privkey, err := wallet.ProcDumpPrivkey(addr.Reqstr)
			if err != nil {
				walletlog.Error("ProcDumpPrivkey", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivkey, err))
			} else {
				var replyStr types.ReplyStr
				replyStr.Replystr = privkey
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyPrivkey, &replyStr))
			}
		case types.EventTokenPreCreate:
			preCreate := msg.Data.(*types.ReqTokenPreCreate)
			reply, err := wallet.procTokenPreCreate(preCreate)
			if err != nil {
				walletlog.Error("procTokenPreCreate", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyTokenPreCreate, err))
			} else {
				walletlog.Info("procTokenPreCreate", "symbol", preCreate.GetSymbol(),
					"txhash", common.ToHex(reply.Hash))
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyTokenPreCreate, reply))
			}
		case types.EventTokenFinishCreate:
			finishCreate := msg.Data.(*types.ReqTokenFinishCreate)
			reply, err := wallet.procTokenFinishCreate(finishCreate)
			if err != nil {
				walletlog.Error("procTokenPreCreate", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyTokenFinishCreate, err))
			} else {
				walletlog.Info("procTokenPreCreate", "symbol", finishCreate.GetSymbol(),
					"txhash", common.ToHex(reply.Hash))
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyTokenFinishCreate, reply))
			}
		case types.EventTokenRevokeCreate:
			revoke := msg.Data.(*types.ReqTokenRevokeCreate)
			reply, err := wallet.procTokenRevokeCreate(revoke)
			if err != nil {
				walletlog.Error("procTokenRevokeCreate", "err", err.Error())

				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyTokenRevokeCreate, err))
			} else {
				walletlog.Info("procTokenRevokeCreate", "symbol", revoke.GetSymbol(),
					"txhash", common.ToHex(reply.Hash))
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyTokenRevokeCreate, reply))
			}
		case types.EventSellToken:
			sellToken := msg.Data.(*types.ReqSellToken)
			replyHash, err := wallet.procSellToken(sellToken)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
				walletlog.Error("procSellToken", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplySellToken, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procSellToken", "tx hash", common.Bytes2Hex(replyHash.Hash), "symbol", sellToken.Sell.Tokensymbol, "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplySellToken, &reply))
			}
		case types.EventBuyToken:
			buyToken := msg.Data.(*types.ReqBuyToken)
			replyHash, err := wallet.procBuyToken(buyToken)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procBuyToken", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyBuyToken, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procBuyToken", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyBuyToken, &reply))
			}
		case types.EventRevokeSellToken:
			revokeSell := msg.Data.(*types.ReqRevokeSell)
			replyHash, err := wallet.procRevokeSell(revokeSell)
			var reply types.Reply
			if err != nil {
				reply.IsOk = false
				walletlog.Error("procRevokeSell", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyRevokeSellToken, err))
			} else {
				reply.IsOk = true
				reply.Msg = replyHash.Hash
				walletlog.Info("procRevokeSell", "tx hash", common.Bytes2Hex(replyHash.Hash), "result", "success")
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyRevokeSellToken, &reply))
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

		case types.EventShowPrivacyAccount:
			reqStr := msg.Data.(*types.ReqStr)
			account, err := wallet.showPrivacyAccounts(reqStr)
			if err != nil {
				walletlog.Error("showPrivacyAccount", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyAccount, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyAccount, account))
			}
		case types.EventShowPrivacyTransfer:
			reqAddrAndHash := msg.Data.(*types.ReqPrivacyBalance)
			account, err := wallet.showPrivacyBalance(reqAddrAndHash)
			if err != nil {
				walletlog.Error("showPrivacyTransfer", "err", err.Error())
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyTransfer, err))
			} else {
				msg.Reply(wallet.client.NewMessage("rpc", types.EventReplyShowPrivacyTransfer, account))
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
			replyHash, err := wallet.procPublic2Privacy(reqPub2Pri)
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
			replyHash, err := wallet.procPrivacy2Privacy(reqPri2Pri)
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
			replyHash, err := wallet.procPrivacy2Public(reqPri2Pub)
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
		default:
			walletlog.Info("ProcRecvMsg unknow msg", "msgtype", msgtype)
		}
		walletlog.Debug("end process")
	}
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
	accounts, err := accountdb.LoadAccounts(wallet.client, addrs)
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
func (wallet *Wallet) ProcCreatNewAccount(Label *types.ReqNewAccount) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}

	if Label == nil || len(Label.GetLabel()) == 0 {
		walletlog.Error("ProcCreatNewAccount Label is nil")
		return nil, types.ErrInputPara
	}

	//首先校验label是否已被使用
	WalletAccStores, err := wallet.walletStore.GetAccountByLabel(Label.GetLabel())
	if WalletAccStores != nil {
		walletlog.Error("ProcCreatNewAccount Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
	}

	var Account types.Account
	var walletAccount types.WalletAccount
	var WalletAccStore types.WalletAccountStore

	//生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(SignType))
	if err != nil {
		walletlog.Error("ProcCreatNewAccount", "err", err)
		return nil, err
	}
	//通过seed获取私钥, 首先通过钱包密码解锁seed然后通过seed生成私钥
	seed, err := wallet.getSeed(wallet.Password)
	if err != nil {
		walletlog.Error("ProcCreatNewAccount", "getSeed err", err)
		return nil, err
	}
	privkeyhex, err := GetPrivkeyBySeed(wallet.walletStore.db, seed)
	if err != nil {
		walletlog.Error("ProcCreatNewAccount", "GetPrivkeyBySeed err", err)
		return nil, err
	}
	privkeybyte, err := common.FromHex(privkeyhex)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("ProcCreatNewAccount", "FromHex err", err)
		return nil, err
	}
	priv, err := cr.PrivKeyFromBytes(privkeybyte)
	if err != nil {
		walletlog.Error("ProcCreatNewAccount", "PrivKeyFromBytes err", err)
		return nil, err
	}
	addr := account.PubKeyToAddress(priv.PubKey().Bytes())
	Account.Addr = addr.String()
	Account.Currency = 0
	Account.Balance = 0
	Account.Frozen = 0

	walletAccount.Acc = &Account
	walletAccount.Label = Label.GetLabel()

	//使用钱包的password对私钥加密 aes cbc
	Encrypted := CBCEncrypterPrivkey([]byte(wallet.Password), priv.Bytes())
	WalletAccStore.Privkey = common.ToHex(Encrypted)
	WalletAccStore.Label = Label.GetLabel()
	WalletAccStore.Addr = addr.String()

	//存储账户信息到wallet数据库中
	err = wallet.walletStore.SetWalletAccount(false, Account.Addr, &WalletAccStore)
	if err != nil {
		return nil, err
	}

	//获取地址对应的账户信息从account模块
	addrs := make([]string, 1)
	addrs[0] = addr.String()
	accounts, err := accountdb.LoadAccounts(wallet.client, addrs)
	if err != nil {
		walletlog.Error("ProcCreatNewAccount", "LoadAccounts err", err)
		return nil, err
	}
	// 本账户是首次创建
	if len(accounts[0].Addr) == 0 {
		accounts[0].Addr = addr.String()
	}
	walletAccount.Acc = accounts[0]

	//从blockchain模块同步Account.Addr对应的所有交易详细信息
	wallet.wg.Add(1)
	go wallet.ReqTxDetailByAddr(addr.String())
	wallet.wg.Add(1)
	go wallet.ReqPrivacyTxByAddr(addr.String())

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

	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(SignType))
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "err", err)
		return nil, types.ErrNewCrypto
	}
	privkeybyte, err := common.FromHex(PrivKey.Privkey)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("ProcImportPrivKey", "FromHex err", err)
		return nil, types.ErrFromHex
	}
	priv, err := cr.PrivKeyFromBytes(privkeybyte)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "PrivKeyFromBytes err", err)
		return nil, types.ErrPrivKeyFromBytes
	}
	addr := account.PubKeyToAddress(priv.PubKey().Bytes())

	//对私钥加密
	Encryptered := CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	Encrypteredstr := common.ToHex(Encryptered)
	//校验PrivKey对应的addr是否已经存在钱包中
	Account, err = wallet.walletStore.GetAccountByAddr(addr.String())
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
	WalletAccStore.Addr = addr.String()
	//存储Addr:label+privkey+addr到数据库
	err = wallet.walletStore.SetWalletAccount(false, addr.String(), &WalletAccStore)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "SetWalletAccount err", err)
		return nil, err
	}

	//获取地址对应的账户信息从account模块
	addrs := make([]string, 1)
	addrs[0] = addr.String()
	accounts, err := accountdb.LoadAccounts(wallet.client, addrs)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "LoadAccounts err", err)
		return nil, err
	}
	// 本账户是首次创建
	if len(accounts[0].Addr) == 0 {
		accounts[0].Addr = addr.String()
	}
	walletaccount.Acc = accounts[0]
	walletaccount.Label = PrivKey.Label

	//从blockchain模块同步Account.Addr对应的所有交易详细信息
	wallet.wg.Add(1)
	go wallet.ReqTxDetailByAddr(addr.String())
	wallet.wg.Add(1)
	go wallet.ReqPrivacyTxByAddr(addr.String())

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
	accounts, err = accountdb.LoadAccounts(wallet.client, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcSendToAddress", "LoadAccounts err", err)
		return nil, err
	}
	Balance := accounts[0].Balance
	amount := SendToAddress.GetAmount()
	if !SendToAddress.Istoken {
		if Balance < amount+wallet.FeeAmount {
			return nil, types.ErrInsufficientBalance
		}
	} else {
		//如果是token转账，一方面需要保证coin的余额满足fee，另一方面则需要保证token的余额满足转账操作
		if Balance < wallet.FeeAmount {
			return nil, types.ErrInsufficientBalance
		}

		if nil == accTokenMap[SendToAddress.TokenSymbol] {
			tokenAccDB := account.NewTokenAccountWithoutDB(SendToAddress.TokenSymbol)
			accTokenMap[SendToAddress.TokenSymbol] = tokenAccDB
		}
		tokenAccDB := accTokenMap[SendToAddress.TokenSymbol]
		tokenAccounts, err = tokenAccDB.LoadAccounts(wallet.client, addrs)
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
	return wallet.sendToAddress(priv, addrto, amount, note, SendToAddress.Istoken, SendToAddress.TokenSymbol)
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
			accounts, err := accountdb.LoadAccounts(wallet.client, addrs)
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
	accounts, err := accountdb.LoadAccounts(wallet.client, addrs)
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
	//ReplyHashes.Hashes = make([][]byte, len(accounts))

	for index, Account := range accounts {
		Privkey := WalletAccStores[index].Privkey
		//解密存储的私钥
		prikeybyte, err := common.FromHex(Privkey)
		if err != nil || len(prikeybyte) == 0 {
			walletlog.Error("ProcMergeBalance", "FromHex err", err)
			return nil, err
		}

		privkey := CBCDecrypterPrivkey([]byte(wallet.Password), prikeybyte)
		priv, err := cr.PrivKeyFromBytes(privkey)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "PrivKeyFromBytes err", err, "index", index)
			//ReplyHashes.Hashes[index] = common.Hash{}.Bytes()
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
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: wallet.FeeAmount, To: addrto, Nonce: r.Int63()}
		tx.Sign(int32(SignType), priv)

		//发送交易信息给mempool模块
		msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
		wallet.client.Send(msg, true)
		_, err = wallet.client.Wait(msg)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "Send tx err", err, "index", index)
			//ReplyHashes.Hashes[index] = common.Hash{}.Bytes()
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
	tempislock := wallet.isWalletLocked
	wallet.isWalletLocked = false
	defer func() {
		wallet.isWalletLocked = tempislock
	}()

	// 钱包已经加密需要验证oldpass的正确性
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		isok := wallet.walletStore.VerifyPasswordHash(Passwd.Oldpass)
		if !isok {
			walletlog.Error("ProcWalletSetPasswd Verify Oldpasswd fail!")
			return types.ErrVerifyOldpasswdFail
		}
	}

	if len(wallet.Password) != 0 && Passwd.Oldpass != wallet.Password {
		walletlog.Error("ProcWalletSetPasswd Oldpass err!")
		return types.ErrVerifyOldpasswdFail
	}

	//使用新的密码生成passwdhash用于下次密码的验证
	err = wallet.walletStore.SetPasswordHash(Passwd.Newpass)
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
	seed, err := wallet.getSeed(Passwd.Oldpass)
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "getSeed err", err)
		return err
	}
	ok, err := SaveSeed(wallet.walletStore.db, seed, Passwd.Newpass)
	if !ok {
		walletlog.Error("ProcWalletSetPasswd", "SaveSeed err", err)
		return err
	}

	//对所有存储的私钥重新使用新的密码加密,通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Error("ProcWalletSetPasswd", "GetAccountByPrefix:err", err)
	}
	if WalletAccStores != nil {
		for _, AccStore := range WalletAccStores {
			//使用old Password解密存储的私钥
			storekey, err := common.FromHex(AccStore.GetPrivkey())
			if err != nil || len(storekey) == 0 {
				walletlog.Error("ProcWalletSetPasswd", "addr", AccStore.Addr, "FromHex err", err)
				continue
			}
			Decrypter := CBCDecrypterPrivkey([]byte(Passwd.Oldpass), storekey)

			//使用新的密码重新加密私钥
			Encrypter := CBCEncrypterPrivkey([]byte(Passwd.Newpass), Decrypter)
			AccStore.Privkey = common.ToHex(Encrypter)
			err = wallet.walletStore.SetWalletAccount(true, AccStore.Addr, AccStore)
			if err != nil {
				walletlog.Error("ProcWalletSetPasswd", "addr", AccStore.Addr, "SetWalletAccount err", err)
			}
		}
	}
	wallet.Password = Passwd.Newpass
	return nil
}

//锁定钱包
func (wallet *Wallet) ProcWalletLock() error {
	//判断钱包是否已保存seed
	has, _ := HasSeed(wallet.walletStore.db)
	if !has {
		return types.ErrSaveSeedFirst
	}

	wallet.isWalletLocked = true
	wallet.isTicketLocked = true
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

	//只解锁挖矿的转账
	if WalletUnLock.WalletOrTicket {
		wallet.isTicketLocked = false
	} else {
		wallet.isWalletLocked = false
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
				wallet.isTicketLocked = true
			})
		} else {
			wallet.minertimeout.Reset(time.Second * time.Duration(Timeout))
		}
	} else { //整个钱包的解锁超时
		if wallet.timeout == nil {
			wallet.timeout = time.AfterFunc(time.Second*time.Duration(Timeout), func() {
				wallet.isWalletLocked = true
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
		//获取Amount
		amount, err := tx.Amount()
		if err != nil {
			continue
		}

		//获取from地址
		pubkey := block.Block.Txs[index].Signature.GetPubkey()
		addr := account.PubKeyToAddress(pubkey)

		//from addr
		fromaddress := addr.String()
		if len(fromaddress) != 0 && wallet.AddrInWallet(fromaddress) {
			wallet.buildAndStoreWalletTxDetail(tx, block, index, newbatch, fromaddress, false)
			walletlog.Debug("ProcWalletAddBlock", "fromaddress", fromaddress)
			continue
		}

		//toaddr
		toaddr := tx.GetTo()
		if len(toaddr) != 0 && wallet.AddrInWallet(toaddr) {
			wallet.buildAndStoreWalletTxDetail(tx, block, index, newbatch, fromaddress, false)
			walletlog.Debug("ProcWalletAddBlock", "toaddr", toaddr)
			continue
		}
		//check whether the privacy tx belong to current wallet
		if types.PrivacyX == string(tx.Execer) {
			var privateAction types.PrivacyAction
			if err := types.Decode(tx.GetPayload(), &privateAction); err != nil {
				walletlog.Error("ProcWalletAddBlock failed to decode payload")
			}
			var RpubKey []byte
			var privacyOutput *types.PrivacyOutput
			var tokenname string
			if types.ActionPublic2Privacy == privateAction.Ty {
				RpubKey = privateAction.GetPublic2Privacy().GetRpubKeytx()
				privacyOutput = privateAction.GetPublic2Privacy().GetOutput()
				tokenname = privateAction.GetPublic2Privacy().GetTokenname()
			} else if types.ActionPrivacy2Privacy == privateAction.Ty {
				RpubKey = privateAction.GetPrivacy2Privacy().GetRpubKeytx()
				privacyOutput = privateAction.GetPrivacy2Privacy().GetOutput()
				tokenname = privateAction.GetPrivacy2Privacy().GetTokenname()
			} else {
				continue
			}

			if privacyInfo, err := wallet.getPrivacyKeyPairsOfWallet(); err == nil {
				matchedCount := 0
				for _, info := range privacyInfo {
					walletlog.Debug("ProcWalletAddBlock", "individual privacyInfo's addr", *info.Addr)
					privacykeyParirs := info.PrivacyKeyPair
					walletlog.Debug("ProcWalletAddBlock", "individual ViewPubkey", common.Bytes2Hex(privacykeyParirs.ViewPubkey.Bytes()),
						"individual SpendPubkey", common.Bytes2Hex(privacykeyParirs.SpendPubkey.Bytes()))
					matched4addr := false
					for indexoutput, output := range privacyOutput.Keyoutput {
						priv, err := privacy.RecoverOnetimePriKey(RpubKey, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(indexoutput))
						if err == nil {
							recoverPub := priv.PubKey().Bytes()[:]
							if bytes.Equal(recoverPub, output.Ometimepubkey) {
								//为了避免匹配成功之后不必要的验证计算，需要统计匹配次数
								//因为目前只会往一个隐私账户转账，
								//一般情况下，只会匹配一次，如果是往其他钱包账户转账，
								//但是如果是往本钱包的其他地址转账，因为可能存在的change，会匹配2次
								matched4addr = true
								txhash := common.ToHex(tx.Hash())
								walletlog.Debug("ProcWalletAddBlock got privacy tx belong to current wallet",
									"Address", *info.Addr, "tx with hash", txhash, "Amount", amount)
								info2store := &types.PrivacyDBStore{
									Txhash:           txhash,
									Tokenname:        tokenname,
									Amount:           output.Amount,
									IndexInTx:        int32(indexoutput),
									TxPublicKeyR:     RpubKey,
									OnetimePublicKey: output.Ometimepubkey,
								}

								txOutInfo := &txOutputInfo{
									amount:            output.Amount,
									globalOutputIndex: 0,
									indexInTx:         indexoutput,
									txHash:            common.BytesToHash(tx.Hash()),
									txPublicKeyR:      privacy.Bytes2PubKeyPrivacy(RpubKey),
									onetimePublicKey:  privacy.Bytes2PubKeyPrivacy(output.Ometimepubkey),
								}

								//首先判断是否存在token对应的walletout
								walletOuts4token, ok := wallet.privacyActive[tokenname]
								if ok {
									walletOuts4addr, ok := walletOuts4token[*info.Addr]
									if ok {
										walletOuts4addr.outs = append(walletOuts4addr.outs, txOutInfo)
									} else {
										var txOutputInfoSlice []*txOutputInfo
										txOutputInfoSlice = append(txOutputInfoSlice, txOutInfo)
										walletOuts := &walletOuts{
											outs: txOutputInfoSlice,
										}
										walletOuts4token[*info.Addr] = walletOuts
									}
								} else {
									var txOutputInfoSlice []*txOutputInfo
									txOutputInfoSlice = append(txOutputInfoSlice, txOutInfo)
									walletOuts := &walletOuts{
										outs: txOutputInfoSlice,
									}

									walletOuts4tokenTemp := make(map[string]*walletOuts, 1)
									walletOuts4tokenTemp[*info.Addr] = walletOuts
									wallet.privacyActive[tokenname] = walletOuts4tokenTemp
								}

								wallet.walletStore.setWalletPrivacyAccountBalance(info.Addr, &txhash, info2store, newbatch, indexoutput)
							}
						}
					}
					if true == matched4addr {
						matchedCount++
						if 2 == matchedCount {
							walletlog.Debug("ProcWalletAddBlock", "Get matched privacy transfer for address", *info.Addr,
								"matchedCount", matchedCount)
							break
						}
					}
				}
				if matchedCount > 0 {
					wallet.buildAndStoreWalletTxDetail(tx, block, index, newbatch, fromaddress, true)
				}
			}
		}
	}
	newbatch.Write()
}

func (wallet *Wallet) buildAndStoreWalletTxDetail(tx *types.Transaction, block *types.BlockDetail, index int, newbatch dbm.Batch, from string, isprivacy bool) {
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

	blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
	heightstr := fmt.Sprintf("%018d", blockheight)
	newbatch.Set([]byte(calcTxKey(heightstr)), txdetailbyte)
	if isprivacy {
		newbatch.Set([]byte(calcRecvPrivacyTxKey(heightstr)), txdetailbyte)
	}
	walletlog.Debug("ProcWalletAddBlock", "heightstr", heightstr)
}

func (wallet *Wallet) needFlushTicket(tx *types.Transaction, receipt *types.ReceiptData) bool {
	if receipt.Ty != types.ExecOk || string(tx.Execer) != "ticket" {
		return false
	}
	pubkey := tx.Signature.GetPubkey()
	addr := account.PubKeyToAddress(pubkey)
	if wallet.AddrInWallet(addr.String()) {
		return true
	}
	return false
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
	for index := 0; index < txlen; index++ {
		blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
		heightstr := fmt.Sprintf("%018d", blockheight)
		if "ticket" == string(block.Block.Txs[index].Execer) {
			tx := block.Block.Txs[index]
			receipt := block.Receipts[index]
			if wallet.needFlushTicket(tx, receipt) {
				needflush = true
			}
		}
		//获取from地址
		pubkey := block.Block.Txs[index].Signature.GetPubkey()
		addr := account.PubKeyToAddress(pubkey)
		fromaddress := addr.String()
		if len(fromaddress) != 0 && wallet.AddrInWallet(fromaddress) {
			newbatch.Delete([]byte(calcTxKey(heightstr)))
			//walletlog.Error("ProcWalletAddBlock", "fromaddress", fromaddress, "heightstr", heightstr)
			continue
		}
		//toaddr
		toaddr := block.Block.Txs[index].GetTo()
		if len(toaddr) != 0 && wallet.AddrInWallet(toaddr) {
			newbatch.Delete([]byte(calcTxKey(heightstr)))
			//walletlog.Error("ProcWalletAddBlock", "toaddr", toaddr, "heightstr", heightstr)
		}
	}
	newbatch.Write()
	if needflush {
		wallet.flushTicket()
	}
}

//地址对应的账户是否属于本钱包
func (wallet *Wallet) AddrInWallet(addr string) bool {
	if len(addr) == 0 {
		return false
	}
	account, err := wallet.walletStore.GetAccountByAddr(addr)
	if err == nil && account != nil {
		return true
	}
	return false
}
func (wallet *Wallet) GetPrivacyTxDetailByHashs(ReqHashes *types.ReqHashes, keyPair *privacy.Privacy, addr *string) {
	//通过txhashs获取对应的txdetail
	msg := wallet.client.NewMessage("blockchain", types.EventGetTransactionByHash, ReqHashes)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		walletlog.Error("GetPrivacyTxDetailByHashs EventGetTransactionByHash", "err", err)
		return
	}
	TxDetails := resp.GetData().(*types.TransactionDetails)
	if TxDetails == nil {
		walletlog.Info("GetPrivacyTxDetailByHashs TransactionDetails is nil")
		return
	}
	newbatch := wallet.walletStore.NewBatch(true)
	for _, txdetal := range TxDetails.Txs {
		if types.PrivacyX != string(txdetal.GetTx().Execer) {
			walletlog.Error("GetPrivacyTxDetailByHashs tx is not with the type of privacy")
			continue
		}

		var privateAction types.PrivacyAction
		if err := types.Decode(txdetal.Tx.Payload, &privateAction); err != nil {
			walletlog.Error("GetPrivacyTxDetailByHashs failed to decode payload")
			return
		}

		var RpubKeytx []byte
		var onetimePubKey []byte
		var tokenname string
		if types.ActionPublic2Privacy == privateAction.Ty {
			RpubKeytx = privateAction.GetPublic2Privacy().GetRpubKeytx()
			onetimePubKey = privateAction.GetPublic2Privacy().GetOnetimePubKey()
			tokenname = privateAction.GetPublic2Privacy().GetTokenname()
		} else if types.ActionPrivacy2Privacy == privateAction.Ty {
			RpubKeytx = privateAction.GetPrivacy2Privacy().GetRpubKeytx()
			onetimePubKey = privateAction.GetPrivacy2Privacy().GetOnetimePubKey()
			tokenname = privateAction.GetPrivacy2Privacy().GetTokenname()
		} else {
			continue
		}

		priv, err := privacy.RecoverOnetimePriKey(RpubKeytx, keyPair.ViewPrivKey, keyPair.SpendPrivKey)
		if err != nil {
			walletlog.Error("GetPrivacyTxDetailByHashs", "Failed to RecoverOnetimePriKey", err)
			return
		}
		recoverPub := priv.PubKey()
		recoverPubByte := recoverPub.Bytes()
		//该地址作为隐私交易的接收地址
		if bytes.Equal(recoverPubByte, onetimePubKey) {
			txhash := common.ToHex(txdetal.Tx.Hash())
			onetimeAddr := account.PubKeyToAddress(recoverPubByte).String()
			info2store := &types.PrivacyDBStore{
				Onetimeaddr: onetimeAddr,
				Txhash:      txhash,
				Tokenname:   tokenname,
			}
			wallet.walletStore.setWalletPrivacyAccountBalance(addr, &txhash, info2store, newbatch)

			//将该隐私交易进行保存
			height := txdetal.GetHeight()
			txindex := txdetal.GetIndex()

			blockheight := height*maxTxNumPerBlock + int64(txindex)
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
			newbatch.Set([]byte(calcTxKey(heightstr)), txdetailbyte)
			newbatch.Set([]byte(calcRecvPrivacyTxKey(heightstr)), txdetailbyte)
		} else {
			//copytx := *txdetal.Tx
			//copytx.Signature = nil
			//data := types.Encode(&copytx)
			//c, err := crypto.New(types.GetSignatureTypeName(crypto.SignTypeOnetimeED25519))
			//if err != nil {
			//	walletlog.Error("GetPrivacyTxDetailByHashs", "Failed to new crypto due to error", err)
			//	continue
			//
			//}
			//signbytes, err := c.SignatureFromBytes(txdetal.Tx.Signature.Signature)
			//if err != nil {
			//	walletlog.Error("GetPrivacyTxDetailByHashs", "Failed to SignatureFromBytes due to error", err)
			//	continue
			//}
			////该地址作为隐私交易的发送地址
			//if recoverPub.VerifyBytes(data, signbytes) {
			//	height := txdetal.GetHeight()
			//	txindex := txdetal.GetIndex()
			//
			//	blockheight := height*maxTxNumPerBlock + int64(txindex)
			//	heightstr := fmt.Sprintf("%018d", blockheight)
			//	var txdetail types.WalletTxDetail
			//	txdetail.Tx = txdetal.GetTx()
			//	txdetail.Height = txdetal.GetHeight()
			//	txdetail.Index = txdetal.GetIndex()
			//	txdetail.Receipt = txdetal.GetReceipt()
			//	txdetail.Blocktime = txdetal.GetBlocktime()
			//	txdetail.Amount = txdetal.GetAmount()
			//	txdetail.Fromaddr = txdetal.GetFromaddr()
			//	txdetail.ActionName = txdetal.GetTx().ActionName()
			//
			//	txdetailbyte, err := proto.Marshal(&txdetail)
			//	if err != nil {
			//		storelog.Error("GetTxDetailByHashs Marshal txdetail err", "Height", height, "index", txindex)
			//		return
			//	}
			//	newbatch.Set([]byte(calcTxKey(heightstr)), txdetailbyte)
			//	walletlog.Debug("GetTxDetailByHashs", "heightstr", heightstr, "txdetail", txdetail.String())
			//}
		}
	}
	newbatch.Write()
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

		blockheight := height*maxTxNumPerBlock + int64(txindex)
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
		newbatch.Set([]byte(calcTxKey(heightstr)), txdetailbyte)
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

	privacyKeyPair, err := wallet.getPrivacykeyPair(addr)
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
		wallet.GetPrivacyTxDetailByHashs(&ReqHashes, privacyKeyPair, &addr)
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
	//入参数校验，seed必须是15个单词或者汉字
	if len(password) == 0 || len(seed) == 0 {
		return false, types.ErrInputPara
	}

	seedarry := strings.Fields(seed)
	if len(seedarry) != SeedLong {
		return false, types.ErrSeedWordNum
	}
	var newseed string
	for index, seedstr := range seedarry {
		//walletlog.Error("saveSeed", "seedstr", seedstr)
		if index != SeedLong-1 {
			newseed += seedstr + " "
		} else {
			newseed += seedstr
		}
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
	if wallet.IsWalletLocked() && !wallet.isTicketLocked {
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
	s.IsTicketLock = wallet.isTicketLocked
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
	return strings.ToUpper(common.ToHex(priv.Bytes())), nil
}

func (wallet *Wallet) procTokenPreCreate(reqTokenPrcCreate *types.ReqTokenPreCreate) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}

	if reqTokenPrcCreate == nil {
		walletlog.Error("procTokenPreCreate input para is nil")
		return nil, types.ErrInputPara
	}

	upSymbol := strings.ToUpper(reqTokenPrcCreate.GetSymbol())
	if upSymbol != reqTokenPrcCreate.GetSymbol() {
		walletlog.Error("procTokenPreCreate", "symbol need be upper", reqTokenPrcCreate.GetSymbol())
		return nil, types.ErrTokenSymbolUpper
	}

	total := reqTokenPrcCreate.GetTotal()
	if total > types.MaxTokenBalance || total <= 0 {
		walletlog.Error("procTokenPreCreate", "total overflow", total)
		return nil, types.ErrTokenTotalOverflow
	}

	creator := reqTokenPrcCreate.GetCreatorAddr()
	addrs := make([]string, 1)
	addrs[0] = creator
	accounts, err := accountdb.LoadAccounts(wallet.client, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("procTokenPreCreate", "LoadAccounts err", err)
		return nil, err
	}

	Balance := accounts[0].Balance
	if Balance < wallet.FeeAmount {
		return nil, types.ErrInsufficientBalance
	}

	creatorAcc, err := accountdb.LoadExecAccountQueue(wallet.client, creator, account.ExecAddress("token").String())
	if err != nil {
		walletlog.Error("procTokenPreCreate", "LoadExecAccountQueue err", err)
		return nil, err
	}

	price := reqTokenPrcCreate.GetPrice()
	if creatorAcc.Balance < price {
		return nil, types.ErrInsufficientBalance
	}

	//  symbol 不存在
	token, err := wallet.checkTokenSymbolExists(reqTokenPrcCreate.GetSymbol(), reqTokenPrcCreate.GetOwnerAddr())
	if err != nil {
		return nil, err
	}
	if token != nil {
		walletlog.Error("procTokenPreCreate", "err", types.ErrTokenExist)
		return nil, types.ErrTokenExist
	}

	priv, err := wallet.getPrivKeyByAddr(addrs[0])
	if err != nil {
		return nil, err
	}

	return wallet.tokenPreCreate(priv, reqTokenPrcCreate)
}

func (wallet *Wallet) procTokenFinishCreate(req *types.ReqTokenFinishCreate) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if req == nil {
		walletlog.Error("procTokenFinishCreate input para is nil")
		return nil, types.ErrInputPara
	}

	upSymbol := strings.ToUpper(req.GetSymbol())
	if upSymbol != req.GetSymbol() {
		walletlog.Error("procTokenFinishCreate", "symbol need be upper", req.GetSymbol())
		return nil, types.ErrTokenSymbolUpper
	}

	addrs := make([]string, 1)
	addrs[0] = req.GetFinisherAddr()
	accounts, err := accountdb.LoadAccounts(wallet.client, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("procTokenFinishCreate", "LoadAccounts err", err)
		return nil, err
	}

	Balance := accounts[0].Balance
	if Balance < wallet.FeeAmount {
		return nil, types.ErrInsufficientBalance
	}

	//  check symbol-owner 是否不存在
	token, err := wallet.checkTokenSymbolExists(req.GetSymbol(), req.GetOwnerAddr())
	if err != nil {
		return nil, err
	}
	if token != nil {
		walletlog.Error("procTokenFinishCreate", "err", types.ErrTokenExist)
		return nil, types.ErrTokenExist
	}

	token2, err2 := wallet.checkTokenStatus(req.GetSymbol(), types.TokenStatusPreCreated, req.GetOwnerAddr())
	if err2 != nil {
		return nil, err
	}
	if token2 == nil {
		walletlog.Error("procTokenFinishCreate", "err", types.ErrTokenNotPrecreated)
		return nil, types.ErrTokenNotPrecreated
	}

	creatorAcc, err3 := accountdb.LoadExecAccountQueue(wallet.client, token2.Creator, account.ExecAddress("token").String())
	if err3 != nil {
		walletlog.Error("procTokenFinishCreate", "LoadAccounts err", err3)
		return nil, err3
	}

	frozen := creatorAcc.Frozen
	if frozen < token2.Price {
		return nil, types.ErrInsufficientBalance
	}

	priv, err := wallet.getPrivKeyByAddr(addrs[0])
	if err != nil {
		return nil, err
	}

	return wallet.tokenFinishCreate(priv, req)
}

func (wallet *Wallet) checkTokenSymbolExists(symbol, owner string) (*types.Token, error) {
	//通过txhashs获取对应的txdetail
	token := types.ReqString{Data: symbol}
	query := types.Query{Execer: []byte("token"), FuncName: "GetTokenInfo", Payload: types.Encode(&token)}
	msg := wallet.client.NewMessage("blockchain", types.EventQuery, &query)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil && err != types.ErrEmpty {
		walletlog.Error("checkTokenSymbolExists", "err", err)
		return nil, err
	} else if err == types.ErrEmpty {
		return nil, nil
	}

	tokenInfo := resp.GetData().(*types.Token)
	if tokenInfo == nil {
		walletlog.Info("checkTokenSymbolExists  is nil")
		return nil, nil
	}

	walletlog.Debug("checkTokenSymbolExists", "tokenInfo", tokenInfo.String())
	return tokenInfo, nil
}

func (wallet *Wallet) checkTokenStatus(symbol string, status int32, owner string) (*types.Token, error) {
	tokens := []string{symbol}
	reqtokens := types.ReqTokens{false, status, tokens}

	query := types.Query{Execer: []byte("token"), FuncName: "GetTokens", Payload: types.Encode(&reqtokens)}
	msg := wallet.client.NewMessage("blockchain", types.EventQuery, &query)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil && err != types.ErrEmpty {
		walletlog.Error("checkTokenSymbolStauts", "err", err)
		return nil, err
	} else if err == types.ErrEmpty {
		return nil, nil
	}

	tokenInfos := resp.GetData().(*types.ReplyTokens).Tokens
	if tokenInfos == nil {
		walletlog.Info("checkTokenSymbolStauts  is nil")
		return nil, nil
	}
	for _, tokenInfo := range tokenInfos {
		if tokenInfo.GetOwner() == owner {
			return tokenInfo, nil
		}
	}

	walletlog.Debug("checkTokenSymbolStauts", "tokenInfo", "not find")
	return nil, nil
}

func (wallet *Wallet) procTokenRevokeCreate(req *types.ReqTokenRevokeCreate) (*types.ReplyHash, error) {
	if req == nil {
		walletlog.Error("procTokenRevokeCreate input para is nil")
		return nil, types.ErrInputPara
	}

	upSymbol := strings.ToUpper(req.GetSymbol())
	if upSymbol != req.GetSymbol() {
		walletlog.Error("procTokenRevokeCreate", "symbol need be upper", req.GetSymbol())
		return nil, types.ErrTokenSymbolUpper
	}

	addrs := make([]string, 1)
	addrs[0] = req.GetRevokerAddr()
	accounts, err := accountdb.LoadAccounts(wallet.client, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("procTokenRevokeCreate", "LoadAccounts err", err)
		return nil, err
	}

	//  check symbol-owner 是否不存在, 是否是precreate 状态， 地址是否对应
	token, err := wallet.checkTokenStatus(req.GetSymbol(), types.TokenStatusPreCreated, req.GetOwnerAddr())
	if err != nil {
		return nil, err
	}
	if token == nil {
		walletlog.Error("procTokenRevokeCreate", "err", types.ErrTokenNotPrecreated)
		return nil, types.ErrTokenNotPrecreated
	}

	if req.RevokerAddr != token.Owner && req.RevokerAddr != token.Creator {
		walletlog.Error("tprocTokenRevokeCreate, different creator/owner vs actor of this revoke",
			"action.fromaddr", req.RevokerAddr, "creator", token.Creator, "owner", token.Owner)
		return nil, types.ErrTokenRevoker
	}

	priv, err := wallet.getPrivKeyByAddr(addrs[0])
	if err != nil {
		return nil, err
	}

	return wallet.tokenRevokeCreate(priv, req)
}

func (wallet *Wallet) procSellToken(reqSellToken *types.ReqSellToken) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if reqSellToken == nil {
		walletlog.Error("procSellToken input para is nil")
		return nil, types.ErrInputPara
	}

	addrs := make([]string, 1)
	addrs[0] = reqSellToken.GetOwner()
	accountTokendb := getTokenAccountDB(reqSellToken.Sell.Tokensymbol)
	accounts, err := accountTokendb.LoadAccounts(wallet.client, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("procSellToken", "LoadAccounts err", err)
		return nil, err
	}

	balance := accounts[0].Balance
	if balance < reqSellToken.Sell.Amountperboardlot*reqSellToken.Sell.Totalboardlot {
		return nil, types.ErrInsufficientBalance
	}

	priv, err := wallet.getPrivKeyByAddr(addrs[0])
	if err != nil {
		return nil, err
	}

	return wallet.sellToken(priv, reqSellToken)
}

func (wallet *Wallet) procBuyToken(reqBuyToken *types.ReqBuyToken) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}

	if reqBuyToken == nil {
		walletlog.Error("procBuyToken input para is nil")
		return nil, types.ErrInputPara
	}
	execaddress := account.ExecAddress("trade")
	account, err := accountdb.LoadExecAccountQueue(wallet.client, reqBuyToken.GetBuyer(), execaddress.String())
	if err != nil {
		log.Error("GetBalance", "err", err.Error())
		return nil, err
	}
	balance := account.Balance

	var sellorder *types.SellOrder
	if sellorder, err = loadSellOrderQueue(wallet.client, reqBuyToken.GetBuy().GetSellid()); err != nil {
		walletlog.Error("procBuyToken failed to loadSellOrderQueue", "token sellid", reqBuyToken.GetBuy().GetSellid())
		return nil, err
	}

	if balance < reqBuyToken.Buy.Boardlotcnt*sellorder.Priceperboardlot {
		return nil, types.ErrInsufficientBalance
	} else if reqBuyToken.Buy.Boardlotcnt > (sellorder.Totalboardlot - sellorder.Soldboardlot) {
		return nil, types.ErrInsuffSellOrder
	}

	priv, err := wallet.getPrivKeyByAddr(reqBuyToken.GetBuyer())
	if err != nil {
		return nil, err
	}

	return wallet.buyToken(priv, reqBuyToken)
}

func (wallet *Wallet) procRevokeSell(reqRevoke *types.ReqRevokeSell) (*types.ReplyHash, error) {

	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if reqRevoke == nil {
		walletlog.Error("procBuyToken input para is nil")
		return nil, types.ErrInputPara
	}

	priv, err := wallet.getPrivKeyByAddr(reqRevoke.GetOwner())
	if err != nil {
		return nil, err
	}

	return wallet.revokeSell(priv, reqRevoke)
}

func getTokenAccountDB(token string) *account.AccountDB {
	if nil == accTokenMap[token] {
		tokenAccDB := account.NewTokenAccountWithoutDB(token)
		accTokenMap[token] = tokenAccDB
	}
	return accTokenMap[token]
}

func loadSellOrderQueue(client queue.Client, sellid string) (*types.SellOrder, error) {
	msg := client.NewMessage("blockchain", types.EventGetLastHeader, nil)
	client.Send(msg, true)
	msg, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}
	get := types.StoreGet{}
	get.StateHash = msg.GetData().(*types.Header).GetStateHash()
	get.Keys = append(get.Keys, []byte(sellid))
	msg = client.NewMessage("store", types.EventStoreGet, &get)
	client.Send(msg, true)
	msg, err = client.Wait(msg)
	if err != nil {
		return nil, err
	}
	values := msg.GetData().(*types.StoreReplyValue)
	value := values.Values[0]
	if value == nil {
		return nil, types.ErrTSellNoSuchOrder
	} else {
		var sellOrder types.SellOrder
		err := types.Decode(value, &sellOrder)
		if err != nil {
			return nil, err
		}
		return &sellOrder, nil
	}
}

//检测钱包是否允许转账到指定地址，判断钱包锁和是否有seed以及挖矿锁
func (wallet *Wallet) IsTransfer(addr string) (bool, error) {

	ok, err := wallet.CheckWalletStatus()
	//钱包已经解锁或者错误是ErrSaveSeedFirst直接返回
	if ok || err == types.ErrSaveSeedFirst {
		return ok, err
	}
	//钱包已经锁定，挖矿锁已经解锁,需要判断addr是否是挖矿合约地址
	if wallet.isTicketLocked == false {
		if addr == account.ExecAddress("ticket").String() {
			return true, nil
		}
	}
	return ok, err

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

func (wallet *Wallet) showPrivacyAccounts(req *types.ReqStr) ([]*types.PrivacyOnetimeAccInfo, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	nilaccRes := make([]*types.PrivacyOnetimeAccInfo, 0)
	addr := req.GetReqstr()
	privacyDBStore, err := wallet.walletStore.listWalletPrivacyAccount(addr)
	if err != nil {
		return nilaccRes, err
	}

	if 0 == len(privacyDBStore) {
		return nilaccRes, nil
	}

	accRes := make([]*types.PrivacyOnetimeAccInfo, len(privacyDBStore))
	for index, ele := range privacyDBStore {
		addrOneTime := ele.Onetimeaddr
		execaddress := account.ExecAddress(types.PrivacyX)
		account, err := accountdb.LoadExecAccountQueue(wallet.client, addrOneTime, execaddress.String())
		if err != nil {
			walletlog.Error("showPrivacyBalance", "err", err.Error())
			return nil, err
		}
		accinfo := &types.PrivacyOnetimeAccInfo{
			Tokenname: ele.Tokenname,
			Balance:   account.Balance,
			Frozen:    account.Frozen,
			Addr:      account.Addr,
			Txhash:    ele.Txhash,
		}
		accRes[index] = accinfo
	}

	return accRes, nil
}

func (wallet *Wallet) showPrivacyBalance(req *types.ReqPrivacyBalance) (*types.Account, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	privacyInfo, err := wallet.getPrivacykeyPair(req.GetAddr())
	if err != nil {
		return nil, err
	}

	R, err := wallet.GetRofPrivateTx(&req.Txhash)
	if err != nil {
		walletlog.Error("transPri2Pri", "Failed to GetRofPrivateTx")
		return nil, err
	}
	//x = Hs(aR) + b
	priv, err := privacy.RecoverOnetimePriKey(R, privacyInfo.ViewPrivKey, privacyInfo.SpendPrivKey)
	if err != nil {
		walletlog.Error("transPri2Pri", "Failed to RecoverOnetimePriKey", err)
		return nil, err
	}
	pub := priv.PubKey()

	addrOneTime := account.PubKeyToAddress(pub.Bytes()[:]).String()

	execaddress := account.ExecAddress(types.PrivacyX)
	account, err := accountdb.LoadExecAccountQueue(wallet.client, addrOneTime, execaddress.String())
	if err != nil {
		walletlog.Error("showPrivacyBalance", "err", err.Error())
		return nil, err
	}

	return account, nil
}

func (wallet *Wallet) showPrivacyPkPair(reqAddr *types.ReqStr) (*types.ReplyPrivacyPkPair, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	privacyInfo, err := wallet.getPrivacykeyPair(reqAddr.GetReqstr())
	if err != nil {
		return nil, err
	}

	replyPrivacyPkPair := &types.ReplyPrivacyPkPair{
		true,
		common.ToHex(privacyInfo.ViewPubkey[:]),
		common.ToHex(privacyInfo.SpendPubkey[:]),
	}

	return replyPrivacyPkPair, nil
}

func (wallet *Wallet) procPublic2Privacy(public2private *types.ReqPub2Pri) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if public2private == nil {
		walletlog.Error("public2private input para is nil")
		return nil, types.ErrInputPara
	}

	priv, err := wallet.getPrivKeyByAddr(public2private.GetSender())
	if err != nil {
		return nil, err
	}

	return wallet.transPub2Pri(priv, public2private)
}

func (wallet *Wallet) procPrivacy2Privacy(privacy2privacy *types.ReqPri2Pri) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if privacy2privacy == nil {
		walletlog.Error("privacy2privacy input para is nil")
		return nil, types.ErrInputPara
	}

	privacyInfo, err := wallet.getPrivacykeyPair(privacy2privacy.GetSender())
	if err != nil {
		walletlog.Error("privacy2privacy failed to getPrivacykeyPair")
		return nil, err
	}

	return wallet.transPri2Pri(privacyInfo, privacy2privacy)
}

func (wallet *Wallet) procPrivacy2Public(privacy2Pub *types.ReqPri2Pub) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.CheckWalletStatus()
	if !ok {
		return nil, err
	}
	if privacy2Pub == nil {
		walletlog.Error("privacy2privacy input para is nil")
		return nil, types.ErrInputPara
	}
	//get 'a'
	privacyInfo, err := wallet.getPrivacykeyPair(privacy2Pub.GetSender())
	if err != nil {
		return nil, err
	}

	return wallet.transPri2Pub(privacyInfo, privacy2Pub)
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
