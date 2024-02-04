// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"sync"
	"time"

	etypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/golang/protobuf/proto"

	lru "github.com/hashicorp/golang-lru"

	"strconv"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
)

var (
	bToken   = []byte("token")
	withdraw = "withdraw"
	txCache  *lru.Cache
)

func init() {
	var err error
	txCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
}

// TxCacheGet 某些交易的cache 加入缓存中，防止重复进行解析或者计算
func TxCacheGet(tx *Transaction) (*TransactionCache, bool) {
	txc, ok := txCache.Get(tx)
	if !ok {
		return nil, ok
	}
	return txc.(*TransactionCache), ok
}

// TxCacheSet 设置 cache
func TxCacheSet(tx *Transaction, txc *TransactionCache) {
	if txc == nil {
		txCache.Remove(tx)
		return
	}
	txCache.Add(tx, txc)
}

// CreateTxGroup 创建组交易, feeRate传入交易费率, 建议通过系统GetProperFee获取
func CreateTxGroup(txs []*Transaction, feeRate int64) (*Transactions, error) {
	if len(txs) < 2 {
		return nil, ErrTxGroupCountLessThanTwo
	}
	txgroup := &Transactions{}
	txgroup.Txs = txs
	totalfee := int64(0)
	minfee := int64(0)
	header := txs[0].Hash()
	for i := len(txs) - 1; i >= 0; i-- {
		txs[i].GroupCount = int32(len(txs))
		totalfee += txs[i].GetFee()
		// Header和Fee设置是为了GetRealFee里面Size的计算，Fee是否为0和不同大小，size也是有差别的，header是否为空差别是common.Sha256Len+2
		// 这里直接设置Header兼容性更好， Next不需要，已经设置过了，唯一不同的是，txs[0].fee会跟实际计算有差别，这里设置一个超大值只做计算
		txs[i].Header = header
		if i == 0 {
			//对txs[0].fee设置一个超大值，大于后面实际计算出的fee，也就>=check时候计算出的fee， 对size影响10个字节，在1000临界值时候有差别
			txs[i].Fee = 1 << 62
		} else {
			txs[i].Fee = 0
		}
		realfee, err := txs[i].GetRealFee(feeRate)
		if err != nil {
			return nil, err
		}
		minfee += realfee
		if i == 0 {
			if totalfee < minfee {
				totalfee = minfee
			}
			txs[0].Fee = totalfee
			header = txs[0].Hash()
		} else {
			txs[i].Fee = 0
			txs[i-1].Next = txs[i].Hash()
		}
	}
	for i := 0; i < len(txs); i++ {
		txs[i].Header = header
	}
	return txgroup, nil
}

// Tx 这比用于检查的交易，包含了所有的交易。
// 主要是为了兼容原来的设计
func (txgroup *Transactions) Tx() *Transaction {
	if len(txgroup.GetTxs()) < 2 {
		return nil
	}
	headtx := txgroup.GetTxs()[0]
	//不会影响原来的tx
	copytx := CloneTx(headtx)
	data := Encode(txgroup)
	//放到header中不影响交易的Hash
	copytx.Header = data
	return copytx
}

// GetTxGroup 获取交易组
func (txgroup *Transactions) GetTxGroup() *Transactions {
	return txgroup
}

// SignN 对交易组的第n笔交易签名
func (txgroup *Transactions) SignN(n int, ty int32, priv crypto.PrivKey) error {
	if n >= len(txgroup.GetTxs()) {
		return ErrIndex
	}
	txgroup.GetTxs()[n].Sign(ty, priv)
	return nil
}

// CheckSign 检测交易组的签名
func (txgroup *Transactions) CheckSign(blockHeight int64) bool {
	txs := txgroup.Txs
	for i := 0; i < len(txs); i++ {
		if !txs[i].checkSign(blockHeight) {
			return false
		}
	}
	return true
}

