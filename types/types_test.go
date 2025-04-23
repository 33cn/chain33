// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types/jsonpb"
	proto "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestAllowExecName(t *testing.T) {
	//allow exec list
	old := AllowUserExec
	defer func() {
		AllowUserExec = old
	}()
	AllowUserExec = nil
	AllowUserExec = append(AllowUserExec, []byte("coins"))
	isok := IsAllowExecName([]byte("a"), []byte("a"))
	assert.Equal(t, isok, false)

	isok = IsAllowExecName([]byte("coins"), []byte("coins"))
	assert.Equal(t, isok, true)

	isok = IsAllowExecName([]byte("coins"), []byte("user.coins"))
	assert.Equal(t, isok, true)

	isok = IsAllowExecName([]byte("coins"), []byte("user.coinsx"))
	assert.Equal(t, isok, false)

	isok = IsAllowExecName([]byte("coins"), []byte("user.coins.evm2"))
	assert.Equal(t, isok, true)

	isok = IsAllowExecName([]byte("coins"), []byte("user.p.guodun.coins.evm2"))
	assert.Equal(t, isok, false)

	isok = IsAllowExecName([]byte("coins"), []byte("user.p.guodun.coins"))
	assert.Equal(t, isok, true)

	isok = IsAllowExecName([]byte("coins"), []byte("user.p.guodun.user.coins"))
	assert.Equal(t, isok, true)

	isok = IsAllowExecName([]byte("#coins"), []byte("user.p.guodun.user.coins"))
	assert.Equal(t, isok, false)

	isok = IsAllowExecName([]byte("coins-"), []byte("user.p.guodun.user.coins"))
	assert.Equal(t, isok, false)
}

func BenchmarkExecName(b *testing.B) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	for i := 0; i < b.N; i++ {
		cfg.ExecName("hello")
	}
}

func BenchmarkG(b *testing.B) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	for i := 0; i < b.N; i++ {
		cfg.G("TestNet")
	}
}

func BenchmarkS(b *testing.B) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	for i := 0; i < b.N; i++ {
		cfg.S("helloword", true)
	}
}
func TestJsonNoName(t *testing.T) {
	flag := int32(1)
	params := struct {
		Flag int32
	}{
		Flag: flag,
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(data), "{\"Flag\":1}")
}

func TestNil(t *testing.T) {
	v := reflect.ValueOf(nil)
	assert.Equal(t, v.IsValid(), false)
}

func TestProtoToJson(t *testing.T) {
	r := &Reply{}
	b, err := json.Marshal(r)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte(`{}`))

	encode := &jsonpb.Marshaler{EmitDefaults: true}
	s, err := encode.MarshalToString(r)
	assert.Nil(t, err)
	assert.Equal(t, s, `{"isOk":false,"msg":null}`)
	var dr Reply
	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":null}`, &dr)
	assert.Nil(t, err)
	assert.Nil(t, dr.Msg)
	encode2 := &jsonpb.Marshaler{EmitDefaults: false}
	s, err = encode2.MarshalToString(r)
	assert.Nil(t, err)
	assert.Equal(t, s, `{}`)

	r = &Reply{Msg: []byte("OK")}
	b, err = json.Marshal(r)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte(`{"msg":"T0s="}`))

	encode = &jsonpb.Marshaler{EmitDefaults: true}
	s, err = encode.MarshalToString(r)
	assert.Nil(t, err)
	assert.Equal(t, s, `{"isOk":false,"msg":"0x4f4b"}`)

	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":"0x4f4b"}`, &dr)
	assert.Nil(t, err)
	assert.Equal(t, dr.Msg, []byte("OK"))

	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":"4f4b"}`, &dr)
	assert.Equal(t, err, jsonpb.ErrBytesFormat)

	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":"0x"}`, &dr)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(dr.Msg))

	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":"str://OK"}`, &dr)
	assert.Nil(t, err)
	assert.Equal(t, dr.Msg, []byte("OK"))

	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":"str://0"}`, &dr)
	assert.Nil(t, err)
	assert.Equal(t, dr.Msg, []byte("0"))

	r = &Reply{Msg: []byte{}}
	b, err = json.Marshal(r)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte(`{}`))

	encode = &jsonpb.Marshaler{EmitDefaults: true}
	s, err = encode.MarshalToString(r)
	assert.Nil(t, err)
	assert.Equal(t, s, `{"isOk":false,"msg":""}`)

	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":""}`, &dr)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(dr.Msg))
}

