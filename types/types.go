// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types 实现了chain33基础结构体、接口、常量等的定义
package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types/jsonpb"
	"github.com/golang/protobuf/proto"

	// 注册system的crypto 加密算法
	_ "github.com/33cn/chain33/system/crypto/init"
)

var tlog = log.New("module", "types")

// Size1Kshiftlen tx消息大小1k
const Size1Kshiftlen uint = 10

// Message 声明proto.Message
type Message proto.Message

//TxGroup 交易组的接口，Transactions 和 Transaction 都符合这个接口
type TxGroup interface {
	Tx() *Transaction
	GetTxGroup() (*Transactions, error)
	CheckSign() bool
}

//ExecName  执行器name
func (c *Chain33Config) ExecName(name string) string {
	if len(name) > 1 && name[0] == '#' {
		return name[1:]
	}
	if IsParaExecName(name) {
		return name
	}
	if c.IsPara() {
		return c.GetTitle() + name
	}
	return name
}

//IsAllowExecName 默认的allow 规则->根据 GetRealExecName 来判断
//name 必须大于3 小于 100
func IsAllowExecName(name []byte, execer []byte) bool {
	// name长度不能超过系统限制
	if len(name) > address.MaxExecNameLength || len(execer) > address.MaxExecNameLength {
		return false
	}
	if len(name) < 3 || len(execer) < 3 {
		return false
	}
	// name中不允许有 "-"
	if bytes.Contains(name, slash) || bytes.Contains(name, sharp) {
		return false
	}
	if !bytes.Equal(name, execer) && !bytes.Equal(name, GetRealExecName(execer)) {
		return false
	}
	if bytes.HasPrefix(name, UserKey) {
		return true
	}
	for i := range AllowUserExec {
		if bytes.Equal(AllowUserExec[i], name) {
			return true
		}
	}
	return false
}

var bytesExec = []byte("exec-")
var commonPrefix = []byte("mavl-")

//GetExecKey  获取执行器key
func GetExecKey(key []byte) (string, bool) {
	n := 0
	start := 0
	end := 0
	for i := len(commonPrefix); i < len(key); i++ {
		if key[i] == '-' {
			n = n + 1
			if n == 2 {
				start = i + 1
			}
			if n == 3 {
				end = i
				break
			}
		}
	}
	if start > 0 && end > 0 {
		if bytes.Equal(key[start:end+1], bytesExec) {
			//find addr
			start = end + 1
			for k := end; k < len(key); k++ {
				if key[k] == ':' { //end+1
					end = k
					return string(key[start:end]), true
				}
			}
		}
	}
	return "", false
}

//FindExecer  查找执行器
func FindExecer(key []byte) (execer []byte, err error) {
	if !bytes.HasPrefix(key, commonPrefix) {
		return nil, ErrMavlKeyNotStartWithMavl
	}
	for i := len(commonPrefix); i < len(key); i++ {
		if key[i] == '-' {
			return key[len(commonPrefix):i], nil
		}
	}
	return nil, ErrNoExecerInMavlKey
}

//GetParaExec  获取平行链执行
func (c *Chain33Config) GetParaExec(execer []byte) []byte {
	//必须是平行链
	if !c.IsPara() {
		return execer
	}
	//必须是相同的平行链
	if !strings.HasPrefix(string(execer), c.GetTitle()) {
		return execer
	}
	return execer[len(c.GetTitle()):]
}

//GetParaExecName 获取平行链上的执行器
func GetParaExecName(execer []byte) []byte {
	if !bytes.HasPrefix(execer, ParaKey) {
		return execer
	}
	count := 0
	for i := 0; i < len(execer); i++ {
		if execer[i] == '.' {
			count++
		}
		if count == 3 && i < (len(execer)-1) {
			newexec := execer[i+1:]
			return newexec
		}
	}
	return execer
}

//GetRealExecName  获取真实的执行器name
func GetRealExecName(execer []byte) []byte {
	//平行链执行器，获取真实执行器的规则
	execer = GetParaExecName(execer)
	//平行链嵌套平行链是不被允许的
	if bytes.HasPrefix(execer, ParaKey) {
		return execer
	}
	if bytes.HasPrefix(execer, UserKey) {
		//不是user.p. 的情况, 而是user. 的情况
		count := 0
		index := 0
		for i := 0; i < len(execer); i++ {
			if execer[i] == '.' {
				count++
			}
			index = i
			if count == 2 {
				index--
				break
			}
		}
		e := execer[len(UserKey) : index+1]
		if len(e) > 0 {
			return e
		}
	}
	return execer
}