// RebuiltGroup 交易内容有变化时需要重新构建交易组
func (txgroup *Transactions) RebuiltGroup() {
	header := txgroup.Txs[0].Hash()
	for i := len(txgroup.Txs) - 1; i >= 0; i-- {
		txgroup.Txs[i].Header = header
		if i == 0 {
			header = txgroup.Txs[0].Hash()
		} else {
			txgroup.Txs[i-1].Next = txgroup.Txs[i].Hash()
		}
	}
	for i := 0; i < len(txgroup.Txs); i++ {
		txgroup.Txs[i].Header = header
	}
}

// SetExpire 设置交易组中交易的过期时间
func (txgroup *Transactions) SetExpire(cfg *Chain33Config, n int, expire time.Duration) {
	if n >= len(txgroup.GetTxs()) {
		return
	}
	txgroup.GetTxs()[n].SetExpire(cfg, expire)
}

// IsExpire 交易是否过期
func (txgroup *Transactions) IsExpire(cfg *Chain33Config, height, blocktime int64) bool {
	txs := txgroup.Txs
	for i := 0; i < len(txs); i++ {
		if txs[i].isExpire(cfg, height, blocktime) {
			return true
		}
	}
	return false
}

// CheckWithFork 和fork 无关的有个检查函数
func (txgroup *Transactions) CheckWithFork(cfg *Chain33Config, checkFork, paraFork bool, height, minfee, maxFee int64) error {
	txs := txgroup.Txs
	if len(txs) < 2 {
		return ErrTxGroupCountLessThanTwo
	}
	para := make(map[string]bool)
	for i := 0; i < len(txs); i++ {
		if txs[i] == nil {
			return ErrTxGroupEmpty
		}
		err := txs[i].check(cfg, height, 0, maxFee)
		if err != nil {
			return err
		}
		if title, ok := GetParaExecTitleName(string(txs[i].Execer)); ok {
			para[title] = true
		}
	}
	//txgroup 只允许一条平行链的交易, 且平行链txgroup须全部是平行链tx
	//如果平行链已经在主链分叉高度前运行了一段时间且有跨链交易，平行链需要自己设置这个fork
	if paraFork {
		if len(para) > 1 {
			tlog.Info("txgroup has multi para transaction")
			return ErrTxGroupParaCount
		}
		if len(para) > 0 {
			for _, tx := range txs {
				if !IsParaExecName(string(tx.Execer)) {
					tlog.Error("para txgroup has main chain transaction")
					return ErrTxGroupParaMainMixed
				}
			}
		}
	}

	for i := 1; i < len(txs); i++ {
		if txs[i].Fee != 0 {
			return ErrTxGroupFeeNotZero
		}
	}
	//检查txs[0] 的费用是否满足要求
	totalfee := int64(0)
	for i := 0; i < len(txs); i++ {
		fee, err := txs[i].GetRealFee(minfee)
		if err != nil {
			return err
		}
		totalfee += fee
	}
	if txs[0].Fee < totalfee {
		return ErrTxFeeTooLow
	}
	if txs[0].Fee > maxFee && maxFee > 0 && checkFork {
		return ErrTxFeeTooHigh
	}
	//检查hash是否符合要求
	for i := 0; i < len(txs); i++ {
		//检查头部是否等于头部hash
		if i == 0 {
			if !bytes.Equal(txs[i].Hash(), txs[i].Header) {
				return ErrTxGroupHeader
			}
		} else {
			if !bytes.Equal(txs[0].Header, txs[i].Header) {
				return ErrTxGroupHeader
			}
		}
		//检查group count
		if txs[i].GroupCount > MaxTxGroupSize {
			return ErrTxGroupCountBigThanMaxSize
		}
		if txs[i].GroupCount != int32(len(txs)) {
			return ErrTxGroupCount
		}
		//检查next
		if i < len(txs)-1 {
			if !bytes.Equal(txs[i].Next, txs[i+1].Hash()) {
				return ErrTxGroupNext
			}
		} else {
			if txs[i].Next != nil {
				return ErrTxGroupNext
			}
		}
	}
	return nil
}

// Check height == 0 的时候，不做检查
func (txgroup *Transactions) Check(cfg *Chain33Config, height, minfee, maxFee int64) error {
	paraFork := cfg.IsFork(height, "ForkTxGroupPara")
	checkFork := cfg.IsFork(height, "ForkBlockCheck")
	return txgroup.CheckWithFork(cfg, checkFork, paraFork, height, minfee, maxFee)
}

