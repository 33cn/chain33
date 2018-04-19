package blacklist

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/executor/drivers/blacklist/httplisten"
	. "gitlab.33.cn/chain33/chain33/executor/drivers/blacklist/types"
	"gitlab.33.cn/chain33/chain33/types"
	//"fmt"
	"strconv"
	"time"

	"fmt"

	"github.com/gogo/protobuf/proto"
)

var blog = log.New("module", "execs.blacklist")
var (
	initialCredit int64 = 1e8
	layout              = "2006-01-02 15:04:05.000"
	date                = "20060102150405.000"
	loc           *time.Location
)

func init() {
	loc, _ = time.LoadLocation("Asia/Shanghai")
	black := newBlackList()
	drivers.Register(black.GetName(), black, 0)
	//开启httpListen
	go httplisten.Init()
}

type BlackList struct {
	drivers.DriverBase
}

func newBlackList() *BlackList {
	n := &BlackList{}
	n.SetChild(n)
	return n
}
func (b *BlackList) GetName() string {
	return "user.blacklist"
}

func (b *BlackList) GetActionValue(tx *types.Transaction) (*BlackAction, error) {
	action := &BlackAction{}
	err := types.Decode(tx.Payload, action)
	if err != nil {
		return nil, err
	}

	return action, nil
}

func (b *BlackList) GetActionName(tx *types.Transaction) string {
	action, err := b.GetActionValue(tx)
	if err != nil {
		return "unknow"
	}
	if action.FuncName == FuncName_SubmitRecord && action.GetRc() != nil {
		return FuncName_SubmitRecord
	} else if action.FuncName == FuncName_QueryOrgById && action.GetOr() != nil {
		return FuncName_QueryOrgById
	} else if action.FuncName == FuncName_QueryRecordById {
		return FuncName_QueryRecordById
	} else if action.FuncName == FuncName_CreateOrg && action.GetOr() != nil {
		return FuncName_CreateOrg
	}
	return "unknow"
}

