package ticket

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
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.ticket")

func init() {
	t := newTicket()
	drivers.Register(t.GetName(), t, 0)
}

type Ticket struct {
	drivers.DriverBase
}

func newTicket() *Ticket {
	t := &Ticket{}
	t.SetChild(t)
	return t
}

func (t *Ticket) GetName() string {
	return "ticket"
}

func (t *Ticket) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.TicketAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	clog.Info("exec ticket tx=", "tx=", action)
	actiondb := NewAction(t, tx)
	if action.Ty == types.TicketActionGenesis && action.GetGenesis() != nil {
		genesis := action.GetGenesis()
		if genesis.Count <= 0 {
			return nil, types.ErrTicketCount
		}
		//new ticks
		return actiondb.GenesisInit(genesis)
	} else if action.Ty == types.TicketActionOpen && action.GetTopen() != nil {
		topen := action.GetTopen()
		if topen.Count <= 0 {
			tlog.Error("topen ", "value", topen)
			return nil, types.ErrTicketCount
		}
		return actiondb.TicketOpen(topen)
	} else if action.Ty == types.TicketActionBind && action.GetTbind() != nil {
		tbind := action.GetTbind()
		return actiondb.TicketBind(tbind)
	} else if action.Ty == types.TicketActionClose && action.GetTclose() != nil {
		tclose := action.GetTclose()
		return actiondb.TicketClose(tclose)
	} else if action.Ty == types.TicketActionMiner && action.GetMiner() != nil {
		miner := action.GetMiner()
		return actiondb.TicketMiner(miner, index)
	}
	//return error
	return nil, types.ErrActionNotSupport
}

func (t *Ticket) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		//这三个是ticket 的log
		if item.Ty == types.TyLogNewTicket || item.Ty == types.TyLogMinerTicket || item.Ty == types.TyLogCloseTicket {
			var ticketlog types.ReceiptTicket
			err := types.Decode(item.Log, &ticketlog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveTicket(&ticketlog)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogTicketBind {
			var ticketlog types.ReceiptTicketBind
			err := types.Decode(item.Log, &ticketlog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveTicketBind(&ticketlog)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

func (t *Ticket) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		//这三个是ticket 的log
		if item.Ty == types.TyLogNewTicket || item.Ty == types.TyLogMinerTicket || item.Ty == types.TyLogCloseTicket {
			var ticketlog types.ReceiptTicket
			err := types.Decode(item.Log, &ticketlog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.delTicket(&ticketlog)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogTicketBind {
			var ticketlog types.ReceiptTicketBind
			err := types.Decode(item.Log, &ticketlog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.delTicketBind(&ticketlog)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

func (t *Ticket) saveTicketBind(b *types.ReceiptTicketBind) (kvs []*types.KeyValue) {
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

func (t *Ticket) delTicketBind(b *types.ReceiptTicketBind) (kvs []*types.KeyValue) {
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

func (t *Ticket) saveTicket(ticketlog *types.ReceiptTicket) (kvs []*types.KeyValue) {
	if ticketlog.PrevStatus > 0 {
		kv := delticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, addticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.Status))
	return kvs
}

func (t *Ticket) delTicket(ticketlog *types.ReceiptTicket) (kvs []*types.KeyValue) {
	if ticketlog.PrevStatus > 0 {
		kv := addticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, delticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.Status))
	return kvs
}

func (t *Ticket) Query(funcname string, params []byte) (types.Message, error) {
	if funcname == "TicketInfos" {
		var info types.TicketInfos
		err := types.Decode(params, &info)
		if err != nil {
			return nil, err
		}
		return Infos(t.GetDB(), &info)
	} else if funcname == "TicketList" {
		var l types.TicketList
		err := types.Decode(params, &l)
		if err != nil {
			return nil, err
		}
		return List(t.GetQueryDB(), t.GetDB(), &l)
	} else if funcname == "MinerAddress" {
		var reqaddr types.ReqString
		err := types.Decode(params, &reqaddr)
		if err != nil {
			return nil, err
		}
		value, err := t.GetQueryDB().Get(calcBindReturnKey(reqaddr.Data))
		if value == nil || err != nil {
			return nil, types.ErrNotFound
		}
		return &types.ReplyString{string(value)}, nil
	} else if funcname == "MinerSourceList" {
		var reqaddr types.ReqString
		err := types.Decode(params, &reqaddr)
		if err != nil {
			return nil, err
		}
		key := calcBindMinerKeyPrefix(reqaddr.Data)
		list := dbm.NewListHelper(t.GetQueryDB())
		values := list.List(key, nil, 0, 1)
		if len(values) == 0 {
			return nil, types.ErrNotFound
		}
		reply := &types.ReplyStrings{}
		for _, value := range values {
			reply.Datas = append(reply.Datas, string(value))
		}
		return reply, nil
	}
	return nil, types.ErrActionNotSupport
}

func calcTicketKey(addr string, ticketID string, status int32) []byte {
	key := fmt.Sprintf("ticket-tl:%s:%d:%s", addr, status, ticketID)
	return []byte(key)
}

func calcBindReturnKey(returnAddress string) []byte {
	key := fmt.Sprintf("ticket-bind:%s", returnAddress)
	return []byte(key)
}

func calcBindMinerKey(minerAddress string, returnAddress string) []byte {
	key := fmt.Sprintf("ticket-miner:%s:%s", minerAddress, returnAddress)
	return []byte(key)
}

func calcBindMinerKeyPrefix(minerAddress string) []byte {
	key := fmt.Sprintf("ticket-miner:%s", minerAddress)
	return []byte(key)
}

func calcTicketPrefix(addr string, status int32) []byte {
	key := fmt.Sprintf("ticket-tl:%s:%d", addr, status)
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