// TransactionCache 交易缓存结构
type TransactionCache struct {
	*Transaction
	txGroup *Transactions
	hash    []byte
	size    int
	signok  int   //init 0, ok 1, err 2
	checkok error //init 0, ok 1, err 2
	checked bool
	payload reflect.Value
	plname  string
	plerr   error
}

// NewTransactionCache new交易缓存
func NewTransactionCache(tx *Transaction) *TransactionCache {
	return &TransactionCache{Transaction: tx}
}

// Hash 交易hash
func (tx *TransactionCache) Hash() []byte {
	if tx.hash == nil {
		tx.hash = tx.Transaction.Hash()
	}
	return tx.hash
}

// SetPayloadValue 设置payload 的cache
func (tx *TransactionCache) SetPayloadValue(plname string, payload reflect.Value, plerr error) {
	tx.payload = payload
	tx.plerr = plerr
	tx.plname = plname
}

// GetPayloadValue 设置payload 的cache
func (tx *TransactionCache) GetPayloadValue() (plname string, payload reflect.Value, plerr error) {
	if tx.plerr != nil || tx.plname != "" {
		return tx.plname, tx.payload, tx.plerr
	}
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		tx.SetPayloadValue("", reflect.ValueOf(nil), ErrExecNotFound)
		return "", reflect.ValueOf(nil), ErrExecNotFound
	}
	plname, payload, plerr = exec.DecodePayloadValue(tx.Tx())
	tx.SetPayloadValue(plname, payload, plerr)
	return
}

// Size 交易缓存的大小
func (tx *TransactionCache) Size() int {
	if tx.size == 0 {
		tx.size = Size(tx.Tx())
	}
	return tx.size
}

// Tx 交易缓存中tx信息
func (tx *TransactionCache) Tx() *Transaction {
	return tx.Transaction
}

// Check 交易缓存中交易组合费用的检测
func (tx *TransactionCache) Check(cfg *Chain33Config, height, minfee, maxFee int64) error {
	if !tx.checked {
		tx.checked = true
		txs, err := tx.GetTxGroup()
		if err != nil {
			tx.checkok = err
			return err
		}
		if txs == nil {
			tx.checkok = tx.check(cfg, height, minfee, maxFee)
		} else {
			tx.checkok = txs.Check(cfg, height, minfee, maxFee)
		}
	}
	return tx.checkok
}

// GetTotalFee 获取交易真实费用
func (tx *TransactionCache) GetTotalFee(minFee int64) (int64, error) {
	txgroup, err := tx.GetTxGroup()
	if err != nil {
		tx.checkok = err
		return 0, err
	}
	var totalfee int64
	if txgroup == nil {
		return tx.GetRealFee(minFee)
	}
	txs := txgroup.Txs
	for i := 0; i < len(txs); i++ {
		fee, err := txs[i].GetRealFee(minFee)
		if err != nil {
			return 0, err
		}
		totalfee += fee
	}
	return totalfee, nil
}

// GetTxGroup 获取交易组
func (tx *TransactionCache) GetTxGroup() (*Transactions, error) {
	var err error
	if tx.txGroup == nil {
		tx.txGroup, err = tx.Transaction.GetTxGroup()
		if err != nil {
			return nil, err
		}
	}
	return tx.txGroup, nil
}

// CheckSign 检测签名
func (tx *TransactionCache) CheckSign(blockHeight int64) bool {
	if tx.signok == 0 {
		tx.signok = 2
		group, err := tx.GetTxGroup()
		if err != nil {
			return false
		}
		if group == nil {
			//非group，简单校验签名
			if ok := tx.checkSign(blockHeight); ok {
				tx.signok = 1
			}
		} else {
			if ok := group.CheckSign(blockHeight); ok {
				tx.signok = 1
			}
		}
	}
	return tx.signok == 1
}