//Encode  编码
func Encode(data proto.Message) []byte {
	b, err := proto.Marshal(data)
	if err != nil {
		panic(err)
	}
	return b
}

//Size  消息大小
func Size(data proto.Message) int {
	return proto.Size(data)
}

//Decode  解码
func Decode(data []byte, msg proto.Message) error {
	return proto.Unmarshal(data, msg)
}

//JSONToPB  JSON格式转换成protobuffer格式
func JSONToPB(data []byte, msg proto.Message) error {
	return jsonpb.Unmarshal(bytes.NewReader(data), msg)
}

//JSONToPBUTF8 默认解码utf8的字符串成bytes
func JSONToPBUTF8(data []byte, msg proto.Message) error {
	decode := &jsonpb.Unmarshaler{EnableUTF8BytesToString: true}
	return decode.Unmarshal(bytes.NewReader(data), msg)
}

//Hash  计算叶子节点的hash
func (leafnode *LeafNode) Hash() []byte {
	data, err := proto.Marshal(leafnode)
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}

var sha256Len = 32

//Hash  计算中间节点的hash
func (innernode *InnerNode) Hash() []byte {
	rightHash := innernode.RightHash
	leftHash := innernode.LeftHash
	hashLen := sha256Len
	if len(innernode.RightHash) > hashLen {
		innernode.RightHash = innernode.RightHash[len(innernode.RightHash)-hashLen:]
	}
	if len(innernode.LeftHash) > hashLen {
		innernode.LeftHash = innernode.LeftHash[len(innernode.LeftHash)-hashLen:]
	}
	data, err := proto.Marshal(innernode)
	if err != nil {
		panic(err)
	}
	innernode.RightHash = rightHash
	innernode.LeftHash = leftHash
	return common.Sha256(data)
}

//NewErrReceipt  new一个新的Receipt
func NewErrReceipt(err error) *Receipt {
	berr := err.Error()
	errlog := &ReceiptLog{Ty: TyLogErr, Log: []byte(berr)}
	return &Receipt{Ty: ExecErr, KV: nil, Logs: []*ReceiptLog{errlog}}
}

//CheckAmount  检测转账金额
func CheckAmount(amount int64) bool {
	if amount <= 0 || amount >= MaxCoin {
		return false
	}
	return true
}

//GetEventName  获取时间name通过事件id
func GetEventName(event int) string {
	name, ok := eventName[event]
	if ok {
		return name
	}
	return "unknow-event"
}

//GetSignName  获取签名类型
func GetSignName(execer string, signType int) string {
	//优先加载执行器的签名类型
	if execer != "" {
		exec := LoadExecutorType(execer)
		if exec != nil {
			name, err := exec.GetCryptoDriver(signType)
			if err == nil {
				return name
			}
		}
	}
	//加载系统执行器的签名类型
	return crypto.GetName(signType)
}

//GetSignType  获取签名类型
func GetSignType(execer string, name string) int {
	//优先加载执行器的签名类型
	if execer != "" {
		exec := LoadExecutorType(execer)
		if exec != nil {
			ty, err := exec.GetCryptoType(name)
			if err == nil {
				return ty
			}
		}
	}
	//加载系统执行器的签名类型
	return crypto.GetType(name)
}

// ConfigPrefix 配置前缀key
var ConfigPrefix = "mavl-config-"

// ConfigKey 原来实现有bug， 但生成的key在状态树里， 不可修改
// mavl-config–{key}  key 前面两个-
func ConfigKey(key string) string {
	return fmt.Sprintf("%s-%s", ConfigPrefix, key)
}

// ManagePrefix 超级管理员账户配置前缀key
var ManagePrefix = "mavl-"

//ManageKey 超级管理员账户key
func ManageKey(key string) string {
	return fmt.Sprintf("%s-%s", ManagePrefix+"manage", key)
}

//ManaeKeyWithHeigh 超级管理员账户key
func (c *Chain33Config) ManaeKeyWithHeigh(key string, height int64) string {
	if c.IsFork(height, "ForkExecKey") {
		return ManageKey(key)
	}
	return ConfigKey(key)
}

