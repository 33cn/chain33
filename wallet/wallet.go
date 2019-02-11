// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package wallet wallet chain33钱包功能实现
package wallet

import (
	"errors"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	dbm "github.com/33cn/chain33/common/db"
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	wcom "github.com/33cn/chain33/wallet/common"
)

var (
	minFee           int64
	maxTxNumPerBlock int64 = types.MaxTxsPerBlock
	// MaxTxHashsPerTime 每次处理的最大交易哈希数量
	MaxTxHashsPerTime int64 = 100
	walletlog               = log.New("module", "wallet")
	// SignType 签名类型 1；secp256k1，2：ed25519，3：sm2
	SignType    = 1
	accountdb   *account.DB
	accTokenMap = make(map[string]*account.DB)
)

func init() {
	wcom.QueryData.Register("wallet", &Wallet{})
}

const (
	// AddTx 添加交易操作
	AddTx int32 = 20001
	// DelTx 删除交易操作
	DelTx int32 = 20002
	// 交易收发方向
	sendTx int32 = 30001
	recvTx int32 = 30002
)

// Wallet 钱包功能的实现类
type Wallet struct {
	client queue.Client
	// 模块间通信的操作接口,建议用api代替client调用
	api                client.QueueProtocolAPI
	mtx                sync.Mutex
	timeout            *time.Timer
	mineStatusReporter wcom.MineStatusReport
	isclosed           int32
	isWalletLocked     int32
	fatalFailureFlag   int32
	Password           string
	FeeAmount          int64
	EncryptFlag        int64
	wg                 *sync.WaitGroup
	walletStore        *walletStore
	random             *rand.Rand
	cfg                *types.Wallet
	done               chan struct{}
	rescanwg           *sync.WaitGroup
	lastHeader         *types.Header
	initFlag           uint32 // 钱包模块是否初始化完毕的标记，默认为0，表示未初始化
}

// SetLogLevel 设置日志登记
func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

// DisableLog 禁用日志
func DisableLog() {
	walletlog.SetHandler(log.DiscardHandler())
	storelog.SetHandler(log.DiscardHandler())
}

// New 创建一个钱包对象
func New(cfg *types.Wallet, sub map[string][]byte) *Wallet {
	//walletStore
	accountdb = account.NewCoinsAccount()
	walletStoreDB := dbm.NewDB("wallet", cfg.Driver, cfg.DbPath, cfg.DbCache)
	//walletStore := NewStore(walletStoreDB)
	walletStore := newStore(walletStoreDB)
	minFee = cfg.MinFee
	signType := types.GetSignType("", cfg.SignType)
	if signType == types.Invalid {
		signType = types.SECP256K1
	}
	SignType = signType

	wallet := &Wallet{
		walletStore:      walletStore,
		isWalletLocked:   1,
		fatalFailureFlag: 0,
		wg:               &sync.WaitGroup{},
		FeeAmount:        walletStore.GetFeeAmount(minFee),
		EncryptFlag:      walletStore.GetEncryptionFlag(),
		done:             make(chan struct{}),
		cfg:              cfg,
		rescanwg:         &sync.WaitGroup{},
		initFlag:         0,
	}
	wallet.random = rand.New(rand.NewSource(types.Now().UnixNano()))
	wcom.QueryData.SetThis("wallet", reflect.ValueOf(wallet))
	wcom.Init(wallet, sub)
	return wallet
}

//Wait for wallet ready
func (wallet *Wallet) Wait() {}

// RegisterMineStatusReporter 向钱包注册状态回报
func (wallet *Wallet) RegisterMineStatusReporter(reporter wcom.MineStatusReport) error {
	if reporter == nil {
		return types.ErrInvalidParam
	}
	if wallet.mineStatusReporter != nil {
		return errors.New("ReporterIsExisted")
	}
	wallet.mineStatusReporter = reporter
	return nil
}

// GetConfig 获取钱包配置
func (wallet *Wallet) GetConfig() *types.Wallet {
	return wallet.cfg
}

// GetAPI 获取操作API
func (wallet *Wallet) GetAPI() client.QueueProtocolAPI {
	return wallet.api
}

// GetMutex 获取钱包互斥量
func (wallet *Wallet) GetMutex() *sync.Mutex {
	return &wallet.mtx
}

// GetDBStore 获取数据库存储对象操作接口
func (wallet *Wallet) GetDBStore() dbm.DB {
	return wallet.walletStore.GetDB()
}

