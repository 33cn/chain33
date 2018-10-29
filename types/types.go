package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types/jsonpb"

	_ "gitlab.33.cn/chain33/chain33/system/crypto/init"
)

var tlog = log.New("module", "types")

const Size_1K_shiftlen uint = 10

type Message proto.Message

type Query4Cli struct {
	Execer   string      `json:"execer"`
	FuncName string      `json:"funcName"`
	Payload  interface{} `json:"payload"`
}

//交易组的接口，Transactions 和 Transaction 都符合这个接口
type TxGroup interface {
	Tx() *Transaction
	GetTxGroup() (*Transactions, error)
	CheckSign() bool
}

func ExecName(name string) string {
	if IsParaExecName(name) {
		return name
	}
	return ExecNamePrefix + name
}

//默认的allow 规则->根据 GetRealExecName 来判断
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
	if bytes.Contains(name, slash) {
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

func GetParaExec(execer []byte) []byte {
	//必须是平行链
	if !IsPara() {
		return execer
	}
	//必须是相同的平行链
	if !strings.HasPrefix(string(execer), GetTitle()) {
		return execer
	}
	return execer[len(GetTitle()):]
}

func getParaExecName(execer []byte) []byte {
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

func GetRealExecName(execer []byte) []byte {
	//平行链执行器，获取真实执行器的规则
	execer = getParaExecName(execer)
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

func Encode(data proto.Message) []byte {
	b, err := proto.Marshal(data)
	if err != nil {
		panic(err)
	}
	return b
}

func Size(data proto.Message) int {
	return proto.Size(data)
}

func Decode(data []byte, msg proto.Message) error {
	return proto.Unmarshal(data, msg)
}

func JsonToPB(data []byte, msg proto.Message) error {
	return jsonpb.Unmarshal(bytes.NewReader(data), msg)
}

func (leafnode *LeafNode) Hash() []byte {
	data, err := proto.Marshal(leafnode)
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}

func (innernode *InnerNode) Hash() []byte {
	rightHash := innernode.RightHash
	leftHash := innernode.LeftHash
	hashLen := len(common.Hash{})
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

func NewErrReceipt(err error) *Receipt {
	berr := err.Error()
	errlog := &ReceiptLog{TyLogErr, []byte(berr)}
	return &Receipt{ExecErr, nil, []*ReceiptLog{errlog}}
}

func CheckAmount(amount int64) bool {
	if amount <= 0 || amount >= MaxCoin {
		return false
	}
	return true
}

func GetEventName(event int) string {
	name, ok := eventName[event]
	if ok {
		return name
	}
	return "unknow-event"
}

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

var ConfigPrefix = "mavl-config-"

func ConfigKey(key string) string {
	return fmt.Sprintf("%s%s", ConfigPrefix, key)
}

var ManagePrefix = "mavl-"

func ManageKey(key string) string {
	return fmt.Sprintf("%s-%s", ManagePrefix+"manage", key)
}

func ManaeKeyWithHeigh(key string, height int64) string {
	if height >= ForkV13ExecKey {
		return ManageKey(key)
	} else {
		return ConfigKey(key)
	}
}

type ReceiptDataResult struct {
	Ty     int32               `json:"ty"`
	TyName string              `json:"tyname"`
	Logs   []*ReceiptLogResult `json:"logs"`
}

type ReceiptLogResult struct {
	Ty     int32       `json:"ty"`
	TyName string      `json:"tyname"`
	Log    interface{} `json:"log"`
	RawLog string      `json:"rawlog"`
}

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

		logIns, err = logType.Decode(lLog)
		lTy = logType.Name()

		result.Logs = append(result.Logs, &ReceiptLogResult{Ty: l.Ty, TyName: lTy, Log: logIns, RawLog: common.ToHex(l.GetLog())})
	}
	return result, nil
}

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

func (t *ReplyGetTotalCoins) IterateRangeByStateHash(key, value []byte) bool {
	fmt.Println("ReplyGetTotalCoins.IterateRangeByStateHash", "key", string(key))
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

type ParaCrossTx interface {
	IsParaCrossTx() bool
}

func PBToJson(r Message) ([]byte, error) {
	encode := &jsonpb.Marshaler{EmitDefaults: true}
	var buf bytes.Buffer
	if err := encode.Marshal(&buf, r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

//判断所有的空值
func IsNil(a interface{}) bool {
	defer func() { recover() }()
	return a == nil || reflect.ValueOf(a).IsNil()
}

//空指针或者接口
func IsNilP(a interface{}) bool {
	if a == nil {
		return true
	}
	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		return v.IsNil()
	}
	return false
}

func MustDecode(data []byte, v interface{}) {
	err := json.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}
}