//ReceiptDataResult 回执数据
type ReceiptDataResult struct {
	Ty     int32               `json:"ty"`
	TyName string              `json:"tyname"`
	Logs   []*ReceiptLogResult `json:"logs"`
}

//ReceiptLogResult 回执log数据
type ReceiptLogResult struct {
	Ty     int32       `json:"ty"`
	TyName string      `json:"tyname"`
	Log    interface{} `json:"log"`
	RawLog string      `json:"rawlog"`
}

//DecodeReceiptLog 编码回执数据
func (r *ReceiptData) DecodeReceiptLog(execer []byte) (*ReceiptDataResult, error) {
	result := &ReceiptDataResult{Ty: r.GetTy()}
	switch r.Ty {
	case 0:
		result.TyName = "ExecErr"
	case 1:
		result.TyName = "ExecPack"
	case 2:
		result.TyName = "ExecOk"
	default:
		return nil, ErrLogType
	}

	logs := r.GetLogs()
	for _, l := range logs {
		var lTy string
		var logIns interface{}
		lLog, err := hex.DecodeString(common.ToHex(l.GetLog())[2:])
		if err != nil {
			return nil, err
		}

		logType := LoadLog(execer, int64(l.Ty))
		if logType == nil {
			//tlog.Error("DecodeReceiptLog:", "Faile to decodeLog with type value logtype", l.Ty)
			return nil, ErrLogType
		}

		logIns, _ = logType.Decode(lLog)
		lTy = logType.Name()

		result.Logs = append(result.Logs, &ReceiptLogResult{Ty: l.Ty, TyName: lTy, Log: logIns, RawLog: common.ToHex(l.GetLog())})
	}
	return result, nil
}

//OutputReceiptDetails 输出回执数据详情
func (r *ReceiptData) OutputReceiptDetails(execer []byte, logger log.Logger) {
	rds, err := r.DecodeReceiptLog(execer)
	if err == nil {
		logger.Debug("receipt decode", "receipt data", rds)
		for _, rdl := range rds.Logs {
			logger.Debug("receipt log", "log", rdl)
		}
	} else {
		logger.Error("decodelogerr", "err", err)
	}
}

//IterateRangeByStateHash 迭代查找
func (t *ReplyGetTotalCoins) IterateRangeByStateHash(key, value []byte) bool {
	tlog.Debug("ReplyGetTotalCoins.IterateRangeByStateHash", "key", string(key))
	var acc Account
	err := Decode(value, &acc)
	if err != nil {
		tlog.Error("ReplyGetTotalCoins.IterateRangeByStateHash", "err", err)
		return true
	}
	//tlog.Info("acc:", "value", acc)
	if t.Num >= t.Count {
		t.NextKey = key
		return true
	}
	t.Num++
	t.Amount += acc.Balance
	return false
}

// GetTxTimeInterval 获取交易有效期
func GetTxTimeInterval() time.Duration {
	return time.Second * 120
}

// ParaCrossTx 平行跨链交易
type ParaCrossTx interface {
	IsParaCrossTx() bool
}