// GetSignType 获取签名类型
func (wallet *Wallet) GetSignType() int {
	return SignType
}

// GetPassword 获取密码
func (wallet *Wallet) GetPassword() string {
	return wallet.Password
}

// Nonce 获取随机值
func (wallet *Wallet) Nonce() int64 {
	return wallet.random.Int63()
}

// AddWaitGroup 添加一个分组等待事件
func (wallet *Wallet) AddWaitGroup(delta int) {
	wallet.wg.Add(delta)
}

// WaitGroupDone 完成分组事件
func (wallet *Wallet) WaitGroupDone() {
	wallet.wg.Done()
}

// GetBlockHeight 获取区块高度
func (wallet *Wallet) GetBlockHeight() int64 {
	return wallet.GetHeight()
}

// GetRandom 获取随机值
func (wallet *Wallet) GetRandom() *rand.Rand {
	return wallet.random
}

// GetWalletDone 是否结束的通道
func (wallet *Wallet) GetWalletDone() chan struct{} {
	return wallet.done
}

// GetLastHeader 获取最新高度信息
func (wallet *Wallet) GetLastHeader() *types.Header {
	return wallet.lastHeader
}

// GetWaitGroup 获取等待互斥量
func (wallet *Wallet) GetWaitGroup() *sync.WaitGroup {
	return wallet.wg
}

// GetAccountByLabel 根据标签获取账号
func (wallet *Wallet) GetAccountByLabel(label string) (*types.WalletAccountStore, error) {
	return wallet.walletStore.GetAccountByLabel(label)
}

// IsRescanUtxosFlagScaning 是否处于扫描UTXO状态
func (wallet *Wallet) IsRescanUtxosFlagScaning() (bool, error) {
	in := &types.ReqNil{}
	flag := false
	for _, policy := range wcom.PolicyContainer {
		out, err := policy.Call("GetUTXOScaningFlag", in)
		if err != nil {
			if err.Error() == types.ErrNotSupport.Error() {
				continue
			}
			return flag, err
		}
		reply, ok := out.(*types.Reply)
		if !ok {
			err = types.ErrTypeAsset
			return flag, err
		}
		flag = reply.IsOk
		return flag, err
	}

	return flag, nil
}

// Close 关闭钱包
func (wallet *Wallet) Close() {
	//等待所有的子线程退出
	//set close flag to isclosed == 1
	atomic.StoreInt32(&wallet.isclosed, 1)
	for _, policy := range wcom.PolicyContainer {
		policy.OnClose()
	}
	close(wallet.done)
	wallet.client.Close()
	wallet.wg.Wait()
	//关闭数据库
	wallet.walletStore.Close()
	walletlog.Info("wallet module closed")
}

// IsClose 检查是否处于关闭状态
func (wallet *Wallet) IsClose() bool {
	return atomic.LoadInt32(&wallet.isclosed) == 1
}

// IsWalletLocked 返回钱包锁的状态
func (wallet *Wallet) IsWalletLocked() bool {
	return atomic.LoadInt32(&wallet.isWalletLocked) != 0
}

// SetQueueClient 初始化客户端消息队列
func (wallet *Wallet) SetQueueClient(cli queue.Client) {
	var err error
	wallet.client = cli
	wallet.client.Sub("wallet")
	wallet.api, err = client.New(cli, nil)
	if err != nil {
		panic("SetQueueClient client.New err")
	}
	wallet.wg.Add(1)
	go wallet.ProcRecvMsg()
	for _, policy := range wcom.PolicyContainer {
		policy.OnSetQueueClient()
	}
	wallet.setInited(true)
}

// GetAccountByAddr 根据地址获取账户
func (wallet *Wallet) GetAccountByAddr(addr string) (*types.WalletAccountStore, error) {
	return wallet.walletStore.GetAccountByAddr(addr)
}

// SetWalletAccount 设置钱包账户
func (wallet *Wallet) SetWalletAccount(update bool, addr string, account *types.WalletAccountStore) error {
	return wallet.walletStore.SetWalletAccount(update, addr, account)
}

// GetPrivKeyByAddr 根据地址获取私钥
func (wallet *Wallet) GetPrivKeyByAddr(addr string) (crypto.PrivKey, error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}
	return wallet.getPrivKeyByAddr(addr)
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

	privkey := wcom.CBCDecrypterPrivkey([]byte(wallet.Password), prikeybyte)
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignName("", SignType))
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

//外部已经加了lock
func (wallet *Wallet) getFee() int64 {
	return wallet.FeeAmount
}

