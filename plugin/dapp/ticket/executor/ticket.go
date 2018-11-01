package executor

/*
coins 是一个货币的exec。内置货币的执行器。

主要提供两种操作：

EventTransfer -> 转移资产
*/

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.ticket")
var driverName = "ticket"

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Ticket{}))
}

func Init(name string, sub []byte) {
	drivers.Register(GetName(), newTicket, types.GetDappFork(driverName, "Enable"))
}

func GetName() string {
	return newTicket().GetName()
}

type Ticket struct {
	drivers.DriverBase
}

func newTicket() drivers.Driver {
	t := &Ticket{}
	t.SetChild(t)
	t.SetExecutorType(types.LoadExecutorType(driverName))
	return t
}

func (t *Ticket) GetDriverName() string {
	return driverName
}

func (t *Ticket) saveTicketBind(b *ty.ReceiptTicketBind) (kvs []*types.KeyValue) {
	//解除原来的绑定
	if len(b.OldMinerAddress) > 0 {
		kv := &types.KeyValue{
			Key:   calcBindMinerKey(b.OldMinerAddress, b.ReturnAddress),
			Value: nil,
		}
		//tlog.Warn("tb:del", "key", string(kv.Key))
		kvs = append(kvs, kv)
	}

	kv := &types.KeyValue{calcBindReturnKey(b.ReturnAddress), []byte(b.NewMinerAddress)}
	//tlog.Warn("tb:add", "key", string(kv.Key), "value", string(kv.Value))
	kvs = append(kvs, kv)
	kv = &types.KeyValue{
		Key:   calcBindMinerKey(b.GetNewMinerAddress(), b.ReturnAddress),
		Value: []byte(b.ReturnAddress),
	}
	//tlog.Warn("tb:add", "key", string(kv.Key), "value", string(kv.Value))
	kvs = append(kvs, kv)
	return kvs
}

func (t *Ticket) delTicketBind(b *ty.ReceiptTicketBind) (kvs []*types.KeyValue) {
	//被取消了，刚好操作反
	kv := &types.KeyValue{
		Key:   calcBindMinerKey(b.NewMinerAddress, b.ReturnAddress),
		Value: nil,
	}
	kvs = append(kvs, kv)
	if len(b.OldMinerAddress) > 0 {
		//恢复旧的绑定
		kv := &types.KeyValue{calcBindReturnKey(b.ReturnAddress), []byte(b.OldMinerAddress)}
		kvs = append(kvs, kv)
		kv = &types.KeyValue{
			Key:   calcBindMinerKey(b.OldMinerAddress, b.ReturnAddress),
			Value: []byte(b.ReturnAddress),
		}
		kvs = append(kvs, kv)
	} else {
		//删除旧的数据
		kv := &types.KeyValue{calcBindReturnKey(b.ReturnAddress), nil}
		kvs = append(kvs, kv)
	}
	return kvs
}

func (t *Ticket) saveTicket(ticketlog *ty.ReceiptTicket) (kvs []*types.KeyValue) {
	if ticketlog.PrevStatus > 0 {
		kv := delticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, addticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.Status))
	return kvs
}

func (t *Ticket) delTicket(ticketlog *ty.ReceiptTicket) (kvs []*types.KeyValue) {
	if ticketlog.PrevStatus > 0 {
		kv := addticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, delticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.Status))
	return kvs
}

func calcTicketKey(addr string, ticketID string, status int32) []byte {
	key := fmt.Sprintf("LODB-ticket-tl:%s:%d:%s", addr, status, ticketID)
	return []byte(key)
}

func calcBindReturnKey(returnAddress string) []byte {
	key := fmt.Sprintf("LODB-ticket-bind:%s", returnAddress)
	return []byte(key)
}

func calcBindMinerKey(minerAddress string, returnAddress string) []byte {
	key := fmt.Sprintf("LODB-ticket-miner:%s:%s", minerAddress, returnAddress)
	return []byte(key)
}

func calcBindMinerKeyPrefix(minerAddress string) []byte {
	key := fmt.Sprintf("LODB-ticket-miner:%s", minerAddress)
	return []byte(key)
}

func calcTicketPrefix(addr string, status int32) []byte {
	key := fmt.Sprintf("LODB-ticket-tl:%s:%d", addr, status)
	return []byte(key)
}

func addticket(addr string, ticketID string, status int32) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcTicketKey(addr, ticketID, status)
	kv.Value = []byte(ticketID)
	return kv
}

func delticket(addr string, ticketID string, status int32) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcTicketKey(addr, ticketID, status)
	kv.Value = nil
	return kv
}

func (t *Ticket) IsFriend(myexec, writekey []byte, tx *types.Transaction) bool {
	clog.Error("ticket  IsFriend", "myex", string(myexec), "writekey", string(writekey))
	//不允许平行链
	return false
}

func (t *Ticket) CheckTx(tx *types.Transaction, index int) error {
	//index == -1 only when check in mempool
	if index == -1 {
		var action ty.TicketAction
		err := types.Decode(tx.Payload, &action)
		if err != nil {
			return err
		}
		if action.Ty == ty.TicketActionMiner && action.GetMiner() != nil {
			return ty.ErrMinerTx
		}
	}
	return nil
}
