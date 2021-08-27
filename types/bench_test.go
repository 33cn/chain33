// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types_test

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/common"
	_ "github.com/33cn/chain33/system/dapp/coins"
	coinstypes "github.com/33cn/chain33/system/dapp/coins/types"
	_ "github.com/33cn/chain33/system/dapp/init"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func Benchmark_CopyProto(b *testing.B) {

	addr, priv := util.Genaddress()
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateCoinsTx(cfg, priv, addr, 1)
	var (
		tx1 types.Transaction
	)
	b.ResetTimer()

	b.Run("copy tx", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			types.CloneTx(tx)
		}
	})

	b.Run("merge tx", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			proto.Merge(&tx1, tx)
		}

	})
	b.Run("clone tx", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			proto.Clone(tx)
		}

	})
}

func Test_ReflectProto(t *testing.T) {

	action := &coinstypes.CoinsAction{}

	msg := action.ProtoReflect()
	transfer := &types.AssetsTransfer{
		Cointoken: "bty",
	}

	des := transfer.ProtoReflect().Descriptor()
	msg1 := dynamicpb.NewMessage(des)
	proto.Merge(msg1, transfer)
	require.Equal(t, common.ToHex(types.Encode(msg1)), common.ToHex(types.Encode(transfer)))
	action2 := dynamicpb.NewMessage(msg.Descriptor())
	field := msg.Descriptor().Fields().Get(0)
	action2.Set(field, protoreflect.ValueOf(proto.Message(msg1)))
	proto.Merge(action, action2)
	require.Equal(t, "bty", action.GetTransfer().GetCointoken())

	req := &types.ReqAccountList{}
	buf := types.Encode(req)
	require.NotNil(t, buf)
	require.Equal(t, 0, len(buf))

	_, priv := util.Genaddress()
	println(common.ToHex(priv.PubKey().Bytes()))
}

func encodeWithBuf(msg types.Message) []byte {

	buf := &proto.Buffer{}
	buf.SetDeterministic(true)
	err := buf.Marshal(msg)
	if err != nil {
		panic(err.Error())
	}
	return buf.Bytes()
}

func Benchmark_EncodeProto(b *testing.B) {

	_, priv := util.Genaddress()
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	block := util.CreateNoneBlock(cfg, priv, 100)

	b.ResetTimer()
	b.Run("directEncode", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			proto.Marshal(block)
		}
	})

	b.Run("bufEncode", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			encodeWithBuf(block)
		}

	})
}

func Test_MarshalProtoCompatibility(t *testing.T) {

	txV2 := &types.Transaction{}
	txHexV1 := "0a05636f696e73122a18010a2610642222313256594b4345584d65663334763755627541364b566f706d7535326f526674397a1a6e080112210364038c653482e649f1bff8bb41a13fde2063687d66ebfefef61bd61308591ae91a4730450221009c85fa086e14ed95f1d7482d0b8521bb393e8777116ff53937f7efbdd1f4f7a00220237c19a669db8713e6e66f65fa7c389bee0dfe4dd4f7f9a371f451fa0808031e20a08d0630d0a6f9cbe2f59994063a22313256594b4345584d65663334763755627541364b566f706d7535326f526674397a5821"

	txBytes, err := common.FromHex(txHexV1)

	require.Nil(t, err)
	err = types.Decode(txBytes, txV2)
	require.Nil(t, err)
	require.Equal(t, txHexV1, hex.EncodeToString(types.Encode(txV2)))
}

func Test_ProtoOneOfReflect(t *testing.T) {

	action := &coinstypes.CoinsAction{}
	des := action.ProtoReflect().Descriptor()
	require.Equal(t, 1, des.Oneofs().Len())

	tx := &types.Transaction{}
	des = tx.ProtoReflect().Descriptor()
	require.Equal(t, 0, des.Oneofs().Len())
}
