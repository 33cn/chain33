package ticket

//database opeartion for execs ticket
import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/big"

	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

var genesisKey = []byte("mavl-acc-genesis")
var addrSeed = []byte("address seed bytes for public key")

type Ticket struct {
	types.Ticket
}

func NewTicket(id, minerAddress, returnWallet string, blocktime int64) *Ticket {
	t := &Ticket{}
	t.TicketId = id
	t.MinerAddress = minerAddress
	t.ReturnAddress = returnWallet
	t.CreateTime = blocktime
	t.Status = 1
	t.IsGenesis = true
	return &t
}

func (t *Ticket) GetReceiptLog() *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = types.TyLogNewTicket
	r := &types.ReceiptNewTicket{}
	r.TicketId = t.TicketId
	log.Log = types.Encode(r)
	return log
}

func (t *Ticket) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(t.Ticket)
	kvset = append(kvset, &types.KeyValue{TicketKey(t.TicketId), value})
	return kvset
}

//address to save key
func TicketKey(id string) (key []byte) {
	key = append(key, []byte("mavl-ticket-")...)
	key = append(key, []byte(id)...)
	return key
}