// TxsToCache 缓存交易信息
func TxsToCache(txs []*Transaction) (caches []*TransactionCache) {
	caches = make([]*TransactionCache, len(txs))
	for i := 0; i < len(caches); i++ {
		caches[i] = NewTransactionCache(txs[i])
	}
	return caches
}

// CacheToTxs 从缓存中获取交易信息
func CacheToTxs(caches []*TransactionCache) (txs []*Transaction) {
	txs = make([]*Transaction, len(caches))
	for i := 0; i < len(caches); i++ {
		txs[i] = caches[i].Tx()
	}
	return txs
}

// Tx 交易详情
func (tx *Transaction) Tx() *Transaction {
	return tx
}

// GetTxGroup 交易组装成交易组格式
func (tx *Transaction) GetTxGroup() (*Transactions, error) {
	if tx.GroupCount < 0 || tx.GroupCount == 1 || tx.GroupCount > 20 {
		return nil, ErrTxGroupCount
	}
	if tx.GroupCount > 0 {
		var txs Transactions
		err := Decode(tx.Header, &txs)
		if err != nil {
			return nil, err
		}
		return &txs, nil
	}
	if tx.Next != nil || tx.Header != nil {
		return nil, ErrNomalTx
	}
	return nil, nil
}

// Size 交易大小
func (tx *Transaction) Size() int {
	return Size(tx)
}

// Sign 交易签名
func (tx *Transaction) Sign(ty int32, priv crypto.PrivKey) {
	tx.Signature = nil
	data := Encode(tx)
	pub := priv.PubKey()
	sign := priv.Sign(data)
	tx.Signature = &Signature{
		Ty:        ty,
		Pubkey:    pub.Bytes(),
		Signature: sign.Bytes(),
	}
}

// CheckSign tx 有些时候是一个交易组
func (tx *Transaction) CheckSign(blockHeight int64) bool {
	return tx.checkSign(blockHeight)
}

// txgroup 的情况
func (tx *Transaction) checkSign(blockHeight int64) bool {
	copytx := CloneTx(tx)
	copytx.Signature = nil
	data := Encode(copytx)
	FreeTx(copytx)
	if tx.GetSignature() == nil {
		return false
	}
	return CheckSign(data, string(tx.Execer), tx.GetSignature(), blockHeight)
}

// Check 交易检测
func (tx *Transaction) Check(cfg *Chain33Config, height, minfee, maxFee int64) error {
	group, err := tx.GetTxGroup()
	if err != nil {
		return err
	}
	if group == nil {
		return tx.check(cfg, height, minfee, maxFee)
	}
	return group.Check(cfg, height, minfee, maxFee)
}

func (tx *Transaction) check(cfg *Chain33Config, height, minfee, maxFee int64) error {
	if minfee == 0 {
		return nil
	}
	// 获取当前交易最小交易费
	realFee, err := tx.GetRealFee(minfee)
	if err != nil {
		return err
	}
	// 检查交易费是否小于最低值
	if tx.Fee < realFee {
		return ErrTxFeeTooLow
	}
	if tx.Fee > maxFee && maxFee > 0 && cfg.IsFork(height, "ForkBlockCheck") {
		return ErrTxFeeTooHigh
	}
	//增加交易中chainID的检测，
	if tx.ChainID != cfg.GetChainID() {
		return ErrTxChainID
	}
	return nil
}

// SetExpire 设置交易过期时间
func (tx *Transaction) SetExpire(cfg *Chain33Config, expire time.Duration) {
	//Txheight处理
	if cfg.IsEnable("TxHeight") && int64(expire) > TxHeightFlag {
		tx.Expire = int64(expire)
		return
	}

	if int64(expire) > ExpireBound {
		if expire < time.Second*120 {
			expire = time.Second * 120
		}
		//用秒数来表示的时间
		tx.Expire = Now().Unix() + int64(expire/time.Second)
	} else {
		tx.Expire = int64(expire)
	}
}