//func (b *BlackList) GetKVPair(tx *types.Transaction) *types.KeyValue {
//	action, err := b.GetActionValue(tx)
//	if err != nil {
//		return nil
//	}
//	if action.FuncName == SubmitRecord && action.GetRc() != nil {
//		return &types.KeyValue{[]byte(b.GetName() + action.GetRc().GetRecordId()), []byte(action.GetRc().String())}
//	} else if action.FuncName == CreateOrg && action.GetOr() != nil {
//		return &types.KeyValue{[]byte(b.GetName() + action.GetOr().GetOrgId()), []byte(action.GetOr().String())}
//	}
//
//	return nil
//}
func (b *BlackList) GetKVPairs(tx *types.Transaction) []*types.KeyValue {
	var kvs []*types.KeyValue
	action, err := b.GetActionValue(tx)
	if err != nil {
		return nil
	}
	if action.FuncName == FuncName_SubmitRecord && action.GetRc() != nil {
		//TODO:以不同的key进行多次存储,以求能够规避chain33不支持多键值查询的短板
		record := action.GetRc()
		record.CreateTime = time.Now().In(loc).Format(layout)
		record.UpdateTime = time.Now().In(loc).Format(layout)
		//key=user.blacklist+recordId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + record.GetRecordId()), []byte(record.String())})
		//key=user.blacklist+clientName+recordId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + record.GetClientName() + record.GetRecordId()), []byte(record.String())})
		//key=user.blacklist+clientId+recordId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + record.GetClientId() + record.GetRecordId()), []byte(record.String())})
		//key=user.blacklist+orgId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + record.GetOrgId()), []byte(record.String())})
		//TODO:这块逻辑需要重新实现，存入后返回recordId
		//org := b.queryOrgById([]byte(record.GetOrgId()))

		//transaction 用于增加积分
		ts := &Transaction{}
		ts.From = GenesisAddr
		//ts.To = org.OrgAddr
		ts.Credit = AddCredit
		ts.CreateTime = time.Now().In(loc).Format(layout)
		ts.UpdateTime = time.Now().In(loc).Format(layout)
		ts.DocType = SubmitRecordDoc
		//kvs_ts, err := b.transfer(ts)
		//if err != nil {
		//	blog.Error("exec transfer func have a err! err: ", err)
		//}
		//if len(kvs_ts) != 0 {
		//	kvs = append(kvs, kvs_ts...)
		//}
		return kvs
	} else if action.FuncName == FuncName_CreateOrg && action.GetOr() != nil {
		org := action.GetOr()
		//如果传入的地址空，则生成随机不重复addr
		if org.GetOrgAddr() == "" {
			org.OrgAddr = generateAddr()
		}
		//机构只要注册就会获得100积分，用于消费
		org.OrgCredit = 100
		org.CreateTime = time.Now().In(loc).Format(layout)
		org.UpdateTime = time.Now().In(loc).Format(layout)
		//key=user.blacklist+orgId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + org.GetOrgId()), []byte(org.String())})
		//key=user.blacklist+orgAddr
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + org.GetOrgAddr()), []byte(org.String())})
		return kvs
	} else if action.FuncName == FuncName_DeleteRecord {
		record := b.deleteRecord([]byte(action.GetRc().GetRecordId()))
		blog.Info("blacklist+++++++++++record+++++recordId:"+strconv.FormatBool(record.GetSearchable()), nil)
		if record.GetRecordId() == "" {
			return kvs
		}
		blog.Info("record Searchable:"+strconv.FormatBool(record.GetSearchable()), nil)
		record.UpdateTime = time.Now().In(loc).Format(layout)
		//key=user.blacklist+clientName+recordId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + record.GetClientName() + record.GetRecordId()), []byte(record.String())})
		//key=user.blacklist+clientId+recordId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + record.GetRecordId()), []byte(record.String())})
		return kvs
	} else if action.FuncName == FuncName_Transfer && action.GetTr() != nil {
		kvs_ts, err := b.transfer(action.GetTr())
		if err != nil {
			blog.Error("exec transfer func have a err! err: ", err)
		}
		if len(kvs_ts) != 0 {
			kvs = append(kvs, kvs_ts...)
		}
		return kvs
	} else if action.FuncName == FuncName_RegisterUser && action.GetUser() != nil {
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + action.GetUser().GetUserName()), []byte(action.GetUser().String())})
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + action.GetUser().GetUserId()), []byte(action.GetUser().String())})
		//kvs = append(kvs,&types.KeyValue{[]byte(b.GetName()+action.GetUser().GetOrgId()+action.GetUser().GetUserId()),[]byte(action.GetUser().String())})
		return kvs
	} else if action.FuncName == FuncName_ModifyUserPwd && action.GetUser() != nil {
		user := b.getUserByName(action.GetUser().GetUserName())
		user.PassWord = action.GetUser().GetPassWord()
		user.UpdateTime = time.Now().In(loc).Format(layout)
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + user.GetUserName()), []byte(user.String())})
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + user.GetUserId()), []byte(user.String())})
		//kvs = append(kvs,&types.KeyValue{[]byte(b.GetName()+action.GetUser().GetOrgId()+action.GetUser().GetUserId()),[]byte(action.GetUser().String())})
		return kvs
	}
	return nil
}

func (b *BlackList) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	action, err := b.GetActionValue(tx)
	if err != nil {
		return nil, err
	}
	blog.Debug("exec blacklist tx=", "tx=", action)
	receipt := &types.Receipt{types.ExecOk, nil, nil}
	if b.GetKVPairs(tx) != nil {
		receipt.KV = append(receipt.KV, b.GetKVPairs(tx)...)
	}
	return receipt, nil
}

func (b *BlackList) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	if b.GetKVPairs(tx) != nil {
		set.KV = append(set.KV, b.GetKVPairs(tx)...)
	}

	return &set, nil
}

func (b *BlackList) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	pairs := b.GetKVPairs(tx)
	if pairs != nil && len(pairs) != 0 {
		var kvs []*types.KeyValue
		for _, kv := range pairs {
			kv.Value = nil
			kvs = append(kvs, kv)
		}
		set.KV = append(set.KV, kvs...)
		//set.KV = append(set.KV, &types.KeyValue{pair.Key, nil})
	}
	//del tx
	//hash, _ := n.GetTx(tx, receipt, index)
	//set.KV = append(set.KV, &types.KeyValue{hash, nil})

	return &set, nil
}

/*
########################################################################################################################
#########以下是基本的操作方法接口
########################################################################################################################
*/
func (b *BlackList) getQuery(param []byte) (*Query, error) {
	query := &Query{}
	err := types.Decode(param, query)
	if err != nil {
		return nil, err
	}
	return query, nil
}

