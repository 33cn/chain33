package types

import (
	"runtime"
	"strings"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	_ "code.aliyun.com/chain33/chain33/common/crypto/ed25519"
	_ "code.aliyun.com/chain33/chain33/common/crypto/secp256k1"
	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
)

var tlog = log.New("module", "types")

type Message proto.Message

func isAllowExecName(name string) bool {
	if strings.HasPrefix(name, "user.") {
		return true
	}
	for i := range AllowUserExec {
		if AllowUserExec[i] == name {
			return true
		}
	}
	return false
}

type TransactionCache struct {
	*Transaction
	hash []byte
	size int
}

func NewTransactionCache(tx *Transaction) *TransactionCache {
	return &TransactionCache{tx, tx.Hash(), tx.Size()}
}

func (tx *TransactionCache) Hash() []byte {
	return tx.hash
}

func (tx *TransactionCache) Size() int {
	return tx.size
}

func (tx *TransactionCache) Tx() *Transaction {
	return tx.Transaction
}

func TxsToCache(txs []*Transaction) (caches []*TransactionCache) {
	caches = make([]*TransactionCache, len(txs), len(txs))
	for i := 0; i < len(caches); i++ {
		caches[i] = NewTransactionCache(txs[i])
	}
	return caches
}

func CacheToTxs(caches []*TransactionCache) (txs []*Transaction) {
	txs = make([]*Transaction, len(caches), len(caches))
	for i := 0; i < len(caches); i++ {
		txs[i] = caches[i].Tx()
	}
	return txs
}

//hash 不包含签名，用户通过修改签名无法重新发送交易
func (tx *Transaction) Hash() []byte {
	copytx := *tx
	copytx.Signature = nil
	data := Encode(&copytx)
	return common.Sha256(data)
}

func (tx *Transaction) Size() int {
	return Size(tx)
}

func (tx *Transaction) Sign(ty int32, priv crypto.PrivKey) {
	tx.Signature = nil
	data := Encode(tx)
	pub := priv.PubKey()
	sign := priv.Sign(data)
	tx.Signature = &Signature{ty, pub.Bytes(), sign.Bytes()}
}

func (tx *Transaction) CheckSign() bool {
	copytx := *tx
	copytx.Signature = nil
	data := Encode(&copytx)
	if tx.GetSignature() == nil {
		return false
	}
	return CheckSign(data, tx.GetSignature())
}

func (tx *Transaction) Check(minfee int64) error {
	if !isAllowExecName(string(tx.Execer)) {
		return ErrExecNameNotAllow
	}
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
	return nil
}

func (tx *Transaction) SetExpire(expire time.Duration) {
	if int64(expire) > expireBound {
		//用秒数来表示的时间
		tx.Expire = time.Now().Unix() + int64(expire/time.Second)
	} else {
		tx.Expire = int64(expire)
	}
}

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

var expireBound int64 = 1000000000 // 交易过期分界线，小于expireBound比较height，大于expireBound比较blockTime

//检查交易是否过期，过期返回true，未过期返回false
func (tx *Transaction) IsExpire(height, blocktime int64) bool {
	valid := tx.Expire
	// Expire为0，返回false
	if valid == 0 {
		return false
	}

	if valid <= expireBound {
		//Expire小于1e9，为height
		if valid > height { // 未过期
			return false
		} else { // 过期
			return true
		}
	} else {
		// Expire大于1e9，为blockTime
		if valid > blocktime { // 未过期
			return false
		} else { // 过期
			return true
		}
	}
}

//解析tx的payload获取amount值
func (tx *Transaction) Amount() (int64, error) {

	if "coins" == string(tx.Execer) {
		var action CoinsAction
		err := Decode(tx.GetPayload(), &action)
		if err != nil {
			return 0, ErrDecode
		}
		if action.Ty == CoinsActionTransfer && action.GetTransfer() != nil {
			transfer := action.GetTransfer()
			return transfer.Amount, nil
		} else if action.Ty == CoinsActionGenesis && action.GetGenesis() != nil {
			gen := action.GetGenesis()
			return gen.Amount, nil
		} else if action.Ty == CoinsActionWithdraw && action.GetWithdraw() != nil {
			transfer := action.GetWithdraw()
			return transfer.Amount, nil
		}
	} else if "ticket" == string(tx.Execer) {
		var action TicketAction
		err := Decode(tx.GetPayload(), &action)
		if err != nil {
			return 0, ErrDecode
		}
		if action.Ty == TicketActionMiner && action.GetMiner() != nil {
			ticketMiner := action.GetMiner()
			return ticketMiner.Reward, nil
		}
	}
	return 0, nil
}