// GetRealFee 获取交易真实费用
func (tx *Transaction) GetRealFee(minFee int64) (int64, error) {
	txSize := Size(tx)
	//如果签名为空，那么加上签名的空间
	if tx.Signature == nil {
		txSize += 300
	}
	if txSize > MaxTxSize {
		return 0, ErrTxMsgSizeTooBig
	}
	// 检查交易费是否小于最低值
	realFee := int64(txSize/1000+1) * minFee
	return realFee, nil
}

// SetRealFee 设置交易真实费用
func (tx *Transaction) SetRealFee(minFee int64) error {
	if tx.Fee == 0 {
		fee, err := tx.GetRealFee(minFee)
		if err != nil {
			return err
		}
		tx.Fee = fee
	}
	return nil
}

// ExpireBound 交易过期边界值
var ExpireBound int64 = 1000000000 // 交易过期分界线，小于expireBound比较height，大于expireBound比较blockTime

// IsExpire 交易是否过期
func (tx *Transaction) IsExpire(cfg *Chain33Config, height, blocktime int64) bool {
	group, _ := tx.GetTxGroup()
	if group == nil {
		return tx.isExpire(cfg, height, blocktime)
	}
	return group.IsExpire(cfg, height, blocktime)
}

// GetTxFee 获取交易的费用，区分单笔交易和交易组
func (tx *Transaction) GetTxFee() int64 {
	group, _ := tx.GetTxGroup()
	if group == nil || len(group.GetTxs()) == 0 {
		return tx.Fee
	}
	return group.Txs[0].Fee
}

// From 交易from地址
func (tx *Transaction) From() string {
	return address.PubKeyToAddr(ExtractAddressID(tx.GetSignature().GetTy()),
		tx.GetSignature().GetPubkey())
}

// 检查交易是否过期，过期返回true，未过期返回false
func (tx *Transaction) isExpire(cfg *Chain33Config, height, blocktime int64) bool {
	valid := tx.Expire
	// Expire为0，返回false
	if valid == 0 {
		return false
	}
	if valid <= ExpireBound {
		//Expire小于1e9，为height valid > height 未过期返回false else true过期
		return valid <= height
	}
	//EnableTxHeight 选项开启, 并且符合条件
	if txHeight := GetTxHeight(cfg, valid, height); txHeight > 0 {
		if txHeight-LowAllowPackHeight <= height && height <= txHeight+HighAllowPackHeight {
			return false
		}
		return true
	}
	// Expire大于1e9，为blockTime  valid > blocktime返回false 未过期 else true过期
	return valid <= blocktime
}

// GetTxHeight 获取交易高度
func GetTxHeight(cfg *Chain33Config, valid int64, height int64) int64 {
	if cfg.IsPara() {
		return -1
	}
	if cfg.IsEnableFork(height, "ForkTxHeight", cfg.IsEnable("TxHeight")) && valid > TxHeightFlag {
		return valid - TxHeightFlag
	}
	return -1
}

// JSON Transaction交易信息转成json结构体
func (tx *Transaction) JSON() string {
	type transaction struct {
		Hash      string     `json:"hash,omitempty"`
		Execer    string     `json:"execer,omitempty"`
		Payload   string     `json:"payload,omitempty"`
		Signature *Signature `json:"signature,omitempty"`
		Fee       int64      `json:"fee,omitempty"`
		Expire    int64      `json:"expire,omitempty"`
		// 随机ID，可以防止payload 相同的时候，交易重复
		Nonce int64 `json:"nonce,omitempty"`
		// 对方地址，如果没有对方地址，可以为空
		To         string `json:"to,omitempty"`
		GroupCount int32  `json:"groupCount,omitempty"`
		Header     string `json:"header,omitempty"`
		Next       string `json:"next,omitempty"`
		ChainID    int32  `json:"chainID,omitempty"`
	}

	newtx := &transaction{}
	newtx.Hash = hex.EncodeToString(tx.Hash())
	newtx.Execer = string(tx.Execer)
	newtx.Payload = hex.EncodeToString(tx.Payload)
	newtx.Signature = tx.Signature
	newtx.Fee = tx.Fee
	newtx.Expire = tx.Expire
	newtx.Nonce = tx.Nonce
	newtx.To = tx.To
	newtx.GroupCount = tx.GroupCount
	newtx.Header = hex.EncodeToString(tx.Header)
	newtx.Next = hex.EncodeToString(tx.Next)
	newtx.ChainID = tx.ChainID

	data, err := json.MarshalIndent(newtx, "", "\t")
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// Amount 解析tx的payload获取amount值
func (tx *Transaction) Amount() (int64, error) {
	// TODO 原来有很多执行器 在这里没有代码， 用默认 0, nil 先
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return 0, nil
	}
	return exec.Amount(tx)
}