func (b *BlackList) Query(funcname string, params []byte) (types.Message, error) {
	query, err := b.getQuery(params)
	if err != nil {
		blog.Error("exec getQuery func have a err:", err)
		return nil, ErrQueryNotSupport
	}
	if funcname == FuncName_QueryRecordById && query.GetQueryRecord() != nil {
		value := b.queryRecord([]byte(query.GetQueryRecord().GetByRecordId()))
		if value == "" {
			return nil, types.ErrNotFound
		}
		return &types.ReplyString{value}, nil
	} else if funcname == FuncName_QueryOrgById && query.GetQueryOrg() != nil {
		value := b.queryOrg([]byte(query.GetQueryOrg().GetOrgId()))
		if value == "" {
			return nil, types.ErrNotFound
		}
		return &types.ReplyString{value}, nil
	} else if funcname == FuncName_QueryRecordByName && query.GetQueryRecord() != nil {
		return &types.ReplyStrings{b.queryRecordByName([]byte(query.GetQueryRecord().GetByClientName()))}, nil
	} else if funcname == FuncName_QueryTxByFromAddr && query.GetQueryTransaction() != nil {
		values := b.queryTransactionByAddr([]byte(query.GetQueryTransaction().GetByFromAddr()))
		return &types.ReplyStrings{values}, nil
	} else if funcname == FuncName_QueryTxByToAddr && query.GetQueryTransaction() != nil {
		values := b.queryTransactionByAddr([]byte(query.GetQueryTransaction().GetByToAddr()))
		return &types.ReplyStrings{values}, nil
	} else if funcname == FuncName_QueryTxById && query.GetQueryTransaction() != nil {
		value := b.queryTransactionById([]byte(query.GetQueryTransaction().GetByTxId()))
		return &types.ReplyString{value}, nil
	} else if funcname == FuncName_LoginCheck && query.GetLoginCheck() != nil {
		if b.loginCheck(query.GetLoginCheck()) {
			return &types.ReplyString{SUCESS}, nil
		} else {
			return &types.ReplyString{FAIL}, nil
		}
	} else if funcname == FuncName_ModifyUserPwd && query.GetLoginCheck() != nil {
		//ToDO:密码修改这块校验逻辑应该放在外面？

	} else if funcname == FuncName_ResetUserPwd && query.GetLoginCheck() != nil {
		//TODO:密码重置功能待实现
	} else if funcname == FuncName_CheckKeyIsExsit && query.GetQueryKey() != nil {
		if b.checkKeyIsExsit(query.GetQueryKey().GetKey()) {
			return &types.ReplyString{SUCESS}, nil
		} else {
			return &types.ReplyString{FAIL}, nil
		}
	}
	return nil, types.ErrActionNotSupport
}
func (b *BlackList) createOrg(action *BlackAction) *types.KeyValue {
	//TODO:这里就简单采用string与byte[]之间的转换，后续如果需要提高安全性可用proto.Marshal序列化
	//data,err := proto.Marshal(action.GetOr())
	//if err != nil {
	//	panic(err)
	//}
	return &types.KeyValue{[]byte(b.GetName() + action.GetOr().GetOrgId()), []byte(action.GetOr().String())}
}
func (b *BlackList) submitRecord(action *BlackAction) *types.KeyValue {
	//data,err := proto.Marshal(action.GetRc())
	//if err != nil {
	//panic(err)
	//}
	return &types.KeyValue{[]byte(b.GetName() + action.GetRc().GetClientName() + action.GetRc().GetRecordId()), []byte(action.GetRc().String())}
}
func (b *BlackList) deleteRecord(recordId []byte) Record {
	var record Record
	str := b.queryRecord(recordId)
	if str != "" {
		err := proto.UnmarshalText(str, &record)
		if err != nil {
			panic(err)
		}
		record.Searchable = false
	}
	return record
}

//TODO:由于levedb不支持多键值查询，因此在写接口函数的时候只能做些规避
func (b *BlackList) queryRecord(recordId []byte) string {
	blog.Debug("recordId:============"+string(recordId), nil)
	recordbytes, _ := b.GetStateDB().Get([]byte(b.GetName() + string(recordId)))
	if recordbytes == nil {
		return ""
	}
	var record Record
	err := proto.UnmarshalText(string(recordbytes), &record)
	if err != nil {
		panic(err)
	}
	if !record.GetSearchable() {
		return ""
	}
	return record.String()
}

