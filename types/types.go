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
	"math"
	"strings"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	log "github.com/33cn/chain33/common/log/log15"

	// 注册默认的btc地址模式
	_ "github.com/33cn/chain33/system/address/btc"
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

// ExecName  执行器name
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

// GetExecName 根据paraName 获取exec name
func GetExecName(exec, paraName string) string {
	if len(exec) > 1 && exec[0] == '#' {
		return exec[1:]
	}
	if IsParaExecName(exec) {
		return exec
	}
	if len(paraName) > 0 {
		return paraName + exec
	}
	return exec
}

// IsAllowExecName 默认的allow 规则->根据 GetRealExecName 来判断
// name 必须大于3 小于 100
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

// GetExecKey  获取执行器key
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

// FindExecer  查找执行器
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

// GetParaExec  获取平行链执行
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

// GetParaExecName 获取平行链上的执行器
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

// GetRealExecName  获取真实的执行器name
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

// EncodeWithBuffer encode with input buffer
func EncodeWithBuffer(data proto.Message, buf *proto.Buffer) []byte {
	return encodeProto(data, buf)
}

// Encode encode msg
func Encode(data proto.Message) []byte {
	return encodeProto(data, nil)
}

// protobuf V2(golang/protobuf 1.4.0+)
// 版本默认Marshal接口不保证序列化结果一致性, 需要主动设置相关标志
// 即使设置了deterministic标志, protobuf官方也不保证后续版本升级以及跨语言的序列化结果一致
// 即该标志只能保证当前版本具备一致性
func encodeProto(data proto.Message, buf *proto.Buffer) []byte {

	if buf == nil {
		var b [0]byte
		buf = proto.NewBuffer(b[:])
	}
	// 设置确定性编码
	buf.SetDeterministic(true)
	err := buf.Marshal(data)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// Size  消息大小
func Size(data proto.Message) int {
	return proto.Size(data)
}

// Decode  解码
func Decode(data []byte, msg proto.Message) error {
	return proto.Unmarshal(data, msg)
}

// JSONToPB  JSON格式转换成protobuffer格式
func JSONToPB(data []byte, msg proto.Message) error {
	return jsonpb.Unmarshal(bytes.NewReader(data), msg)
}

// JSONToPBUTF8 默认解码utf8的字符串成bytes
func JSONToPBUTF8(data []byte, msg proto.Message) error {
	decode := &jsonpb.Unmarshaler{EnableUTF8BytesToString: true}
	return decode.Unmarshal(bytes.NewReader(data), msg)
}

// Hash  计算叶子节点的hash
func (leafnode *LeafNode) Hash() []byte {
	data := Encode(leafnode)
	return common.Sha256(data)
}

var sha256Len = 32

// Hash  计算中间节点的hash
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
	data := Encode(innernode)
	innernode.RightHash = rightHash
	innernode.LeftHash = leftHash
	return common.Sha256(data)
}

// NewErrReceipt  new一个新的Receipt
func NewErrReceipt(err error) *Receipt {
	berr := err.Error()
	errlog := &ReceiptLog{Ty: TyLogErr, Log: []byte(berr)}
	return &Receipt{Ty: ExecErr, KV: nil, Logs: []*ReceiptLog{errlog}}
}

// CheckAmount  检测转账金额
func CheckAmount(amount, coinPrecision int64) bool {
	if amount <= 0 || amount >= MaxCoin*coinPrecision {
		return false
	}
	return true
}