func TestJsonpbUTF8(t *testing.T) {
	r := &Reply{Msg: []byte("OK")}
	b, err := PBToJSONUTF8(r)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte(`{"isOk":false,"msg":"OK"}`))

	var newreply Reply
	err = JSONToPBUTF8(b, &newreply)
	assert.Nil(t, err)
	assert.Equal(t, r, &newreply)
}

func TestJsonpb(t *testing.T) {
	r := &Reply{Msg: []byte("OK")}
	b, err := PBToJSON(r)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte(`{"isOk":false,"msg":"0x4f4b"}`))

	var newreply Reply
	err = JSONToPB(b, &newreply)
	assert.Nil(t, err)
	assert.Equal(t, r, &newreply)
}

func TestHex(t *testing.T) {
	s := "0x4f4b"
	b, err := common.FromHex(s)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte("OK"))
}

func TestGetLogName(t *testing.T) {
	name := GetLogName([]byte("xxx"), 0)
	assert.Equal(t, "LogReserved", name)
	assert.Equal(t, "LogErr", GetLogName([]byte("coins"), 1))
	assert.Equal(t, "LogFee", GetLogName([]byte("token"), 2))
	assert.Equal(t, "LogReserved", GetLogName([]byte("xxxx"), 100))
}

func TestDecodeLog(t *testing.T) {
	data, _ := common.FromHex("0x0a2b10c0c599b78c1d2222314c6d7952616a4e44686f735042746259586d694c466b5174623833673948795565122b1080ab8db78c1d2222314c6d7952616a4e44686f735042746259586d694c466b5174623833673948795565")
	l, err := DecodeLog([]byte("xxx"), 2, data)
	assert.Nil(t, err)
	j, err := json.Marshal(l)
	assert.Nil(t, err)
	assert.Equal(t, "{\"prev\":{\"balance\":999769400000,\"addr\":\"1LmyRajNDhosPBtbYXmiLFkQtb83g9HyUe\"},\"current\":{\"balance\":999769200000,\"addr\":\"1LmyRajNDhosPBtbYXmiLFkQtb83g9HyUe\"}}", string(j))
}

func TestGetRealExecName(t *testing.T) {
	a := []struct {
		key     string
		realkey string
	}{
		{"coins", "coins"},
		{"user.p.coins", "user.p.coins"},
		{"user.p.guodun.coins", "coins"},
		{"user.evm.hash", "evm"},
		{"user.p.para.evm.hash", "evm.hash"},
		{"user.p.para.user.evm.hash", "evm"},
		{"user.p.para.", "user.p.para."},
	}
	for _, v := range a {
		assert.Equal(t, string(GetRealExecName([]byte(v.key))), v.realkey)
	}
}

func genPrefixEdge(prefix []byte) (r []byte) {
	for j := 0; j < len(prefix); j++ {
		r = append(r, prefix[j])
	}

	i := len(prefix) - 1
	for i >= 0 {
		if r[i] < 0xff {
			r[i]++
			break
		} else {
			i--
		}
	}

	return r
}

func (t *StoreListReply) IterateCallBack(key, value []byte) bool {
	if t.Mode == 1 { //[start, end)
		if t.Num >= t.Count {
			t.NextKey = key
			return true
		}
		t.Num++
		t.Keys = append(t.Keys, cloneByte(key))
		t.Values = append(t.Values, cloneByte(value))
		return false
	} else if t.Mode == 2 { //prefix + suffix
		if len(key) > len(t.Suffix) {
			if string(key[len(key)-len(t.Suffix):]) == string(t.Suffix) {
				t.Num++
				t.Keys = append(t.Keys, cloneByte(key))
				t.Values = append(t.Values, cloneByte(value))
				if t.Num >= t.Count {
					t.NextKey = key
					return true
				}
			}
			return false
		}
		return false
	} else {
		fmt.Println("StoreListReply.IterateCallBack unsupported mode", "mode", t.Mode)
		return true
	}
}

