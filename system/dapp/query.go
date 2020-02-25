// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dapp

import (
	"reflect"

	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
)

// GetPrefixCount query the number keys of the specified prefix, for statistical
func (d *DriverBase) GetPrefixCount(key *types.ReqKey) (types.Message, error) {
	var counts types.Int64
	db := d.GetLocalDB()
	counts.Data = db.PrefixCount(key.Key)
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