// GetEventName  获取时间name通过事件id
func GetEventName(event int) string {
	name, ok := eventName[event]
	if ok {
		return name
	}
	return "unknow-event"
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

// ManageKey 超级管理员账户key
func ManageKey(key string) string {
	return fmt.Sprintf("%s-%s", ManagePrefix+"manage", key)
}

// ReceiptDataResult 回执数据
type ReceiptDataResult struct {
	Ty     int32               `json:"ty"`
	TyName string              `json:"tyname"`
	Logs   []*ReceiptLogResult `json:"logs"`
}

// ReceiptLogResult 回执log数据
type ReceiptLogResult struct {
	Ty     int32       `json:"ty"`
	TyName string      `json:"tyname"`
	Log    interface{} `json:"log"`
	RawLog string      `json:"rawlog"`
}

// DecodeReceiptLog 编码回执数据
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

// OutputReceiptDetails 输出回执数据详情
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

// IterateRangeByStateHash 迭代查找
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

// MustPBToJSON panic when error
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
	tlog.Info("acc:", "value", &acc)
	t.Amount += acc.Balance
	t.Amount += acc.Frozen

	t.AmountActive += acc.Balance
	t.AmountFrozen += acc.Frozen

	item := &ExecBalanceItem{ExecAddr: execAddr, Frozen: acc.Frozen, Active: acc.Balance}
	t.Items = append(t.Items, item)
}

// CloneAccount copy account
func CloneAccount(acc *Account) *Account {
	return &Account{
		Currency: acc.Currency,
		Balance:  acc.Balance,
		Frozen:   acc.Frozen,
		Addr:     acc.Addr,
	}
}

// Clone  克隆
func Clone(data proto.Message) proto.Message {
	return proto.Clone(data)
}

// Clone 添加一个浅拷贝函数
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

// Clone 浅拷贝： BlockDetail
func (b *BlockDetail) Clone() *BlockDetail {
	if b == nil {
		return nil
	}
	return &BlockDetail{
		Block:          b.Block.Clone(),
		Receipts:       CloneReceipts(b.Receipts),
		KV:             cloneKVList(b.KV),
		PrevStatusHash: b.PrevStatusHash,
	}
}

// Clone 浅拷贝ReceiptData
func (r *ReceiptData) Clone() *ReceiptData {
	if r == nil {
		return nil
	}
	return &ReceiptData{
		Ty:   r.Ty,
		Logs: cloneReceiptLogs(r.Logs),
	}
}

// Clone 浅拷贝 receiptLog
func (r *ReceiptLog) Clone() *ReceiptLog {
	if r == nil {
		return nil
	}
	return &ReceiptLog{
		Ty:  r.Ty,
		Log: r.Log,
	}
}

// Clone KeyValue
func (kv *KeyValue) Clone() *KeyValue {
	if kv == nil {
		return nil
	}
	return &KeyValue{
		Key:   kv.Key,
		Value: kv.Value,
	}
}

// Clone Block 浅拷贝(所有的types.Message 进行了拷贝)
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

// Clone BlockBody 浅拷贝(所有的types.Message 进行了拷贝)
func (b *BlockBody) Clone() *BlockBody {
	if b == nil {
		return nil
	}
	return &BlockBody{
		Txs:        cloneTxs(b.Txs),
		Receipts:   CloneReceipts(b.Receipts),
		MainHash:   b.MainHash,
		MainHeight: b.MainHeight,
		Hash:       b.Hash,
		Height:     b.Height,
	}
}

// CloneReceipts 浅拷贝交易回报
func CloneReceipts(b []*ReceiptData) []*ReceiptData {
	if b == nil {
		return nil
	}
	rs := make([]*ReceiptData, len(b))
	for i := 0; i < len(b); i++ {
		rs[i] = b[i].Clone()
	}
	return rs
}

// cloneReceiptLogs 浅拷贝 ReceiptLogs
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

// cloneKVList 拷贝kv 列表
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

// Bytes2Str 高效字节数组转字符串
// 相比普通直接转化，性能提升35倍，提升程度和转换的byte长度线性相关，且不存在内存开销
// 需要注意b的修改会导致最终string的变更，比较适合临时变量转换，不适合
func Bytes2Str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Str2Bytes 高效字符串转字节数组
// 相比普通直接转化，性能提升13倍， 提升程度和转换的byte长度线性相关，且不存在内存开销
// 需要注意不能修改转换后的byte数组，本质上是修改了底层string，将会panic
func Str2Bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// Hash  计算hash
func (hashes *ReplyHashes) Hash() []byte {
	data := Encode(hashes)
	return common.Sha256(data)
}