// Assets  获取交易中的资产
func (tx *Transaction) Assets() ([]*Asset, error) {
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return nil, nil
	}
	return exec.GetAssets(tx)
}

// GetRealToAddr 解析tx的payload获取real to值
func (tx *Transaction) GetRealToAddr() string {
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return tx.To
	}
	return exec.GetRealToAddr(tx)
}

// GetViewFromToAddr 解析tx的payload获取view from to 值
func (tx *Transaction) GetViewFromToAddr() (string, string) {
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return tx.From(), tx.To
	}
	return exec.GetViewFromToAddr(tx)
}

// ActionName 获取tx交易的Actionname
func (tx *Transaction) ActionName() string {
	execName := string(tx.Execer)
	exec := LoadExecutorType(execName)
	if exec == nil {
		//action name 不会影响系统状态，主要是做显示使用
		realname := GetRealExecName(tx.Execer)
		exec = LoadExecutorType(string(realname))
		if exec == nil {
			return "unknown"
		}
	}
	return exec.ActionName(tx)
}

// IsWithdraw 判断交易是withdraw交易，需要做from和to地址的swap，方便上层客户理解
func (tx *Transaction) IsWithdraw(coinExec string) bool {
	if bytes.Equal(tx.GetExecer(), []byte(coinExec)) || bytes.Equal(tx.GetExecer(), bToken) {
		if tx.ActionName() == withdraw {
			return true
		}
	}
	return false
}

// ParseExpire parse expire to int from during or height
func ParseExpire(expire string) (int64, error) {
	if len(expire) == 0 {
		return 0, ErrInvalidParam
	}
	if expire[0] == 'H' && expire[1] == ':' {
		txHeight, err := strconv.ParseInt(expire[2:], 10, 64)
		if err != nil {
			return 0, err
		}
		if txHeight <= 0 {
			//fmt.Printf("txHeight should be grate to 0")
			return 0, ErrHeightLessZero
		}
		if txHeight+TxHeightFlag < txHeight {
			return 0, ErrHeightOverflow
		}

		return txHeight + TxHeightFlag, nil
	}

	blockHeight, err := strconv.ParseInt(expire, 10, 64)
	if err == nil {
		return blockHeight, nil
	}

	expireTime, err := time.ParseDuration(expire)
	if err == nil {
		return int64(expireTime), nil
	}

	return 0, err
}

// CalcTxShortHash 取txhash的前指定字节，目前默认5
func CalcTxShortHash(hash []byte) string {
	if len(hash) >= 5 {
		return hex.EncodeToString(hash[0:5])
	}
	return ""
}

// TransactionSort 对主链以及平行链交易分类
// 构造一个map用于临时存储各个子链的交易, 按照title分类，主链交易的title设置成main
// 并对map按照title进行排序，不然每次遍历map顺序会不一致
func TransactionSort(rawtxs []*Transaction) []*Transaction {
	txMap := make(map[string]*Transactions)

	for _, tx := range rawtxs {
		title, isPara := GetParaExecTitleName(string(tx.Execer))
		if !isPara {
			title = MainChainName
		}
		if txMap[title] != nil {
			txMap[title].Txs = append(txMap[title].Txs, tx)
		} else {
			var temptxs Transactions
			temptxs.Txs = append(temptxs.Txs, tx)
			txMap[title] = &temptxs
		}
	}

	//需要按照title排序，不然每次遍历的map的顺序会不一致
	var newMp = make([]string, 0)
	for k := range txMap {
		newMp = append(newMp, k)
	}
	sort.Strings(newMp)

	var txs Transactions
	for _, v := range newMp {
		txs.Txs = append(txs.Txs, txMap[v].GetTxs()...)
	}
	return txs.GetTxs()
}