func cloneByte(v []byte) []byte {
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func TestIterateCallBack_PrefixWithoutExecAddr(t *testing.T) {
	key := "mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	//prefix1 := "mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:"
	prefix2 := "mavl-coins-bty-exec-"
	//execAddr := "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
	addr := "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"

	var reply = &StoreListReply{
		Start:  []byte(prefix2),
		End:    genPrefixEdge([]byte(prefix2)),
		Suffix: []byte(addr),
		Mode:   int64(2),
		Count:  int64(100),
	}

	var acc = &Account{
		Currency: 0,
		Balance:  1,
		Frozen:   1,
		Addr:     addr,
	}

	value := Encode(acc)

	bRet := reply.IterateCallBack([]byte(key), value)
	assert.Equal(t, false, bRet)
	assert.Equal(t, 1, len(reply.Keys))
	assert.Equal(t, 1, len(reply.Values))
	assert.Equal(t, int64(1), reply.Num)
	assert.Equal(t, 0, len(reply.NextKey))

	bRet = reply.IterateCallBack([]byte(key), value)
	assert.Equal(t, false, bRet)
	assert.Equal(t, 2, len(reply.Keys))
	assert.Equal(t, 2, len(reply.Values))
	assert.Equal(t, int64(2), reply.Num)
	assert.Equal(t, 0, len(reply.NextKey))

	key2 := "mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:2JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	bRet = reply.IterateCallBack([]byte(key2), value)
	assert.Equal(t, false, bRet)
	assert.Equal(t, 2, len(reply.Keys))
	assert.Equal(t, 2, len(reply.Values))
	assert.Equal(t, int64(2), reply.Num)
	assert.Equal(t, 0, len(reply.NextKey))

	key3 := "mavl-coins-bty-exec-26htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	bRet = reply.IterateCallBack([]byte(key3), value)
	assert.Equal(t, false, bRet)
	assert.Equal(t, 3, len(reply.Keys))
	assert.Equal(t, 3, len(reply.Values))
	assert.Equal(t, int64(3), reply.Num)
	assert.Equal(t, 0, len(reply.NextKey))

	reply.Count = int64(4)

	bRet = reply.IterateCallBack([]byte(key3), value)
	assert.Equal(t, true, bRet)
	assert.Equal(t, 4, len(reply.Keys))
	assert.Equal(t, 4, len(reply.Values))
	assert.Equal(t, int64(4), reply.Num)
	assert.Equal(t, key3, string(reply.NextKey))
	fmt.Println(string(reply.NextKey))
}

func TestIterateCallBack_PrefixWithExecAddr(t *testing.T) {
	key := "mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	prefix1 := "mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:"
	//execAddr := "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
	addr := "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"

	var reply = &StoreListReply{
		Start:  []byte(prefix1),
		End:    genPrefixEdge([]byte(prefix1)),
		Suffix: []byte(addr),
		Mode:   int64(2),
		Count:  int64(1),
	}

	var acc = &Account{
		Currency: 0,
		Balance:  1,
		Frozen:   1,
		Addr:     addr,
	}

	value := Encode(acc)

	key2 := "mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:2JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	bRet := reply.IterateCallBack([]byte(key2), value)
	assert.Equal(t, false, bRet)
	assert.Equal(t, 0, len(reply.Keys))
	assert.Equal(t, 0, len(reply.Values))
	assert.Equal(t, int64(0), reply.Num)
	assert.Equal(t, 0, len(reply.NextKey))

	bRet = reply.IterateCallBack([]byte(key), value)
	assert.Equal(t, true, bRet)
	assert.Equal(t, 1, len(reply.Keys))
	assert.Equal(t, 1, len(reply.Values))
	assert.Equal(t, int64(1), reply.Num)
	assert.Equal(t, len(key), len(reply.NextKey))

	//key2 := "mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:2JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	reply.NextKey = nil
	reply.Count = int64(2)
	bRet = reply.IterateCallBack([]byte(key2), value)
	assert.Equal(t, false, bRet)
	assert.Equal(t, 1, len(reply.Keys))
	assert.Equal(t, 1, len(reply.Values))
	assert.Equal(t, int64(1), reply.Num)
	assert.Equal(t, 0, len(reply.NextKey))

	reply.NextKey = nil
	key3 := "mavl-coins-bty-exec-26htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	bRet = reply.IterateCallBack([]byte(key3), value)
	assert.Equal(t, true, bRet)
	assert.Equal(t, 2, len(reply.Keys))
	assert.Equal(t, 2, len(reply.Values))
	assert.Equal(t, int64(2), reply.Num)
	assert.Equal(t, len(key3), len(reply.NextKey))

	bRet = reply.IterateCallBack([]byte(key), value)
	assert.Equal(t, true, bRet)
	assert.Equal(t, 3, len(reply.Keys))
	assert.Equal(t, 3, len(reply.Values))
	assert.Equal(t, int64(3), reply.Num)
	assert.Equal(t, len(key), len(reply.NextKey))
}

func TestJsonpbUTF8Tx(t *testing.T) {
	NewChain33Config(GetDefaultCfgstring())
	bdata, err := common.FromHex("0a05636f696e73121018010a0c108084af5f1a05310a320a3320e8b31b30b9b69483d7f9d3f04c3a22314b67453376617969715a4b6866684d66744e3776743267447639486f4d6b393431")
	assert.Nil(t, err)
	var r Transaction
	err = Decode(bdata, &r)
	assert.Nil(t, err)
	plType := LoadExecutorType("coins")
	var pl Message
	if plType != nil {
		pl, err = plType.DecodePayload(&r)
		if err != nil {
			pl = nil
		}
	}
	var pljson json.RawMessage
	assert.NotNil(t, pl)
	pljson, err = PBToJSONUTF8(pl)
	assert.Nil(t, err)
	assert.Equal(t, string(pljson), `{"transfer":{"cointoken":"","amount":"200000000","note":"1\n2\n3","to":""},"ty":1}`)
}

func TestSignatureClone(t *testing.T) {
	s1 := &Signature{Ty: 1, Pubkey: []byte("Pubkey1"), Signature: []byte("Signature1")}
	s2 := s1.Clone()
	s2.Pubkey = []byte("Pubkey2")
	assert.Equal(t, s1.Ty, s2.Ty)
	assert.Equal(t, s1.Signature, s2.Signature)
	assert.Equal(t, []byte("Pubkey1"), s1.Pubkey)
	assert.Equal(t, []byte("Pubkey2"), s2.Pubkey)
}

func TestTxClone(t *testing.T) {
	s1 := &Signature{Ty: 1, Pubkey: []byte("Pubkey1"), Signature: []byte("Signature1")}
	tx1 := &Transaction{Execer: []byte("Execer1"), Fee: 1, Signature: s1}
	tx2 := tx1.Clone()
	tx2.Signature.Pubkey = []byte("Pubkey2")
	tx2.Fee = 2
	assert.Equal(t, tx1.Execer, tx2.Execer)
	assert.Equal(t, int64(1), tx1.Fee)
	assert.Equal(t, tx1.Signature.Ty, tx2.Signature.Ty)
	assert.Equal(t, []byte("Pubkey1"), tx1.Signature.Pubkey)
	assert.Equal(t, []byte("Pubkey2"), tx2.Signature.Pubkey)

	tx2.Signature = nil
	assert.NotNil(t, tx1.Signature)
	assert.Nil(t, tx2.Signature)
}

func TestBlockClone(t *testing.T) {
	b1 := getTestBlockDetail()
	b2 := b1.Clone()

	b2.Block.Signature.Ty = 22
	assert.NotEqual(t, b1.Block.Signature.Ty, b2.Block.Signature.Ty)
	assert.Equal(t, b1.Block.Signature.Signature, b2.Block.Signature.Signature)

	b2.Block.Txs[1].Execer = []byte("E22")
	assert.NotEqual(t, b1.Block.Txs[1].Execer, b2.Block.Txs[1].Execer)
	assert.Equal(t, b1.Block.Txs[1].Fee, b2.Block.Txs[1].Fee)

	b2.KV[1].Key = []byte("key22")
	assert.NotEqual(t, b1.KV[1].Key, b2.KV[1].Key)
	assert.Equal(t, b1.KV[1].Value, b2.KV[1].Value)

	b2.Receipts[1].Ty = 22
	assert.NotEqual(t, b1.Receipts[1].Ty, b2.Receipts[1].Ty)
	assert.Equal(t, b1.Receipts[1].Logs, b2.Receipts[1].Logs)

	b2.Block.Txs[0] = nil
	assert.NotNil(t, b1.Block.Txs[0])
}

func TestBlockBody(t *testing.T) {
	detail := getTestBlockDetail()
	b1 := BlockBody{
		Txs:        detail.Block.Txs,
		Receipts:   detail.Receipts,
		MainHash:   []byte("MainHash1"),
		MainHeight: 1,
		Hash:       []byte("Hash"),
		Height:     1,
	}
	b2 := b1.Clone()

	b2.Txs[1].Execer = []byte("E22")
	assert.NotEqual(t, b1.Txs[1].Execer, b2.Txs[1].Execer)
	assert.Equal(t, b1.Txs[1].Fee, b2.Txs[1].Fee)

	b2.Receipts[1].Ty = 22
	assert.NotEqual(t, b1.Receipts[1].Ty, b2.Receipts[1].Ty)
	assert.Equal(t, b1.Receipts[1].Logs, b2.Receipts[1].Logs)

	b2.Txs[0] = nil
	assert.NotNil(t, b1.Txs[0])

	b2.MainHash = []byte("MainHash2")
	assert.NotEqual(t, b1.MainHash, b2.MainHash)
	assert.Equal(t, b1.Height, b2.Height)
}

func getTestBlockDetail() *BlockDetail {
	s1 := &Signature{Ty: 1, Pubkey: []byte("Pubkey1"), Signature: []byte("Signature1")}
	s2 := &Signature{Ty: 2, Pubkey: []byte("Pubkey2"), Signature: []byte("Signature2")}
	tx1 := &Transaction{Execer: []byte("Execer1"), Fee: 1, Signature: s1}
	tx2 := &Transaction{Execer: []byte("Execer2"), Fee: 2, Signature: s2}

	sigBlock := &Signature{Ty: 1, Pubkey: []byte("BlockPubkey1"), Signature: []byte("BlockSignature1")}
	block := &Block{
		Version:    1,
		ParentHash: []byte("ParentHash"),
		TxHash:     []byte("TxHash"),
		StateHash:  []byte("TxHash"),
		Height:     1,
		BlockTime:  1,
		Difficulty: 1,
		MainHash:   []byte("MainHash"),
		MainHeight: 1,
		Signature:  sigBlock,
		Txs:        []*Transaction{tx1, tx2},
	}
	kv1 := &KeyValue{Key: []byte("key1"), Value: []byte("value1")}
	kv2 := &KeyValue{Key: []byte("key1"), Value: []byte("value1")}

	log1 := &ReceiptLog{Ty: 1, Log: []byte("log1")}
	log2 := &ReceiptLog{Ty: 2, Log: []byte("log2")}
	receipts := []*ReceiptData{
		{Ty: 11, Logs: []*ReceiptLog{log1}},
		{Ty: 12, Logs: []*ReceiptLog{log2}},
	}

	return &BlockDetail{
		Block:          block,
		Receipts:       receipts,
		KV:             []*KeyValue{kv1, kv2},
		PrevStatusHash: []byte("PrevStatusHash"),
	}
}

// 测试数据对象复制效率
func deepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func BenchmarkDeepCody(b *testing.B) {
	block := getRealBlockDetail()
	if block == nil {
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var b1 *BlockDetail
		_ = deepCopy(b1, block)
	}

}

func BenchmarkProtoClone(b *testing.B) {
	block := getRealBlockDetail()
	if block == nil {
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = proto.Clone(block).(*BlockDetail)
	}
}

func BenchmarkProtoMarshal(b *testing.B) {
	block := getRealBlockDetail()
	if block == nil {
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x, _ := proto.Marshal(block)
		var b BlockDetail
		proto.Unmarshal(x, &b)
	}
}

func BenchmarkCopyBlockDetail(b *testing.B) {
	block := getRealBlockDetail()
	if block == nil {
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = block.Clone()
	}
}

func getRealBlockDetail() *BlockDetail {
	hexBlock := "0aad071220d29ccba4a90178614c8036f962201fdf56e77b419ca570371778e129a0e0e2841a20dd690056e7719d5b5fd1ad3e76f3b4366db9f3278ca29fbe4fc523bbf756fa8a222064cdbc89fe14ae7b851a71e41bd36a82db8cd3b69f4434f6420b279fea4fd25028e90330d386c4ea053a99040a067469636b657412ed02501022e80208ffff8bf8011080cab5ee011a70313271796f6361794e46374c7636433971573461767873324537553431664b5366763a3078386434663130653666313762616533636538663764303239616263623461393839616563333333386238386662333537656165663035613265326465373930343a30303030303035363533224230783537656632356366363036613734393462626164326432666233373734316137636332346663663066653064303637363638636564306235653961363735336632206b9836b2d295ca16634ea83359342db3ab1ab5f15200993bf6a09024089b7b693a810136f5a704a427a0562653345659193a0e292f16200ce67d6a8f8af631149a904fb80f0c12f525f7ce208fbf2183f2052af6252a108bb2614db0ccf8d91d4e910a04c472f5113275fe937f68ed6b2b0b522cc5fc2594b9fec60c0b22524b828333aaa982be125ec0f69645c86de3d331b26aa84c29a06824e977ce51f76f34f0629c1a6d080112210320bbac09528e19c55b0f89cb37ab265e7e856b1a8c388780322dbbfd194b52ba1a46304402200971163de32cb6a17e925eb3fcec8a0ccc193493635ecbf52357f5365edc2c82022039d84aa7078bc51ef83536038b7fd83087bf4deb965370f211a5589d4add551720a08d063097c5abcc9db1e4a92b3a22313668747663424e53454137665a6841644c4a706844775152514a614870794854703af4010a05746f6b656e124438070a400a0879696e6865626962120541424344452080d0dbc3f4022864322231513868474c666f47653633656665576138664a34506e756b686b6e677436706f4b38011a6d08011221021afd97c3d9a47f7ead3ca34fe5a73789714318df2c608cf2c7962378dc858ecb1a4630440220316a241f19b392e685ef940dee48dc53f90fc5c8be108ceeef1d3c65f530ee5f02204c89708cc7dac408a88e9fa6ad8e723d93fc18fe114d4a416e280f5a15656b0920a08d0628ca87c4ea0530dd91be81ae9ed1882d3a22313268704a4248796268316d537943696a51324d514a506b377a376b5a376a6e516158ffff8bf80162200b97166ad507aea57a4bb6e6b9295ec082cdc670b8468a83b559dbd900ffb83068e90312e80508021a5e0802125a0a2b1080978dcaef192222313271796f6361794e46374c7636433971573461767873324537553431664b536676122b10e08987caef192222313271796f6361794e46374c7636433971573461767873324537553431664b5366761a9f010871129a010a70313271796f6361794e46374c7636433971573461767873324537553431664b5366763a3078386434663130653666313762616533636538663764303239616263623461393839616563333333386238386662333537656165663035613265326465373930343a30303030303035363533100218012222313271796f6361794e46374c7636433971573461767873324537553431664b5366761a620805125e0a2d1080e0cd90f6a6ca342222313668747663424e53454137665a6841644c4a706844775152514a61487079485470122d1080aa83fff7a6ca342222313668747663424e53454137665a6841644c4a706844775152514a614870794854701a870108081282010a22313668747663424e53454137665a6841644c4a706844775152514a61487079485470122d1880f0cae386e68611222231344b454b6259744b4b516d34774d7468534b394a344c61346e41696964476f7a741a2d1880ba80d288e68611222231344b454b6259744b4b516d34774d7468534b394a344c61346e41696964476f7a741a620805125e0a2d1080aa83fff7a6ca342222313668747663424e53454137665a6841644c4a706844775152514a61487079485470122d1080f0898ef9a6ca342222313668747663424e53454137665a6841644c4a706844775152514a614870794854701a8f010808128a010a22313668747663424e53454137665a6841644c4a706844775152514a6148707948547012311080e0ba84bf03188090c0ac622222314251585336547861595947356d41446157696a344178685a5a55547077393561351a311080e0ba84bf031880d6c6bb632222314251585336547861595947356d41446157696a344178685a5a555470773935613512970208021a5c080212580a2a10c099b8c321222231513868474c666f47653633656665576138664a34506e756b686b6e677436706f4b122a10a08cb2c321222231513868474c666f47653633656665576138664a34506e756b686b6e677436706f4b1a82010809127e0a22313268704a4248796268316d537943696a51324d514a506b377a376b5a376a6e5161122a108094ebdc03222231513868474c666f47653633656665576138664a34506e756b686b6e677436706f4b1a2c109c93ebdc031864222231513868474c666f47653633656665576138664a34506e756b686b6e677436706f4b1a3008d301122b0a054142434445122231513868474c666f47653633656665576138664a34506e756b686b6e677436706f4b"
	bs, err := hex.DecodeString(hexBlock)
	if err != nil {
		return nil
	}
	var block BlockDetail
	err = Decode(bs, &block)
	if err != nil {
		return nil
	}
	return &block
}

// go test -run=none -bench=2Str -benchmem
func BenchmarkBytes2Str(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	buf := []byte{1, 2, 4, 8, 16, 32, 64, 128}
	s := ""
	b.ResetTimer()

	b.Run("DirectBytes2Str", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s = string(buf)
		}
	})

	b.Run("UnsafeBytes2Str", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s = Bytes2Str(buf)
		}
	})

	assert.Equal(b, string(buf), s)
}