// FormatAmount2FloatDisplay 将传输、计算的amount值按精度格式化成浮点显示值，支持精度可配置
// strconv.FormatFloat(float64(amount/types.Int1E4)/types.Float1E4, 'f', 4, 64) 的方法在amount很大的时候会丢失精度
// 比如在9001234567812345678时候，float64 最大精度只能在900123456781234的大小，decimal处理可以完整保持精度
// 在coinPrecision支持可配时候，对不同精度统一处理,而不是限定在1e4
// round:是否需要对小数后四位值圆整，以912345678为例，912345678/1e8=9.1235, 非圆整例子:(912345678/1e4)/float(1e4)
func FormatAmount2FloatDisplay(amount, coinPrecision int64, round bool) string {
	n := int64(math.Log10(float64(coinPrecision)))
	//小数左移n位，0保持不变
	d := decimal.NewFromInt(amount).Shift(int32(-n))

	//coinPrecision:5~8,
	//v=99.12345678  => 99.1234,需要先truncate掉，不然5678会round到前一位也就是99.1235
	if n > ShowPrecisionNum {
		//有些需要圆整上来的,比如交易费,0.12345678,圆整为0.1235
		if round {
			return d.StringFixedBank(int32(ShowPrecisionNum))
		}
		//有些需要直接truncate掉
		return d.Truncate(int32(ShowPrecisionNum)).StringFixedBank(int32(ShowPrecisionNum))

	}

	return d.StringFixedBank(int32(n))
}

// FormatAmount2FixPrecisionDisplay 将传输、计算的amount值按配置精度格式化成浮点显示值，不设缺省精度
func FormatAmount2FixPrecisionDisplay(amount, coinPrecision int64) string {
	n := int64(math.Log10(float64(coinPrecision)))
	//小数左移n位，0保持不变
	d := decimal.NewFromInt(amount).Shift(int32(-n))

	return d.StringFixedBank(int32(n))
}

// FormatFloatDisplay2Value 将显示、输入的float amount值按精度格式化成计算值，小数点后精度只保留4位
// 浮点数算上浮点能精确表达最大16位长的数字(1234567.12345678),考虑到16个9会被表示为1e16,这里限制最多15个字符
// 本函数然后小数位只精确到4位，后面补0
func FormatFloatDisplay2Value(amount float64, coinPrecision int64) (int64, error) {
	f := decimal.NewFromFloat(amount)
	strVal := f.String()
	if len(strVal) <= 0 {
		return 0, errors.Wrapf(ErrInvalidParam, "input=%f", amount)
	}
	//小数位输入不能超过配置小数位精度
	coinPrecisionNum := int(math.Log10(float64(coinPrecision)))
	i := strings.Index(strVal, ".")
	if i > 0 && len(strVal)-i-1 > coinPrecisionNum {
		return 0, errors.Wrapf(ErrInvalidParam, "input decimalpoint num=%d great than config coinPrecision=%d", len(strVal)-i-1, coinPrecisionNum)
	}

	//因为float精度原因，限制输入浮点数最多字符数
	if len(strVal) > MaxFloatCharNum {
		return 0, errors.Wrapf(ErrInvalidParam, "input=%f,len=%d great than %v", amount, len(strVal), MaxFloatCharNum)
	}

	//如果配置精度超过4位，小数位只精确到后4位, 例如1.23456789 ->123450000
	if int64(coinPrecisionNum) > ShowPrecisionNum {
		return f.Truncate(int32(ShowPrecisionNum)).Shift(int32(coinPrecisionNum)).IntPart(), nil
	}
	//如果配置精度小于4位，乘精度
	return f.Shift(int32(coinPrecisionNum)).IntPart(), nil

}