//需要传入name和recordId
func (b *BlackList) queryRecordByName(name []byte) []string {
	var recordList []string

	//it := b.GetStateDB().Iterator([]byte(b.GetName()+string(name)), false)
	//for it.Next() {
	//	var record Record
	//	err := proto.UnmarshalText(string(it.Value()), &record)
	//	if err != nil {
	//		panic(err)
	//	}
	//	if !record.GetSearchable() {
	//		continue
	//	}
	//	recordList = append(recordList, string(it.Value()))
	//}
	return recordList
}
func (b *BlackList) queryTransactionById(txId []byte) string {
	value, _ := b.GetStateDB().Get([]byte(b.GetName() + string(txId)))
	if value != nil {
		return string(value)
	}
	return ""
}
func (b *BlackList) queryTransactionByAddr(addr []byte) []string {
	var txs []string
	//it := b.GetStateDB().Iterator([]byte(b.GetName()+string(addr)), false)
	//for it.Next() {
	//var tx Transaction
	//err := proto.UnmarshalText(string(recordByte), &tx)
	//if err != nil {
	//	panic(err)
	//}
	//	txs = append(txs, string(it.Value()))
	//}
	return txs
}
func (b *BlackList) issueCredit() {

}
func (b *BlackList) issueCreditToOrg() {

}

//积分转移
func (b *BlackList) transfer(ts *Transaction) ([]*types.KeyValue, error) {
	var kvs []*types.KeyValue
	if ts.From == ts.To {
		return kvs, nil
	}
	//创世地址可以无限发送积分，故不用做检查积分是否充足
	if ts.From != GenesisAddr {
		orgF := b.queryOrgByAddr([]byte(ts.From))
		if orgF.OrgCredit < ts.Credit {
			blog.Error("积分不足，请充值!")
			return kvs, CreditInsufficient
		}
		orgF.OrgCredit = orgF.OrgCredit - ts.Credit
		orgF.UpdateTime = time.Now().In(loc).Format(layout)
		//key=user.blacklist+orgId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + orgF.GetOrgId()), []byte(orgF.String())})
		//key=user.blacklist+orgAddr
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + orgF.GetOrgAddr()), []byte(orgF.String())})
	}
	orgT := b.queryOrgByAddr([]byte(ts.To))
	orgT.OrgCredit = orgT.OrgCredit + ts.Credit
	orgT.UpdateTime = time.Now().In(loc).Format(layout)
	//key=user.blacklist+orgId
	kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + orgT.GetOrgId()), []byte(orgT.String())})
	//key=user.blacklist+orgAddr
	kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + orgT.GetOrgAddr()), []byte(orgT.String())})
	//transfer  key=user.blacklist+txId
	kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + ts.GetTxId()), []byte(ts.String())})
	//transfer  key=user.blacklist+ToAddr
	kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + ts.GetTo()), []byte(ts.String())})
	//transfer  key=user.blacklist+FromAddr
	kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + ts.GetFrom()), []byte(ts.String())})
	return kvs, nil
}
func (b *BlackList) queryOrg(orgId []byte) string {
	value, _ := b.GetStateDB().Get([]byte(b.GetName() + string(orgId)))
	if value != nil {
		return string(value)
	}
	return ""
}
func (b *BlackList) queryOrgById(orgId []byte) Org {
	var org Org
	value, _ := b.GetStateDB().Get([]byte(b.GetName() + string(orgId)))
	blog.Debug("value===================:", string(value))
	fmt.Println(value)
	if value != nil {
		err := proto.UnmarshalText(string(value), &org)
		if err != nil {
			panic(err)
		}
	}
	return org
}
func (b *BlackList) queryOrgByAddr(addr []byte) Org {
	var org Org
	value, _ := b.GetStateDB().Get([]byte(b.GetName() + string(addr)))
	if value != nil {
		err := proto.UnmarshalText(string(value), &org)
		if err != nil {
			panic(err)
		}
	}
	blog.Info("orgAddr=================:", addr)
	return org
}
func (b *BlackList) loginCheck(user *User) bool {
	var u User
	value, _ := b.GetStateDB().Get([]byte(b.GetName() + string(user.GetUserName())))
	if value != nil {
		err := proto.UnmarshalText(string(value), &u)
		if err != nil {
			panic(err)
		}
		if u.GetPassWord() == user.GetPassWord() {
			return true
		}
	}
	return false
}
func (b *BlackList) getUserByName(name string) *User {
	var u User
	value, _ := b.GetStateDB().Get([]byte(b.GetName() + name))
	if value != nil {
		err := proto.UnmarshalText(string(value), &u)
		if err != nil {
			panic(err)
		}
	}
	return &u
}
func (b *BlackList) checkKeyIsExsit(key string) bool {
	value, _ := b.GetStateDB().Get([]byte(b.GetName() + key))
	if value != nil {
		return true
	}
	return false
}