// AddrInWallet 地址对应的账户是否属于本钱包
func (wallet *Wallet) AddrInWallet(addr string) bool {
	if !wallet.isInited() {
		return false
	}
	if len(addr) == 0 {
		return false
	}
	acc, err := wallet.walletStore.GetAccountByAddr(addr)
	if err == nil && acc != nil {
		return true
	}
	return false
}

//IsTransfer 检测钱包是否允许转账到指定地址，判断钱包锁和是否有seed以及挖矿锁
func (wallet *Wallet) IsTransfer(addr string) (bool, error) {

	ok, err := wallet.CheckWalletStatus()
	//钱包已经解锁或者错误是ErrSaveSeedFirst直接返回
	if ok || err == types.ErrSaveSeedFirst {
		return ok, err
	}
	//钱包已经锁定，挖矿锁已经解锁,需要判断addr是否是挖矿合约地址
	//这里依赖了ticket 挖矿合约
	if !wallet.isTicketLocked() {
		if addr == address.ExecAddress("ticket") {
			return true, nil
		}
	}
	return ok, err
}

//CheckWalletStatus 钱包状态检测函数,解锁状态，seed是否已保存
func (wallet *Wallet) CheckWalletStatus() (bool, error) {
	if !wallet.isInited() {
		return false, types.ErrNotInited
	}
	// 钱包锁定，ticket已经解锁，返回只解锁了ticket的错误
	if wallet.IsWalletLocked() && !wallet.isTicketLocked() {
		return false, types.ErrOnlyTicketUnLocked
	} else if wallet.IsWalletLocked() {
		return false, types.ErrWalletIsLocked
	}

	//判断钱包是否已保存seed
	has, err := wallet.walletStore.HasSeed()
	if !has || err != nil {
		return false, types.ErrSaveSeedFirst
	}
	return true, nil
}

func (wallet *Wallet) isTicketLocked() bool {
	locked := true
	if wallet.mineStatusReporter != nil {
		locked = wallet.mineStatusReporter.IsTicketLocked()
	}
	return locked
}

func (wallet *Wallet) isAutoMinning() bool {
	autoMining := false
	if wallet.mineStatusReporter != nil {
		autoMining = wallet.mineStatusReporter.IsAutoMining()
	}
	return autoMining
}

// GetWalletStatus 获取钱包状态
func (wallet *Wallet) GetWalletStatus() *types.WalletStatus {
	var err error
	s := &types.WalletStatus{}
	s.IsWalletLock = wallet.IsWalletLocked()
	s.IsHasSeed, err = wallet.walletStore.HasSeed()
	s.IsAutoMining = wallet.isAutoMinning()
	s.IsTicketLock = wallet.isTicketLocked()
	if err != nil {
		walletlog.Debug("GetWalletStatus HasSeed ", "err", err)
	}
	walletlog.Debug("GetWalletStatus", "walletstatus", s)
	return s
}

// GetWalletAccounts 获取账号列表
//output:
//type WalletAccountStore struct {
//	Privkey   string  //加密后的私钥hex值
//	Label     string
//	Addr      string
//	TimeStamp string
//获取钱包的所有账户地址列表，
func (wallet *Wallet) GetWalletAccounts() ([]*types.WalletAccountStore, error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	//通过Account前缀查找获取钱包中的所有账户信息
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Info("GetWalletAccounts", "GetAccountByPrefix:err", err)
		return nil, err
	}
	return WalletAccStores, err
}

func (wallet *Wallet) updateLastHeader(block *types.BlockDetail, mode int) error {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	header, err := wallet.api.GetLastHeader()
	if err != nil {
		return err
	}
	if block != nil {
		if mode == 1 && block.Block.Height > header.Height {
			wallet.lastHeader = &types.Header{
				BlockTime: block.Block.BlockTime,
				Height:    block.Block.Height,
				StateHash: block.Block.StateHash,
			}
		} else if mode == -1 && wallet.lastHeader != nil && wallet.lastHeader.Height == block.Block.Height {
			wallet.lastHeader = header
		}
	}
	if block == nil || wallet.lastHeader == nil {
		wallet.lastHeader = header
	}
	return nil
}

func (wallet *Wallet) setInited(flag bool) {
	if flag && !wallet.isInited() {
		atomic.StoreUint32(&wallet.initFlag, 1)
	}
}

func (wallet *Wallet) isInited() bool {
	return atomic.LoadUint32(&wallet.initFlag) != 0
}