//获取tx交易的Actionname
func (tx *Transaction) ActionName() string {
	if "coins" == string(tx.Execer) {
		var action CoinsAction
		err := Decode(tx.Payload, &action)
		if err != nil {
			return "unknow-err"
		}
		if action.Ty == CoinsActionTransfer && action.GetTransfer() != nil {
			return "transfer"
		} else if action.Ty == CoinsActionWithdraw && action.GetWithdraw() != nil {
			return "withdraw"
		} else if action.Ty == CoinsActionGenesis && action.GetGenesis() != nil {
			return "genesis"
		} else {
			return "unknow"
		}
	} else if "ticket" == string(tx.Execer) {
		var action TicketAction
		err := Decode(tx.Payload, &action)
		if err != nil {
			return "unknow-err"
		}
		if action.Ty == TicketActionGenesis && action.GetGenesis() != nil {
			return "genesis"
		} else if action.Ty == TicketActionOpen && action.GetTopen() != nil {
			return "open"
		} else if action.Ty == TicketActionClose && action.GetTclose() != nil {
			return "close"
		} else if action.Ty == TicketActionMiner && action.GetMiner() != nil {
			return "miner"
		} else if action.Ty == TicketActionBind && action.GetTbind() != nil {
			return "bindminer"
		}
		return "unknow"
	} else if "none" == string(tx.Execer) {
		return "none"
	} else if "hashlock" == string(tx.Execer) {
		var action HashlockAction
		err := Decode(tx.Payload, &action)
		if err != nil {
			return "unknow-err"
		}
		if action.Ty == HashlockActionLock && action.GetHlock() != nil {
			return "lock"
		} else if action.Ty == HashlockActionUnlock && action.GetHunlock() != nil {
			return "unlock"
		} else if action.Ty == HashlockActionSend && action.GetHsend() != nil {
			return "send"
		} else {
			return "unknow"
		}
	} else if "retrieve" == string(tx.Execer) {
		var action RetrieveAction
		err := Decode(tx.Payload, &action)
		if err != nil {
			return "unknow-err"
		}
		if action.Ty == RetrievePre && action.GetPreRet() != nil {
			return "prepare"
		} else if action.Ty == RetrievePerf && action.GetPerfRet() != nil {
			return "perform"
		} else if action.Ty == RetrieveBackup && action.GetBackup() != nil {
			return "backup"
		} else if action.Ty == RetrieveCancel && action.GetCancel() != nil {
			return "cancel"
		} else {
			return "unknow"
		}
	}
	return "unknow"
}

func (block *Block) Hash() []byte {
	head := &Header{}
	head.Version = block.Version
	head.ParentHash = block.ParentHash
	head.TxHash = block.TxHash
	head.BlockTime = block.BlockTime
	head.Height = block.Height
	data, err := proto.Marshal(head)
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}

func (block *Block) GetHeader() *Header {
	head := &Header{}
	head.Version = block.Version
	head.ParentHash = block.ParentHash
	head.TxHash = block.TxHash
	head.BlockTime = block.BlockTime
	head.Height = block.Height
	return head
}

func (block *Block) CheckSign() bool {
	//检查区块的签名
	if block.Signature != nil {
		hash := block.Hash()
		sign := block.GetSignature()
		if !CheckSign(hash, sign) {
			return false
		}
	}
	//检查交易的签名
	cpu := runtime.NumCPU()
	ok := checkAll(block.Txs, cpu)
	return ok
}

func gen(done <-chan struct{}, task []*Transaction) <-chan *Transaction {
	ch := make(chan *Transaction)
	go func() {
		defer func() {
			close(ch)
		}()
		for i := 0; i < len(task); i++ {
			select {
			case ch <- task[i]:
			case <-done:
				return
			}
		}
	}()
	return ch
}

type result struct {
	isok bool
}

func check(data *Transaction) bool {
	return data.CheckSign()
}

func checksign(done <-chan struct{}, taskes <-chan *Transaction, c chan<- result) {
	for task := range taskes {
		select {
		case c <- result{check(task)}:
		case <-done:
			return
		}
	}
}

func checkAll(task []*Transaction, n int) bool {
	done := make(chan struct{})
	defer close(done)

	taskes := gen(done, task)

	// Start a fixed number of goroutines to read and digest files.
	c := make(chan result) // HLc
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			checksign(done, taskes, c) // HLc
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c) // HLc
	}()
	// End of pipeline. OMIT
	for r := range c {
		if r.isok == false {
			return false
		}
	}
	return true
}

func CheckSign(data []byte, sign *Signature) bool {
	c, err := crypto.New(GetSignatureTypeName(int(sign.Ty)))
	if err != nil {
		return false
	}
	pub, err := c.PubKeyFromBytes(sign.Pubkey)
	if err != nil {
		return false
	}
	signbytes, err := c.SignatureFromBytes(sign.Signature)
	if err != nil {
		return false
	}
	return pub.VerifyBytes(data, signbytes)
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

func (leafnode *LeafNode) Hash() []byte {
	data, err := proto.Marshal(leafnode)
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}

func (innernode *InnerNode) Hash() []byte {
	data, err := proto.Marshal(innernode)
	if err != nil {
		panic(err)
	}
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

func GetSignatureTypeName(signType int) string {
	if signType == 1 {
		return "secp256k1"
	} else if signType == 2 {
		return "ed25519"
	} else if signType == 3 {
		return "sm2"
	} else {
		return "unknow"
	}
}

func (t *ReplyGetTotalCoins) IterateRangeByStateHash(key, value []byte) bool {
        tlog.Info("ReplyGetTotalCoins.IterateRangeByStateHash", "key", string(key), "value", string(value))
        var acc Account
        err := Decode(value, &acc)
        if err != nil {
                tlog.Error("ReplyGetTotalCoins.IterateRangeByStateHash", "err", err)
                return true
        }
        //tlog.Info("acc:", "value", acc)
        t.Count += 1
        t.Amount += acc.Balance
/*
        if t.Count == t.Sum {
                return true
        }
*/
        return false
}
