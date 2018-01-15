package types

import (
	"runtime"
	"sync/atomic"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	_ "code.aliyun.com/chain33/chain33/common/crypto/ed25519"
	_ "code.aliyun.com/chain33/chain33/common/crypto/secp256k1"
	proto "github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
)

var tlog = log.New("module", "types")

type Message proto.Message

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

var expireBound int64 = 1000000000 // 交易过期分界线，小于expireBound比较height，大于expireBound比较blockTime

//检查交易是否过期，过期返回true，未过期返回false
func (tx *Transaction) IsExpire(height, blocktime int64) bool {
	valid := tx.Expire
	// Expire为0，返回true
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
	ok := multiCoreSign(block.Txs, cpu)
	return ok
}

func multiCoreSign(txs []*Transaction, n int) bool {
	if len(txs) == 0 {
		return true
	}
	taskch := make(chan *Transaction)
	resultch := make(chan bool)
	done := make(chan struct{}, n)
	var exitflag int64 = 0
	var errflag int64 = 0
	for i := 0; i < n; i++ {
		go func(i int) {
			for taskitem := range taskch {
				//time.Sleep(time.Second)
				if !isExit(exitflag) {
					resultch <- taskitem.CheckSign()
				} else {
					break
				}
			}
			done <- struct{}{}
		}(i)
	}
	//发送任务
	go func() {
		for i := 0; i < len(txs); i++ {
			if !isExit(exitflag) {
				taskch <- txs[i]
			} else {
				break
			}
		}
	}()
	systemexit := make(chan int, 1)
	//接收任务回报

	go func() {
		for i := 0; i < len(txs); i++ {
			select {
			case result := <-resultch:
				if !result {
					atomic.StoreInt64(&exitflag, 1)
					atomic.StoreInt64(&errflag, 1)
					closeCh(taskch)
					return
				}
			case <-systemexit:
				return
			}
		}
		closeCh(taskch)
	}()
	for i := 0; i < n; i++ {
		<-done
	}
	systemexit <- 1
	return atomic.LoadInt64(&errflag) == 0
}

func isExit(exitflag int64) bool {
	return atomic.LoadInt64(&exitflag) == 1
}

func closeCh(ch chan *Transaction) {
	for {
		select {
		case <-ch:
		default:
			close(ch)
			return
		}
	}
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