func BenchmarkStr2Bytes(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	s := "BenchmarkStr2Bytes"
	var buf []byte

	b.ResetTimer()

	b.Run("DirectStr2Byte", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf = []byte(s)
		}
	})

	b.Run("UnsafeStr2Byte", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf = Str2Bytes(s)
		}
	})
	assert.Equal(b, []byte(s), buf)
}

func TestFormatAmount2FloatDisplay(t *testing.T) {
	//round example
	//val := 90012345678
	//roundnot :=strconv.FormatFloat(float64(val/1e4)/float64(1e4), 'f', 4, 64)
	//round :=strconv.FormatFloat(float64(val)/float64(1e8), 'f', 4, 64)
	//fmt.Println("notround=",roundnot,"round=",round)

	r := FormatAmount2FloatDisplay(0, DefaultCoinPrecision, false)
	assert.Equal(t, r, "0.0000")

	r = FormatAmount2FloatDisplay(1, DefaultCoinPrecision, false)
	assert.Equal(t, r, "0.0000")

	r = FormatAmount2FloatDisplay(12345678, DefaultCoinPrecision, false)
	assert.Equal(t, r, "0.1234")

	r = FormatAmount2FloatDisplay(12345678, DefaultCoinPrecision, true)
	assert.Equal(t, r, "0.1235")

	r = FormatAmount2FloatDisplay(1234567812345678, DefaultCoinPrecision, false)
	assert.Equal(t, r, "12345678.1234")

	r = FormatAmount2FloatDisplay(9001234567812345678, DefaultCoinPrecision, false)
	assert.Equal(t, r, "90012345678.1234")

	r = FormatAmount2FloatDisplay(12345678, 1, false)
	assert.Equal(t, r, "12345678")

	r = FormatAmount2FloatDisplay(0, 1, false)
	assert.Equal(t, r, "0")

	r = FormatAmount2FloatDisplay(9001234567812345678, 1, false)
	assert.Equal(t, r, "9001234567812345678")

	r = FormatAmount2FloatDisplay(1234567812345678, 10, false)
	assert.Equal(t, r, "123456781234567.8")

	r = FormatAmount2FloatDisplay(9001234567812345678, 10, false)
	assert.Equal(t, r, "900123456781234567.8")

	r = FormatAmount2FloatDisplay(0, 10, false)
	assert.Equal(t, r, "0.0")

	r = FormatAmount2FloatDisplay(100000000, DefaultCoinPrecision, false)
	assert.Equal(t, r, "1.0000")

}

