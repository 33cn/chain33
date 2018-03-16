package blacklist

import (
	"code.aliyun.com/chain33/chain33/executor/drivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
	//"fmt"
	"time"

	//"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/proto"
)

var clog = log.New("module", "execs.blacklist")
var (
	initialCredit int64 = 1e8
	layout              = "2006-01-02 15:04:05.000"
	date                = "20060102150405.000"
	loc           *time.Location
)

func init() {
	loc, _ = time.LoadLocation("Asia/Shanghai")
	black := newBlackList()
	drivers.Register(black.GetName(), black)
}

type BlackList struct {
	drivers.DriverBase
}

func newBlackList() *BlackList {
	n := &BlackList{}
	n.SetChild(n)
	n.SetIsFree(true)
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
	if action.FuncName == SubmitRecord && action.GetRc() != nil {
		return SubmitRecord
	} else if action.FuncName == QueryOrg && action.GetOr() != nil {
		return QueryOrg
	} else if action.FuncName == QueryRecord {
		return QueryRecord
	} else if action.FuncName == CreateOrg && action.GetOr() != nil {
		return CreateOrg
	}
	return "unknow"
}

func (b *BlackList) GetKVPair(tx *types.Transaction) *types.KeyValue {
	action, err := b.GetActionValue(tx)
	if err != nil {
		return nil
	}
	if action.FuncName == SubmitRecord && action.GetRc() != nil {
		return &types.KeyValue{[]byte(b.GetName() + action.GetRc().GetRecordId()), []byte(action.GetRc().String())}
	} else if action.FuncName == CreateOrg && action.GetOr() != nil {
		return &types.KeyValue{[]byte(b.GetName() + action.GetOr().GetOrgId()), []byte(action.GetOr().String())}
	}

	return nil
}
func (b *BlackList) GetKVPairs(tx *types.Transaction) []*types.KeyValue {
	var kvs []*types.KeyValue
	action, err := b.GetActionValue(tx)
	if err != nil {
		return nil
	}
	if action.FuncName == SubmitRecord && action.GetRc() != nil {
		//TODO:以不同的key进行多次存储,以求能够规避chain33不支持多键值查询的短板
		//key=user.blacklist+recordId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + action.GetRc().GetRecordId()), []byte(action.GetRc().String())})
		//key=user.blacklist+clientName+recordId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + action.GetRc().GetClientName() + action.GetRc().GetRecordId()), []byte(action.GetRc().String())})
		//key=user.blacklist+clientId+recordId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + action.GetRc().GetClientId() + action.GetRc().GetRecordId()), []byte(action.GetRc().String())})
		//key=user.blacklist+orgId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + action.GetRc().GetOrgId()), []byte(action.GetRc().String())})
		return kvs
	} else if action.FuncName == CreateOrg && action.GetOr() != nil {
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + action.GetOr().GetOrgId()), []byte(action.GetOr().String())})
		return kvs
	} else if action.FuncName == DeleteRecord {
		record := b.deleteRecord([]byte(b.GetName() + action.GetRc().GetClientName() + action.GetRc().GetRecordId()))
		//key=user.blacklist+clientName+recordId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + record.GetClientName() + record.GetRecordId()), []byte(record.String())})
		//key=user.blacklist+clientId+recordId
		kvs = append(kvs, &types.KeyValue{[]byte(b.GetName() + record.GetClientId() + record.GetRecordId()), []byte(record.String())})
	}
	return nil
}

func (b *BlackList) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	action, err := b.GetActionValue(tx)
	if err != nil {
		return nil, err
	}
	clog.Debug("exec blacklist tx=", "tx=", action)
	receipt := &types.Receipt{types.ExecOk, nil, nil}
	if b.GetKVPair(tx) != nil {
		receipt.KV = append(receipt.KV, b.GetKVPair(tx))
	}
	return receipt, nil
}

func (b *BlackList) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	if b.GetKVPair(tx) != nil {
		set.KV = append(set.KV, b.GetKVPair(tx))
	}

	return &set, nil
}

func (b *BlackList) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	pair := b.GetKVPair(tx)
	if pair != nil {
		set.KV = append(set.KV, &types.KeyValue{pair.Key, nil})
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

func (b *BlackList) Query(funcname string, params []byte) (types.Message, error) {
	if funcname == QueryRecord {
		value := b.queryRecord(params)
		if value == "" {
			return nil, types.ErrNotFound
		}
		return &types.ReplyString{value}, nil
	} else if funcname == QueryOrg {
		value := b.queryOrg(params)
		if value == "" {
			return nil, types.ErrNotFound
		}
		return &types.ReplyString{value}, nil
	} else if funcname == QueryRecordByName {
		return &types.ReplyStrings{b.queryRecordByName(params)}, nil
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
	recordbytes := b.GetQueryDB().Get([]byte(b.GetName() + string(recordId)))
	if recordbytes != nil {
		return string(recordbytes)
	}
	return ""
}

//需要传入name和recordId
func (b *BlackList) queryRecordByName(name []byte) []string {
	var recordList []string
	recordBytes := b.GetQueryDB().PrefixScan([]byte(b.GetName() + string(name)))
	for recordByte := range recordBytes {
		var record Record
		err := proto.UnmarshalText(string(recordByte), &record)
		if err != nil {
			panic(err)
		}
		if !record.GetSearchable() {
			continue
		}
		recordList = append(recordList, string(recordByte))
	}
	return recordList
}
func (b *BlackList) queryTransaction() {

}
func (b *BlackList) issueCredit() {

}
func (b *BlackList) issueCreditToOrg() {

}
func (b *BlackList) transfer() {

}
func (b *BlackList) queryOrg(orgId []byte) string {
	value := b.GetQueryDB().Get([]byte(b.GetName() + string(orgId)))
	if value != nil {
		return string(value)
	}
	return ""
}
func (b *BlackList) queryAgency() {

}
