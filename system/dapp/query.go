// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dapp

import (
	"errors"
	"reflect"

	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
)

// GetTxsByAddr find all transactions in this address by the addr prefix
// query transaction are placed by default ：coins in the query
func (d *DriverBase) GetTxsByAddr(addr *types.ReqAddr) (types.Message, error) {
	db := d.GetLocalDB()
	var prefix []byte
	var key []byte
	var txinfos [][]byte
	var err error
	//取最新的交易hash列表
	if addr.Flag == 0 { //所有的交易hash列表
		prefix = types.CalcTxAddrHashKey(addr.GetAddr(), "")
	} else if addr.Flag > 0 { //from的交易hash列表
		prefix = types.CalcTxAddrDirHashKey(addr.GetAddr(), addr.Flag, "")
	} else {
		return nil, errors.New("flag unknown")
	}
	if addr.GetHeight() == -1 {
		txinfos, err = db.List(prefix, nil, addr.Count, addr.GetDirection())
		if err != nil {
			return nil, err
		}
		if len(txinfos) == 0 {
			return nil, errors.New("tx does not exist")
		}
	} else { //翻页查找指定的txhash列表
		heightstr := HeightIndexStr(addr.GetHeight(), addr.GetIndex())
		if addr.Flag == 0 {
			key = types.CalcTxAddrHashKey(addr.GetAddr(), heightstr)
		} else if addr.Flag > 0 { //from的交易hash列表
			key = types.CalcTxAddrDirHashKey(addr.GetAddr(), addr.Flag, heightstr)
		} else {
			return nil, errors.New("flag unknown")
		}
		txinfos, err = db.List(prefix, key, addr.Count, addr.Direction)
		if err != nil {
			return nil, err
		}
		if len(txinfos) == 0 {
			return nil, errors.New("tx does not exist")
		}
	}
	var replyTxInfos types.ReplyTxInfos
	replyTxInfos.TxInfos = make([]*types.ReplyTxInfo, len(txinfos))
	for index, txinfobyte := range txinfos {
		var replyTxInfo types.ReplyTxInfo
		err := types.Decode(txinfobyte, &replyTxInfo)
		if err != nil {
			return nil, err
		}
		replyTxInfos.TxInfos[index] = &replyTxInfo
	}
	return &replyTxInfos, nil
}

// GetPrefixCount query the number keys of the specified prefix, for statistical
func (d *DriverBase) GetPrefixCount(key *types.ReqKey) (types.Message, error) {
	var counts types.Int64
	db := d.GetLocalDB()
	counts.Data = db.PrefixCount(key.Key)
	return &counts, nil
}

// GetAddrTxsCount query the transaction count for the specified address ，for statistical
func (d *DriverBase) GetAddrTxsCount(reqkey *types.ReqKey) (types.Message, error) {
	var counts types.Int64
	db := d.GetLocalDB()
	TxsCount, err := db.Get(reqkey.Key)
	if err != nil && err != types.ErrNotFound {
		counts.Data = 0
		return &counts, nil
	}
	if len(TxsCount) == 0 {
		counts.Data = 0
		return &counts, nil
	}
	err = types.Decode(TxsCount, &counts)
	if err != nil {
		counts.Data = 0
		return &counts, nil
	}
	return &counts, nil
}

// Query defines query function
func (d *DriverBase) Query(funcname string, params []byte) (msg types.Message, err error) {
	funcmap := d.child.GetFuncMap()
	funcname = "Query_" + funcname
	if _, ok := funcmap[funcname]; !ok {
		blog.Error(funcname+" funcname not find", "func", funcname)
		return nil, types.ErrActionNotSupport
	}
	ty := funcmap[funcname].Type
	if ty.NumIn() != 2 {
		blog.Error(funcname+" err num in param", "num", ty.NumIn())
		return nil, types.ErrActionNotSupport
	}
	paramin := ty.In(1)
	if paramin.Kind() != reflect.Ptr {
		blog.Error(funcname + "  param is not pointer")
		return nil, types.ErrActionNotSupport
	}
	p := reflect.New(ty.In(1).Elem())
	queryin := p.Interface()
	if in, ok := queryin.(proto.Message); ok {
		err := types.Decode(params, in)
		if err != nil {
			return nil, err
		}
		return types.CallQueryFunc(d.childValue, funcmap[funcname], in)
	}
	blog.Error(funcname + " in param is not proto.Message")
	return nil, types.ErrActionNotSupport
}
