// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	"github.com/hashicorp/golang-lru"

	"strconv"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
)

var (
	bCoins   = []byte("coins")
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

//TxCacheGet 某些交易的cache 加入缓存中，防止重复进行解析或者计算
func TxCacheGet(tx *Transaction) (*TransactionCache, bool) {
	txc, ok := txCache.Get(tx)
	if !ok {
		return nil, ok
	}
	return txc.(*TransactionCache), ok
}

//TxCacheSet 设置 cache
func TxCacheSet(tx *Transaction, txc *TransactionCache) {
	if txc == nil {
		txCache.Remove(tx)
		return
	}
	txCache.Add(tx, txc)
}

// CreateTxGroup 创建组交易
func CreateTxGroup(txs []*Transaction) (*Transactions, error) {
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
		realfee, err := txs[i].GetRealFee(GInt("MinFee"))
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

//Tx 这比用于检查的交易，包含了所有的交易。
//主要是为了兼容原来的设计
func (txgroup *Transactions) Tx() *Transaction {
	if len(txgroup.GetTxs()) < 2 {
		return nil
	}
	headtx := txgroup.GetTxs()[0]
	//不会影响原来的tx
	copytx := *headtx
	data := Encode(txgroup)
	//放到header中不影响交易的Hash
	copytx.Header = data
	return &copytx
}

//GetTxGroup 获取交易组
func (txgroup *Transactions) GetTxGroup() *Transactions {
	return txgroup
}

//SignN 对交易组的第n笔交易签名
func (txgroup *Transactions) SignN(n int, ty int32, priv crypto.PrivKey) error {
	if n >= len(txgroup.GetTxs()) {
		return ErrIndex
	}
	txgroup.GetTxs()[n].Sign(ty, priv)
	return nil
}

//CheckSign 检测交易组的签名
func (txgroup *Transactions) CheckSign() bool {
	txs := txgroup.Txs
	for i := 0; i < len(txs); i++ {
		if !txs[i].checkSign() {
			return false
		}
	}
	return true
}

//IsExpire 交易是否过期
func (txgroup *Transactions) IsExpire(height, blocktime int64) bool {
	txs := txgroup.Txs
	for i := 0; i < len(txs); i++ {
		if txs[i].isExpire(height, blocktime) {
			return true
		}
	}
	return false
}

//Check height == 0 的时候，不做检查
func (txgroup *Transactions) Check(height, minfee, maxFee int64) error {
	txs := txgroup.Txs
	if len(txs) < 2 {
		return ErrTxGroupCountLessThanTwo
	}
	para := make(map[string]bool)
	for i := 0; i < len(txs); i++ {
		if txs[i] == nil {
			return ErrTxGroupEmpty
		}
		err := txs[i].check(height, 0, maxFee)
		if err != nil {
			return err
		}
		name := string(txs[i].Execer)
		if IsParaExecName(name) {
			para[name] = true
		}
	}
	//txgroup 只允许一条平行链的交易
	if IsEnableFork(height, "ForkV24TxGroupPara", EnableTxGroupParaFork) {
		if len(para) > 1 {
			tlog.Info("txgroup has multi para transaction")
			return ErrTxGroupParaCount
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
	if txs[0].Fee > maxFee && maxFee > 0 && IsFork(height, "ForkBlockCheck") {
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

//TransactionCache 交易缓存结构
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

//NewTransactionCache new交易缓存
func NewTransactionCache(tx *Transaction) *TransactionCache {
	return &TransactionCache{Transaction: tx}
}

//Hash 交易hash
func (tx *TransactionCache) Hash() []byte {
	if tx.hash == nil {
		tx.hash = tx.Transaction.Hash()
	}
	return tx.hash
}

//SetPayloadValue 设置payload 的cache
func (tx *TransactionCache) SetPayloadValue(plname string, payload reflect.Value, plerr error) {
	tx.payload = payload
	tx.plerr = plerr
	tx.plname = plname
}

//GetPayloadValue 设置payload 的cache
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

//Size 交易缓存的大小
func (tx *TransactionCache) Size() int {
	if tx.size == 0 {
		tx.size = Size(tx.Tx())
	}
	return tx.size
}

//Tx 交易缓存中tx信息
func (tx *TransactionCache) Tx() *Transaction {
	return tx.Transaction
}

//Check 交易缓存中交易组合费用的检测
func (tx *TransactionCache) Check(height, minfee, maxFee int64) error {
	if !tx.checked {
		tx.checked = true
		txs, err := tx.GetTxGroup()
		if err != nil {
			tx.checkok = err
			return err
		}
		if txs == nil {
			tx.checkok = tx.check(height, minfee, maxFee)
		} else {
			tx.checkok = txs.Check(height, minfee, maxFee)
		}
	}
	return tx.checkok
}

//GetTxGroup 获取交易组
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

//CheckSign 检测签名
func (tx *TransactionCache) CheckSign() bool {
	if tx.signok == 0 {
		tx.signok = 2
		group, err := tx.GetTxGroup()
		if err != nil {
			return false
		}
		if group == nil {
			//非group，简单校验签名
			if ok := tx.checkSign(); ok {
				tx.signok = 1
			}
		} else {
			if ok := group.CheckSign(); ok {
				tx.signok = 1
			}
		}
	}
	return tx.signok == 1
}

//TxsToCache 缓存交易信息
func TxsToCache(txs []*Transaction) (caches []*TransactionCache) {
	caches = make([]*TransactionCache, len(txs))
	for i := 0; i < len(caches); i++ {
		caches[i] = NewTransactionCache(txs[i])
	}
	return caches
}

//CacheToTxs 从缓存中获取交易信息
func CacheToTxs(caches []*TransactionCache) (txs []*Transaction) {
	txs = make([]*Transaction, len(caches))
	for i := 0; i < len(caches); i++ {
		txs[i] = caches[i].Tx()
	}
	return txs
}

//HashSign hash 不包含签名，用户通过修改签名无法重新发送交易
func (tx *Transaction) HashSign() []byte {
	copytx := *tx
	copytx.Signature = nil
	data := Encode(&copytx)
	return common.Sha256(data)
}

//Tx 交易详情
func (tx *Transaction) Tx() *Transaction {
	return tx
}

//GetTxGroup 交易组装成交易组格式
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

//Hash 交易的hash不包含header的值，引入tx group的概念后，做了修改
func (tx *Transaction) Hash() []byte {
	copytx := clone(tx)
	copytx.Signature = nil
	copytx.Header = nil
	data := Encode(copytx)
	return common.Sha256(data)
}

//clone copytx := proto.Clone(tx).(*Transaction) too slow
func clone(tx *Transaction) *Transaction {
	copytx := &Transaction{}
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
	return copytx
}

//Size 交易大小
func (tx *Transaction) Size() int {
	return Size(tx)
}

//Sign 交易签名
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

//CheckSign tx 有些时候是一个交易组
func (tx *Transaction) CheckSign() bool {
	return tx.checkSign()
}

//txgroup 的情况
func (tx *Transaction) checkSign() bool {
	copytx := *tx
	copytx.Signature = nil
	data := Encode(&copytx)
	if tx.GetSignature() == nil {
		return false
	}
	return CheckSign(data, string(tx.Execer), tx.GetSignature())
}

//Check 交易检测
func (tx *Transaction) Check(height, minfee, maxFee int64) error {
	group, err := tx.GetTxGroup()
	if err != nil {
		return err
	}
	if group == nil {
		return tx.check(height, minfee, maxFee)
	}
	return group.Check(height, minfee, maxFee)
}

func (tx *Transaction) check(height, minfee, maxFee int64) error {
	txSize := Size(tx)
	if txSize > int(MaxTxSize) {
		return ErrTxMsgSizeTooBig
	}
	if minfee == 0 {
		return nil
	}
	// 检查交易费是否小于最低值
	realFee := int64(txSize/1000+1) * minfee
	if tx.Fee < realFee {
		return ErrTxFeeTooLow
	}
	if tx.Fee > maxFee && maxFee > 0 && IsFork(height, "ForkBlockCheck") {
		return ErrTxFeeTooHigh
	}
	return nil
}

//SetExpire 设置交易过期时间
func (tx *Transaction) SetExpire(expire time.Duration) {
	//Txheight处理
	if IsEnable("TxHeight") && int64(expire) > TxHeightFlag {
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

//GetRealFee 获取交易真实费用
func (tx *Transaction) GetRealFee(minFee int64) (int64, error) {
	txSize := Size(tx)
	//如果签名为空，那么加上签名的空间
	if tx.Signature == nil {
		txSize += 300
	}
	if txSize > int(MaxTxSize) {
		return 0, ErrTxMsgSizeTooBig
	}
	// 检查交易费是否小于最低值
	realFee := int64(txSize/1000+1) * minFee
	return realFee, nil
}

//SetRealFee 设置交易真实费用
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

//ExpireBound 交易过期边界值
var ExpireBound int64 = 1000000000 // 交易过期分界线，小于expireBound比较height，大于expireBound比较blockTime

//IsExpire 交易是否过期
func (tx *Transaction) IsExpire(height, blocktime int64) bool {
	group, _ := tx.GetTxGroup()
	if group == nil {
		return tx.isExpire(height, blocktime)
	}
	return group.IsExpire(height, blocktime)
}

//From 交易from地址
func (tx *Transaction) From() string {
	return address.PubKeyToAddr(tx.GetSignature().GetPubkey())
}

//检查交易是否过期，过期返回true，未过期返回false
func (tx *Transaction) isExpire(height, blocktime int64) bool {
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
	if txHeight := GetTxHeight(valid, height); txHeight > 0 {
		if txHeight-LowAllowPackHeight <= height && height <= txHeight+HighAllowPackHeight {
			return false
		}
		return true
	}
	// Expire大于1e9，为blockTime  valid > blocktime返回false 未过期 else true过期
	return valid <= blocktime
}

//GetTxHeight 获取交易高度
func GetTxHeight(valid int64, height int64) int64 {
	if IsEnableFork(height, "ForkTxHeight", IsEnable("TxHeight")) && valid > TxHeightFlag {
		return valid - TxHeightFlag
	}
	return -1
}

//JSON Transaction交易信息转成json结构体
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
	data, err := json.MarshalIndent(newtx, "", "\t")
	if err != nil {
		return err.Error()
	}
	return string(data)
}

//Amount 解析tx的payload获取amount值
func (tx *Transaction) Amount() (int64, error) {
	// TODO 原来有很多执行器 在这里没有代码， 用默认 0, nil 先
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return 0, nil
	}
	return exec.Amount(tx)
}

//Assets  获取交易中的资产
func (tx *Transaction) Assets() ([]*Asset, error) {
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return nil, nil
	}
	return exec.GetAssets(tx)
}

//GetRealToAddr 解析tx的payload获取real to值
func (tx *Transaction) GetRealToAddr() string {
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return tx.To
	}
	return exec.GetRealToAddr(tx)
}

//GetViewFromToAddr 解析tx的payload获取view from to 值
func (tx *Transaction) GetViewFromToAddr() (string, string) {
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return tx.From(), tx.To
	}
	return exec.GetViewFromToAddr(tx)
}

//ActionName 获取tx交易的Actionname
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

//IsWithdraw 判断交易是withdraw交易，需要做from和to地址的swap，方便上层客户理解
func (tx *Transaction) IsWithdraw() bool {
	if bytes.Equal(tx.GetExecer(), bCoins) || bytes.Equal(tx.GetExecer(), bToken) {
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