func TestFormatAmount2FixPrecisionDisplay(t *testing.T) {
	r := FormatAmount2FixPrecisionDisplay(0, DefaultCoinPrecision)
	assert.Equal(t, r, "0.00000000")

	r = FormatAmount2FixPrecisionDisplay(1, DefaultCoinPrecision)
	assert.Equal(t, r, "0.00000001")

	r = FormatAmount2FixPrecisionDisplay(12345678, DefaultCoinPrecision)
	assert.Equal(t, r, "0.12345678")

	r = FormatAmount2FixPrecisionDisplay(1234567812345678, DefaultCoinPrecision)
	assert.Equal(t, r, "12345678.12345678")

	r = FormatAmount2FixPrecisionDisplay(9001234567812345678, DefaultCoinPrecision)
	assert.Equal(t, r, "90012345678.12345678")

	r = FormatAmount2FixPrecisionDisplay(12345678, 1)
	assert.Equal(t, r, "12345678")

	r = FormatAmount2FixPrecisionDisplay(0, 1)
	assert.Equal(t, r, "0")

	r = FormatAmount2FixPrecisionDisplay(9001234567812345678, 1)
	assert.Equal(t, r, "9001234567812345678")

	r = FormatAmount2FixPrecisionDisplay(1234567812345678, 10)
	assert.Equal(t, r, "123456781234567.8")

	r = FormatAmount2FixPrecisionDisplay(9001234567812345678, 10)
	assert.Equal(t, r, "900123456781234567.8")

	r = FormatAmount2FixPrecisionDisplay(0, 10)
	assert.Equal(t, r, "0.0")

	r = FormatAmount2FixPrecisionDisplay(100000000, DefaultCoinPrecision)
	assert.Equal(t, r, "1.00000000")

}