// PBToJSON 消息类型转换
func PBToJSON(r Message) ([]byte, error) {
	encode := &jsonpb.Marshaler{EmitDefaults: true}
	var buf bytes.Buffer
	if err := encode.Marshal(&buf, r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PBToJSONUTF8 消息类型转换
func PBToJSONUTF8(r Message) ([]byte, error) {
	encode := &jsonpb.Marshaler{EmitDefaults: true, EnableUTF8BytesToString: true}
	var buf bytes.Buffer
	if err := encode.Marshal(&buf, r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

//MustPBToJSON panic when error
func MustPBToJSON(req Message) []byte {
	data, err := PBToJSON(req)
	if err != nil {
		panic(err)
	}
	return data
}

// MustDecode 数据是否已经编码
func MustDecode(data []byte, v interface{}) {
	if data == nil {
		return
	}
	err := json.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}
}

// AddItem 添加item
func (t *ReplyGetExecBalance) AddItem(execAddr, value []byte) {
	var acc Account
	err := Decode(value, &acc)
	if err != nil {
		tlog.Error("ReplyGetExecBalance.AddItem", "err", err)
		return
	}
	tlog.Info("acc:", "value", acc)
	t.Amount += acc.Balance
	t.Amount += acc.Frozen

	t.AmountActive += acc.Balance
	t.AmountFrozen += acc.Frozen

	item := &ExecBalanceItem{ExecAddr: execAddr, Frozen: acc.Frozen, Active: acc.Balance}
	t.Items = append(t.Items, item)
}

//Clone  克隆
func Clone(data proto.Message) proto.Message {
	return proto.Clone(data)
}

//Clone 添加一个浅拷贝函数
func (sig *Signature) Clone() *Signature {
	if sig == nil {
		return nil
	}
	return &Signature{
		Ty:        sig.Ty,
		Pubkey:    sig.Pubkey,
		Signature: sig.Signature,
	}
}

//这里要避免用 tmp := *tx 这样就会读 可能被 proto 其他线程修改的 size 字段
//proto buffer 字段发生更改之后，一定要修改这里，否则可能引起严重的bug
func cloneTx(tx *Transaction) *Transaction {
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

//Clone copytx := proto.Clone(tx).(*Transaction) too slow
func (tx *Transaction) Clone() *Transaction {
	if tx == nil {
		return nil
	}
	tmp := cloneTx(tx)
	tmp.Signature = tx.Signature.Clone()
	return tmp
}

//Clone 浅拷贝： BlockDetail
func (b *BlockDetail) Clone() *BlockDetail {
	if b == nil {
		return nil
	}
	return &BlockDetail{
		Block:          b.Block.Clone(),
		Receipts:       cloneReceipts(b.Receipts),
		KV:             cloneKVList(b.KV),
		PrevStatusHash: b.PrevStatusHash,
	}
}

//Clone 浅拷贝ReceiptData
func (r *ReceiptData) Clone() *ReceiptData {
	if r == nil {
		return nil
	}
	return &ReceiptData{
		Ty:   r.Ty,
		Logs: cloneReceiptLogs(r.Logs),
	}
}

//Clone 浅拷贝 receiptLog
func (r *ReceiptLog) Clone() *ReceiptLog {
	if r == nil {
		return nil
	}
	return &ReceiptLog{
		Ty:  r.Ty,
		Log: r.Log,
	}
}

//Clone KeyValue
func (kv *KeyValue) Clone() *KeyValue {
	if kv == nil {
		return nil
	}
	return &KeyValue{
		Key:   kv.Key,
		Value: kv.Value,
	}
}

//Clone Block 浅拷贝(所有的types.Message 进行了拷贝)
func (b *Block) Clone() *Block {
	if b == nil {
		return nil
	}
	return &Block{
		Version:    b.Version,
		ParentHash: b.ParentHash,
		TxHash:     b.TxHash,
		StateHash:  b.StateHash,
		Height:     b.Height,
		BlockTime:  b.BlockTime,
		Difficulty: b.Difficulty,
		MainHash:   b.MainHash,
		MainHeight: b.MainHeight,
		Signature:  b.Signature.Clone(),
		Txs:        cloneTxs(b.Txs),
	}
}

//Clone BlockBody 浅拷贝(所有的types.Message 进行了拷贝)
func (b *BlockBody) Clone() *BlockBody {
	if b == nil {
		return nil
	}
	return &BlockBody{
		Txs:        cloneTxs(b.Txs),
		Receipts:   cloneReceipts(b.Receipts),
		MainHash:   b.MainHash,
		MainHeight: b.MainHeight,
		Hash:       b.Hash,
		Height:     b.Height,
	}
}

//cloneReceipts 浅拷贝交易回报
func cloneReceipts(b []*ReceiptData) []*ReceiptData {
	if b == nil {
		return nil
	}
	rs := make([]*ReceiptData, len(b))
	for i := 0; i < len(b); i++ {
		rs[i] = b[i].Clone()
	}
	return rs
}

//cloneReceiptLogs 浅拷贝 ReceiptLogs
func cloneReceiptLogs(b []*ReceiptLog) []*ReceiptLog {
	if b == nil {
		return nil
	}
	rs := make([]*ReceiptLog, len(b))
	for i := 0; i < len(b); i++ {
		rs[i] = b[i].Clone()
	}
	return rs
}

//cloneTxs  拷贝 txs
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

//cloneKVList 拷贝kv 列表
func cloneKVList(b []*KeyValue) []*KeyValue {
	if b == nil {
		return nil
	}
	kv := make([]*KeyValue, len(b))
	for i := 0; i < len(b); i++ {
		kv[i] = b[i].Clone()
	}
	return kv
}

//Hash  计算hash
func (hashes *ReplyHashes) Hash() []byte {
	data, err := proto.Marshal(hashes)
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}