var (
	// 用于交易结构protobuf编码buffer
	txProtoBufferPool = &sync.Pool{
		New: func() interface{} {
			return proto.NewBuffer(make([]byte, 0, 256))
		},
	}
	// 交易结构内存池
	txPool = &sync.Pool{
		New: func() interface{} { return &Transaction{} },
	}
)

// NewTx new tx object
func NewTx() *Transaction {
	return txPool.Get().(*Transaction)
}

// FreeTx free tx object
func FreeTx(txs ...*Transaction) {
	for _, tx := range txs {
		resetTx(tx)
		txPool.Put(tx)
	}
}

// reset tx value
// protobuf内部Reset方法含有unsafe不稳定操作, 可能导致交易哈希计算不稳定
func resetTx(tx *Transaction) {
	*tx = Transaction{}
}

// Hash 交易的hash不包含header的值，引入tx group的概念后，做了修改
func (tx *Transaction) Hash() []byte {
	copytx := CloneTx(tx)
	copytx.Signature = nil
	copytx.Header = nil
	buffer := txProtoBufferPool.Get().(*proto.Buffer)
	data := EncodeWithBuffer(copytx, buffer)
	FreeTx(copytx)
	hash := common.Sha256(data)
	buffer.Reset()
	txProtoBufferPool.Put(buffer)
	return hash
}

// FullHash 交易的fullhash包含交易的签名信息
func (tx *Transaction) FullHash() []byte {
	copytx := tx.Clone()
	buffer := txProtoBufferPool.Get().(*proto.Buffer)
	data := EncodeWithBuffer(copytx, buffer)
	FreeTx(copytx)
	hash := common.Sha256(data)
	buffer.Reset()
	txProtoBufferPool.Put(buffer)
	return hash
}

// TxGroup 交易组的接口，Transactions 和 Transaction 都符合这个接口
type TxGroup interface {
	Tx() *Transaction
	GetTxGroup() (*Transactions, error)
	CheckSign(blockHeight int64) bool
}

// CloneTx clone tx
// 这里要避免用 tmp := *tx 这样就会读 可能被 proto 其他线程修改的 size 字段
// proto buffer 字段发生更改之后，一定要修改这里，否则可能引起严重的bug
func CloneTx(tx *Transaction) *Transaction {
	copytx := NewTx()
	copytx.Execer = tx.Execer
	copytx.Payload = tx.Payload
	copytx.Signature = tx.Signature
	copytx.Fee = tx.Fee
	copytx.Expire = tx.Expire
	copytx.Nonce = tx.Nonce
	copytx.To = tx.To
	copytx.GroupCount = tx.GroupCount
	copytx.Header = tx.Header
	copytx.Next = tx.Next
	copytx.ChainID = tx.ChainID
	return copytx
}

// Clone copytx := proto.Clone(tx).(*Transaction) too slow
func (tx *Transaction) Clone() *Transaction {
	if tx == nil {
		return nil
	}
	tmp := CloneTx(tx)
	tmp.Signature = tx.Signature.Clone()
	return tmp
}

// GetEthTxHash 获取eth 兼容交易的交易哈希
func (tx *Transaction) GetEthTxHash() []byte {
	if !IsEthSignID(tx.GetSignature().GetTy()) {
		return nil
	}
	var evmaction EVMContractAction4Chain33
	err := Decode(tx.GetPayload(), &evmaction)
	if err == nil {
		var etx etypes.Transaction
		etxBytes, err := common.FromHex(evmaction.GetNote())
		if err == nil && etx.UnmarshalBinary(etxBytes) == nil {
			return etx.Hash().Bytes()
		}
	}

	return nil
}

// cloneTxs  拷贝 txs
func cloneTxs(b []*Transaction) []*Transaction {
	if b == nil {
		return nil
	}
	txs := make([]*Transaction, len(b))
	for i := 0; i < len(b); i++ {
		txs[i] = b[i].Clone()
	}
	return txs
}