func TestFormatFloatDisplay2Value(t *testing.T) {
	//超过最大字符数
	in := float64(9512345678.12347734)
	_, err := FormatFloatDisplay2Value(in, DefaultCoinPrecision)
	assert.NotNil(t, err)

	//小数位超过配置精度
	in = float64(1123456.123456781)
	_, err = FormatFloatDisplay2Value(in, DefaultCoinPrecision)
	assert.NotNil(t, err)

	//小数位超过配置精度
	in = float64(112345678.12345)
	_, err = FormatFloatDisplay2Value(in, 1e4)
	assert.NotNil(t, err)

	//没有小数位，按精度扩展
	in = float64(112345678)
	v, err := FormatFloatDisplay2Value(in, 1e4)
	assert.Nil(t, err)
	assert.Equal(t, int64(1123456780000), v)

	//最大值
	in = float64(9999999999999.9)
	v, err = FormatFloatDisplay2Value(in, 1e1)
	assert.Nil(t, err)
	assert.Equal(t, int64(99999999999999), v)
	//最大值
	in = float64(999999999999999)
	v, err = FormatFloatDisplay2Value(in, 1)
	assert.Nil(t, err)
	assert.Equal(t, int64(999999999999999), v)

	//有小数位扩展
	in = float64(112345678.12)
	v, err = FormatFloatDisplay2Value(in, 1e4)
	assert.Nil(t, err)
	assert.Equal(t, int64(1123456781200), v)

	//最大小数位扩展
	in = float64(MaxCoin)
	v, err = FormatFloatDisplay2Value(in, DefaultCoinPrecision)
	assert.Nil(t, err)
	assert.Equal(t, MaxCoin*100000000, v)

	in = float64(0)
	v, err = FormatFloatDisplay2Value(in, DefaultCoinPrecision)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), v)

	//测试小数位扩展
	in = float64(0.1)
	v, err = FormatFloatDisplay2Value(in, 1e4)
	assert.Nil(t, err)
	assert.Equal(t, int64(1000), v)

	in = float64(9999999.1234567)
	v, err = FormatFloatDisplay2Value(in, DefaultCoinPrecision)
	assert.Nil(t, err)
	assert.Equal(t, int64(999999912340000), v)

	in = float64(999999999.1)
	v, err = FormatFloatDisplay2Value(in, 1e5)
	assert.Nil(t, err)
	assert.Equal(t, int64(99999999910000), v)

	in = float64(999999999)
	v, err = FormatFloatDisplay2Value(in, 1e5)
	assert.Nil(t, err)
	assert.Equal(t, int64(99999999900000), v)

}

func TestGetParaExecName(t *testing.T) {

	driverName := GetParaExecName([]byte("user.p.para.none"))
	assert.Equal(t, "none", string(driverName))
	driverName = GetParaExecName([]byte("user.p.para.user.none"))
	assert.Equal(t, "user.none", string(driverName))
	driverName = GetRealExecName([]byte("user.p.para.user.none"))
	assert.Equal(t, "none", string(driverName))
}
