// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package merkle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

//测试两个交易的roothash以及branch.获取bitcoin的99997 block作为验证
// "height": 99997,
//  "merkleroot": "5140e5972f672bf8e81bc189894c55a410723b095716eaeec845490aed785f0e",
//  "tx": [
//    "b86f5ef1da8ddbdb29ec269b535810ee61289eeac7bf2b2523b494551f03897c",
//    "80c6f121c3e9fe0a59177e49874d8c703cbadee0700a782e4002e87d862373c6"

func Test_TwoTxMerkle(t *testing.T) {
	RootHash := "5140e5972f672bf8e81bc189894c55a410723b095716eaeec845490aed785f0e"

	tx0string := "b86f5ef1da8ddbdb29ec269b535810ee61289eeac7bf2b2523b494551f03897c"
	tx1string := "80c6f121c3e9fe0a59177e49874d8c703cbadee0700a782e4002e87d862373c6"

	t.Logf("Test_TwoTxMerkle bitcoin roothash :%s", RootHash)

	rootHash, err := NewHashFromStr(RootHash)
	if err != nil {
		t.Errorf("NewHashFromStr RootHash err:%s", err.Error())
	}
	rootHashbyte := rootHash.CloneBytes()

	tx0hash, err := NewHashFromStr(tx0string)
	if err != nil {
		t.Errorf("NewHashFromStr tx0string err:%s", err.Error())
	}
	tx0byte := tx0hash.CloneBytes()

	tx1hash, err := NewHashFromStr(tx1string)
	if err != nil {
		t.Errorf("NewHashFromStr tx1string err:%s", err.Error())
	}
	tx1byte := tx1hash.CloneBytes()

	leaves := make([][]byte, 2)

	leaves[0] = tx0byte
	leaves[1] = tx1byte

	bitroothash := GetMerkleRoot(leaves)

	bitroothashstr, err := NewHash(bitroothash)
	if err == nil {
		t.Logf("Test_TwoTxMerkle GetMerkleRoot roothash :%s", bitroothashstr.String())
	}
	if !bytes.Equal(rootHashbyte, bitroothash) {
		t.Errorf("Test_TwoTxMerkle  rootHashbyte :%v,GetMerkleRoot :%v", rootHashbyte, bitroothash)
		return
	}

	for txindex := 0; txindex < 2; txindex++ {
		roothashd, branchs := GetMerkleRootAndBranch(leaves, uint32(txindex))

		brroothashstr, err := NewHash(roothashd)
		if err == nil {
			t.Logf("Test_TwoTxMerkle GetMerkleRootAndBranch roothash :%s", brroothashstr.String())
		}

		brroothash := GetMerkleRootFromBranch(branchs, leaves[txindex], uint32(txindex))
		if bytes.Equal(bitroothash, brroothash) && bytes.Equal(rootHashbyte, brroothash) {
			brRoothashstr, err := NewHash(brroothash)
			if err == nil {
				t.Logf("Test_TwoTxMerkle GetMerkleRootFromBranch roothash :%s", brRoothashstr.String())
			}
			t.Logf("Test_TwoTxMerkle bitroothash == brroothash :%d", txindex)
		}
	}
}

// 测试三个交易的roothash以及branch
// "height": 99960,
//
//	"merkleroot": "34d5a57822efa653019edfee29b9586a0d0d807572275b45f39a7e9c25614bf9",
//	"tx": [
//	  "f89c65bdcd695e4acc621256085f20d7c093097e04a1ce34b606a5829cbaf2c6",
//	  "1818bef9c6aeed09de0ed999b5f2868b3555084437e1c63f29d5f37b69bb214f",
//	  "d43a40a2db5bad2bd176c27911ed86d97bff734425953b19c8cf77910b21020d"
func Test_OddTxMerkle(t *testing.T) {

	RootHash := "34d5a57822efa653019edfee29b9586a0d0d807572275b45f39a7e9c25614bf9"

	tx0string := "f89c65bdcd695e4acc621256085f20d7c093097e04a1ce34b606a5829cbaf2c6"
	tx1string := "1818bef9c6aeed09de0ed999b5f2868b3555084437e1c63f29d5f37b69bb214f"
	tx2string := "d43a40a2db5bad2bd176c27911ed86d97bff734425953b19c8cf77910b21020d"

	t.Logf("Test_OddTxMerkle bitcoin roothash :%s", RootHash)

	rootHash, err := NewHashFromStr(RootHash)
	if err != nil {
		t.Errorf("NewHashFromStr RootHash err:%s", err.Error())
	}
	rootHashbyte := rootHash.CloneBytes()

	tx0hash, err := NewHashFromStr(tx0string)
	if err != nil {
		t.Errorf("NewHashFromStr tx0string err:%s", err.Error())
	}
	tx0byte := tx0hash.CloneBytes()

	tx1hash, err := NewHashFromStr(tx1string)
	if err != nil {
		t.Errorf("NewHashFromStr tx1string err:%s", err.Error())
	}
	tx1byte := tx1hash.CloneBytes()

	tx2hash, err := NewHashFromStr(tx2string)
	if err != nil {
		t.Errorf("NewHashFromStr tx2string err:%s", err.Error())
	}
	tx2byte := tx2hash.CloneBytes()

	leaves := make([][]byte, 3)

	leaves[0] = tx0byte
	leaves[1] = tx1byte
	leaves[2] = tx2byte

	bitroothash := GetMerkleRoot(leaves)

	bitroothashstr, err := NewHash(bitroothash)
	if err == nil {
		t.Logf("Test_OddTxMerkle GetMerkleRoot roothash :%s", bitroothashstr.String())
	}
	if !bytes.Equal(rootHashbyte, bitroothash) {
		t.Errorf("Test_OddTxMerkle  rootHashbyte :%v,GetMerkleRoot :%v", rootHashbyte, bitroothash)
		return
	}

	for txindex := 0; txindex < 3; txindex++ {
		branchs := GetMerkleBranch(leaves, uint32(txindex))

		brroothash := GetMerkleRootFromBranch(branchs, leaves[txindex], uint32(txindex))
		if bytes.Equal(bitroothash, brroothash) && bytes.Equal(rootHashbyte, brroothash) {
			brRoothashstr, err := NewHash(brroothash)
			if err == nil {
				t.Logf("Test_OddTxMerkle GetMerkleRootFromBranch roothash :%s", brRoothashstr.String())
			}
			t.Logf("Test_OddTxMerkle bitroothash == brroothash :%d", txindex)
		}
	}
}

//测试六个交易的roothash以及branch
//		"height": 99974,
//  	"merkleroot": "272066470ccf8ee2bb6f48f263c5b2ffc56813be40001c4b33fec0677d69e3cc",
//  	"tx": [
//    "dc5d3f45eeffaab3a71e9930c80a34f5da7ee4cdbc6e7270e1c6d3397f835836",
//    "0ed98b4429bfbf78cfb1e87b9c16cc6641df0a8b2df861c13dc987785912cc48",
//    "8108da17be5960b009de33b60fcdfd5856abbd8b7c543afde31b0601b6e2aeea",
//    "ecffa48ff29b13b295c25d97987f4899ed4f607691c316b0fad685fa1ab268d9",
//    "b57f79efed1d0e999495a34840aa7b41e15b43627225849d83effe56152df4e7",
//    "8ded96ae2a609555df2390faa2f001b04e6e0ba8a60c69f9b432c9637157d766"

func Test_SixTxMerkle(t *testing.T) {
	RootHash := "272066470ccf8ee2bb6f48f263c5b2ffc56813be40001c4b33fec0677d69e3cc"

	tx0string := "dc5d3f45eeffaab3a71e9930c80a34f5da7ee4cdbc6e7270e1c6d3397f835836"
	tx1string := "0ed98b4429bfbf78cfb1e87b9c16cc6641df0a8b2df861c13dc987785912cc48"
	tx2string := "8108da17be5960b009de33b60fcdfd5856abbd8b7c543afde31b0601b6e2aeea"
	tx3string := "ecffa48ff29b13b295c25d97987f4899ed4f607691c316b0fad685fa1ab268d9"
	tx4string := "b57f79efed1d0e999495a34840aa7b41e15b43627225849d83effe56152df4e7"
	tx5string := "8ded96ae2a609555df2390faa2f001b04e6e0ba8a60c69f9b432c9637157d766"

	t.Logf("Test_SixTxMerkle bitcoin roothash :%s", RootHash)

	rootHash, err := NewHashFromStr(RootHash)
	if err != nil {
		t.Errorf("NewHashFromStr RootHash err:%s", err.Error())
	}
	rootHashbyte := rootHash.CloneBytes()

	tx0hash, err := NewHashFromStr(tx0string)
	if err != nil {
		t.Errorf("NewHashFromStr tx0string err:%s", err.Error())
	}
	tx0byte := tx0hash.CloneBytes()

	tx1hash, err := NewHashFromStr(tx1string)
	if err != nil {
		t.Errorf("NewHashFromStr tx1string err:%s", err.Error())
	}
	tx1byte := tx1hash.CloneBytes()

	tx2hash, err := NewHashFromStr(tx2string)
	if err != nil {
		t.Errorf("NewHashFromStr tx2string err:%s", err.Error())
	}
	tx2byte := tx2hash.CloneBytes()

	tx3hash, err := NewHashFromStr(tx3string)
	if err != nil {
		t.Errorf("NewHashFromStr tx3string err:%s", err.Error())
	}
	tx3byte := tx3hash.CloneBytes()

	tx4hash, err := NewHashFromStr(tx4string)
	if err != nil {
		t.Errorf("NewHashFromStr tx4string err:%s", err.Error())
	}
	tx4byte := tx4hash.CloneBytes()

	tx5hash, err := NewHashFromStr(tx5string)
	if err != nil {
		t.Errorf("NewHashFromStr tx5string err:%s", err.Error())
	}
	tx5byte := tx5hash.CloneBytes()

	leaves := make([][]byte, 6)

	leaves[0] = tx0byte
	leaves[1] = tx1byte
	leaves[2] = tx2byte
	leaves[3] = tx3byte

	leaves[4] = tx4byte
	leaves[5] = tx5byte

	bitroothash := GetMerkleRoot(leaves)

	bitroothashstr, err := NewHash(bitroothash)
	if err == nil {
		t.Logf("Test_SixTxMerkle GetMerkleRoot roothash :%s", bitroothashstr.String())
	}
	if !bytes.Equal(rootHashbyte, bitroothash) {
		t.Errorf("Test_SixTxMerkle  rootHashbyte :%v,GetMerkleRoot :%v", rootHashbyte, bitroothash)
		return
	}

	for txindex := 0; txindex < 6; txindex++ {
		branchs := GetMerkleBranch(leaves, uint32(txindex))

		brroothash := GetMerkleRootFromBranch(branchs, leaves[txindex], uint32(txindex))
		if bytes.Equal(bitroothash, brroothash) && bytes.Equal(rootHashbyte, brroothash) {
			brRoothashstr, err := NewHash(brroothash)
			if err == nil {
				t.Logf("Test_SixTxMerkle GetMerkleRootFromBranch roothash :%s", brRoothashstr.String())
			}
			t.Logf("Test_SixTxMerkle bitroothash == brroothash :%d", txindex)
		}
	}
}

func BenchmarkHashTwo(b *testing.B) {
	b.ReportAllocs()
	left := common.GetRandBytes(32, 32)
	right := common.GetRandBytes(32, 32)
	for i := 0; i < b.N; i++ {
		getHashFromTwoHash(left, right)
	}
}

func BenchmarkHashTwo2(b *testing.B) {
	b.ReportAllocs()
	left := common.GetRandBytes(32, 32)
	right := common.GetRandBytes(32, 32)
	cache := make([]byte, 64)
	for i := 0; i < b.N; i++ {
		GetHashFromTwoHash(cache, left, right)
	}
}

// 原来的版本更快，这个方案只是做一个性能测试的对比
func getHashFromTwoHash(left []byte, right []byte) []byte {
	if left == nil || right == nil {
		return nil
	}
	h := sha256.New()
	h.Write(left)
	h.Write(right)
	hash1 := h.Sum(nil)
	h.Reset()
	h.Write(hash1)
	return h.Sum(nil)
}

//优化办法:
//1. 减少内存分配
//2. 改进算法

var benchlen = 100000

func BenchmarkGetMerkelRoot(b *testing.B) {
	b.ReportAllocs()
	var hashlist [][]byte
	for i := 0; i < benchlen; i++ {
		key := common.GetRandBytes(32, 32)
		hashlist = append(hashlist, key)
	}
	var prevroot []byte
	for i := 0; i < b.N; i++ {
		calc := make([][]byte, len(hashlist))
		copy(calc, hashlist)
		newroot := GetMerkleRoot(calc)
		if prevroot != nil && !bytes.Equal(prevroot, newroot) {
			b.Error("root is not the same")
		}
		prevroot = newroot
	}
}

func BenchmarkGetMerkelRoot2(b *testing.B) {
	b.ReportAllocs()
	var hashlist [][]byte
	for i := 0; i < benchlen; i++ {
		key := common.GetRandBytes(32, 32)
		hashlist = append(hashlist, key)
	}
	var prevroot []byte
	for i := 0; i < b.N; i++ {
		calc := make([][]byte, len(hashlist))
		copy(calc, hashlist)
		newroot, _, _ := Computation(calc, 1, 0)
		if prevroot != nil && !bytes.Equal(prevroot, newroot) {
			b.Error("root is not the same")
		}
		prevroot = newroot
	}
}

func TestGetMerkelRoot1(t *testing.T) {
	for i := 0; i < 2000; i++ {
		ok := testGetMerkelRoot1(t, i)
		if !ok {
			t.Error("calc merkel root error", i)
			return
		}
	}
}

func testGetMerkelRoot1(t *testing.T, testlen int) bool {
	var hashlist [][]byte
	for i := 0; i < testlen; i++ {
		key := sha256.Sum256([]byte(fmt.Sprint(i)))
		hashlist = append(hashlist, key[:])
	}
	hash1 := GetMerkleRoot(hashlist)

	hashlist = nil
	for i := 0; i < testlen; i++ {
		key := sha256.Sum256([]byte(fmt.Sprint(i)))
		hashlist = append(hashlist, key[:])
	}
	hash2 := getMerkleRoot(hashlist)
	if !bytes.Equal(hash1, hash2) {
		println("failed1")
		return false
	}

	hashlist = nil
	for i := 0; i < testlen; i++ {
		key := sha256.Sum256([]byte(fmt.Sprint(i)))
		hashlist = append(hashlist, key[:])
	}
	hash3, _, _ := Computation(hashlist, 1, 0)
	if !bytes.Equal(hash1, hash3) {
		println("failed2")
		return false
	}
	return true
}

func TestLog2(t *testing.T) {
	assert.Equal(t, log2(0), 0)
	assert.Equal(t, log2(1), 1)
	assert.Equal(t, log2(2), 1)
	assert.Equal(t, log2(3), 1)
	assert.Equal(t, log2(4), 2)
	assert.Equal(t, log2(5), 2)
	assert.Equal(t, log2(6), 2)
	assert.Equal(t, log2(7), 2)
	assert.Equal(t, log2(8), 3)
	assert.Equal(t, log2(256), 8)
}

func TestCalcMainMerkleRoot(t *testing.T) {

	cfg := types.NewChain33Config(types.GetDefaultCfgstring())

	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx2 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630de92c3828ad194b26d3a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx3 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630b0d6c895c4d28efe5d3a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	tx21, _ := hex.DecodeString(tx2)
	tx31, _ := hex.DecodeString(tx3)

	var txList types.Transactions
	tx12 := &types.Transaction{}
	types.Decode(tx11, tx12)
	tx22 := &types.Transaction{}
	types.Decode(tx21, tx22)
	tx32 := &types.Transaction{}
	types.Decode(tx31, tx32)

	//构建三笔单个交易并添加到交易列表中
	tx12.Execer = []byte("hashlock")
	tx22.Execer = []byte("voken")
	tx32.Execer = []byte("coins")
	txList.Txs = append(txList.Txs, tx12)
	txList.Txs = append(txList.Txs, tx22)
	txList.Txs = append(txList.Txs, tx32)

	//构建主链的交易组并添加到交易列表中
	tx111, tx221, tx321 := modifyTxExec(tx12, tx22, tx32, "paracross", "game", "guess")
	group, err := types.CreateTxGroup([]*types.Transaction{tx111, tx221, tx321}, 1000000)
	if err != nil {
		t.Error(err)
		return
	}
	groupTx := group.Tx()
	txGroup, err := groupTx.GetTxGroup()
	if err != nil {
		t.Error(err)
		return
	}
	txList.Txs = append(txList.Txs, txGroup.GetTxs()...)

	//构建三笔不同平行链的单笔交易
	tx1111, tx2211, tx3211 := modifyTxExec(tx12, tx22, tx32, "user.p.test.js", "user.p.para.lottery", "user.p.fuzamei.norm")

	txList.Txs = append(txList.Txs, tx1111)
	txList.Txs = append(txList.Txs, tx2211)
	txList.Txs = append(txList.Txs, tx3211)

	//构建user.p.test.平行链的交易组并添加到交易列表中
	tx1112, tx2212, tx3212 := modifyTxExec(tx12, tx22, tx32, "user.p.test.evm", "user.p.test.relay", "user.p.test.ticket")
	group, err = types.CreateTxGroup([]*types.Transaction{tx1112, tx2212, tx3212}, 1000000)
	if err != nil {
		t.Error(err)
		return
	}
	groupTx = group.Tx()
	txGroup, err = groupTx.GetTxGroup()
	if err != nil {
		t.Error(err)
		return
	}
	txList.Txs = append(txList.Txs, txGroup.GetTxs()...)

	//构建user.p.para.平行链的交易组并添加到交易列表中
	tx1113, tx2213, tx3213 := modifyTxExec(tx12, tx22, tx32, "user.p.para.coins", "user.p.para.paracross", "user.p.para.pokerbull")

	group, err = types.CreateTxGroup([]*types.Transaction{tx1113, tx2213, tx3213}, 1000000)
	if err != nil {
		t.Error(err)
		return
	}
	groupTx = group.Tx()
	txGroup, err = groupTx.GetTxGroup()
	if err != nil {
		t.Error(err)
		return
	}
	txList.Txs = append(txList.Txs, txGroup.GetTxs()...)

	//构建user.p.fuzamei.平行链的交易组并添加到交易列表中
	tx1114, tx2214, tx3214 := modifyTxExec(tx12, tx22, tx32, "user.p.fuzamei.norm", "user.p.fuzamei.coins", "user.p.fuzamei.retrieve")

	group, err = types.CreateTxGroup([]*types.Transaction{tx1114, tx2214, tx3214}, 1000000)
	if err != nil {
		t.Error(err)
		return
	}
	groupTx = group.Tx()
	txGroup, err = groupTx.GetTxGroup()
	if err != nil {
		t.Error(err)
		return
	}
	txList.Txs = append(txList.Txs, txGroup.GetTxs()...)

	//构造一些主链交易的合约名排序在user后面的交易
	tx1115, tx2215, tx3215 := modifyTxExec(tx12, tx22, tx32, "varacross", "wame", "zuess")
	txList.Txs = append(txList.Txs, tx1115)
	txList.Txs = append(txList.Txs, tx2215)
	txList.Txs = append(txList.Txs, tx3215)

	sorTxList := types.TransactionSort(txList.Txs)

	assert.Equal(t, len(txList.Txs), len(sorTxList))

	for _, sorttx := range sorTxList {
		var equal bool
		sortHash := sorttx.Hash()
		for _, tx := range txList.Txs {
			txHash := tx.Hash()
			if bytes.Equal(sortHash, txHash) {
				equal = true
				break
			}
		}
		assert.Equal(t, equal, true)
	}

	var mixChainHashes [][]byte

	//主链子roothash[0-9]
	oldMixMainHash := calcSingleLayerMerkleRoot(sorTxList[0:9])
	mixChainHashes = append(mixChainHashes, oldMixMainHash)

	// fuzamei平行链的子roothash[9-13]
	oldMixFuzameiHash := calcSingleLayerMerkleRoot(sorTxList[9:13])
	mixChainHashes = append(mixChainHashes, oldMixFuzameiHash)

	// para平行链的子roothash
	oldMixParaHash := calcSingleLayerMerkleRoot(sorTxList[13:17])
	mixChainHashes = append(mixChainHashes, oldMixParaHash)

	// test平行链的子roothash
	oldMixTestHash := calcSingleLayerMerkleRoot(sorTxList[17:21])
	mixChainHashes = append(mixChainHashes, oldMixTestHash)

	oldMixChainHash := GetMerkleRoot(mixChainHashes)

	newMixChainHash, childMixChainHash := CalcMultiLayerMerkleInfo(cfg, 1, sorTxList)

	assert.Equal(t, newMixChainHash, oldMixChainHash)

	assert.Equal(t, len(childMixChainHash), 4)
	//主链子roothash的检测
	assert.Equal(t, childMixChainHash[0].ChildHash, oldMixMainHash)
	assert.Equal(t, childMixChainHash[0].StartIndex, int32(0))
	assert.Equal(t, childMixChainHash[0].Title, types.MainChainName)
	assert.Equal(t, childMixChainHash[0].GetTxCount(), int32(9))
	//fuzamei平行链子roothash的检测
	assert.Equal(t, childMixChainHash[1].ChildHash, oldMixFuzameiHash)
	assert.Equal(t, childMixChainHash[1].StartIndex, int32(9))
	assert.Equal(t, childMixChainHash[1].Title, "user.p.fuzamei.")
	assert.Equal(t, childMixChainHash[1].GetTxCount(), int32(4))
	//para平行链子roothash的检测
	assert.Equal(t, childMixChainHash[2].ChildHash, oldMixParaHash)
	assert.Equal(t, childMixChainHash[2].StartIndex, int32(13))
	assert.Equal(t, childMixChainHash[2].Title, "user.p.para.")
	assert.Equal(t, childMixChainHash[2].GetTxCount(), int32(4))
	//test平行链子roothash的检测
	assert.Equal(t, childMixChainHash[3].ChildHash, oldMixTestHash)
	assert.Equal(t, childMixChainHash[3].StartIndex, int32(17))
	assert.Equal(t, childMixChainHash[3].Title, "user.p.test.")
	assert.Equal(t, childMixChainHash[3].GetTxCount(), int32(4))

	var leaves [][]byte
	for _, childChain := range childMixChainHash {
		leaves = append(leaves, childChain.ChildHash)
	}

	//获取主链子roothash的MerkleBranch
	for index, childChain := range childMixChainHash {
		childChainBranch := GetMerkleBranch(leaves, uint32(index))
		rootHash := GetMerkleRootFromBranch(childChainBranch, childChain.ChildHash, uint32(index))
		assert.Equal(t, rootHash, newMixChainHash)
	}

	//构建全是主链的交易列表
	var txMainList types.Transactions
	tx51111, tx52211, tx53211 := modifyTxExec(tx12, tx22, tx32, "user.write", "coins", "ticket")
	txMainList.Txs = append(txMainList.Txs, tx51111)
	txMainList.Txs = append(txMainList.Txs, tx52211)
	txMainList.Txs = append(txMainList.Txs, tx53211)

	tx61111, tx62211, tx63211 := modifyTxExec(tx12, tx22, tx32, "ajs", "zottery", "norm")
	group, err = types.CreateTxGroup([]*types.Transaction{tx61111, tx62211, tx63211}, 1000000)
	if err != nil {
		t.Error(err)
		return
	}
	groupTx = group.Tx()
	txGroup, err = groupTx.GetTxGroup()
	if err != nil {
		t.Error(err)
		return
	}
	txMainList.Txs = append(txMainList.Txs, txGroup.GetTxs()...)

	sorTxMainList := types.TransactionSort(txMainList.Txs)

	newrootHash, childHash := CalcMultiLayerMerkleInfo(cfg, 1, sorTxMainList)
	oldrootHash := calcSingleLayerMerkleRoot(sorTxMainList)
	assert.Equal(t, newrootHash, oldrootHash)
	assert.Equal(t, childHash[0].ChildHash, oldrootHash)
	assert.Equal(t, childHash[0].StartIndex, int32(0))
	assert.Equal(t, childHash[0].Title, types.MainChainName)
	assert.Equal(t, childHash[0].GetTxCount(), int32(6))

	roothashTem := CalcMerkleRoot(cfg, 1, sorTxMainList)
	assert.Equal(t, roothashTem, oldrootHash)
	//构建全是同一个平行链的交易列表
	var txParaTestList types.Transactions
	tx71111, tx72211, tx73211 := modifyTxExec(tx12, tx22, tx32, "user.p.test.js", "user.p.test.lottery", "user.p.test.norm")
	txParaTestList.Txs = append(txParaTestList.Txs, tx71111)
	txParaTestList.Txs = append(txParaTestList.Txs, tx72211)
	txParaTestList.Txs = append(txParaTestList.Txs, tx73211)

	tx81111, tx82211, tx83211 := modifyTxExec(tx12, tx22, tx32, "user.p.test.coins", "user.p.test.token", "user.p.test.none")

	group, err = types.CreateTxGroup([]*types.Transaction{tx81111, tx82211, tx83211}, 1000000)
	if err != nil {
		t.Error(err)
		return
	}
	groupTx = group.Tx()
	txGroup, err = groupTx.GetTxGroup()
	if err != nil {
		t.Error(err)
		return
	}
	txParaTestList.Txs = append(txParaTestList.Txs, txGroup.GetTxs()...)

	sorTxParaTestList := types.TransactionSort(txParaTestList.Txs)

	newPararootHash, childParaHash := CalcMultiLayerMerkleInfo(cfg, 1, sorTxParaTestList)
	oldPararootHash := calcSingleLayerMerkleRoot(sorTxParaTestList)
	assert.Equal(t, newPararootHash, oldPararootHash)

	assert.Equal(t, childParaHash[0].ChildHash, oldPararootHash)
	assert.Equal(t, childParaHash[0].StartIndex, int32(0))
	assert.Equal(t, childParaHash[0].Title, "user.p.test.")
	assert.Equal(t, childParaHash[0].GetTxCount(), int32(6))

	//构造一个主链和一个平行链的交易列表
	var txMPList types.Transactions
	tx91111, tx92211, tx93211 := modifyTxExec(tx12, tx22, tx32, "coins", "user.p.test.lottery", "token")
	txMPList.Txs = append(txMPList.Txs, tx91111)
	txMPList.Txs = append(txMPList.Txs, tx92211)
	txMPList.Txs = append(txMPList.Txs, tx93211)

	sorTxMPList := types.TransactionSort(txMPList.Txs)

	var hashes [][]byte
	oldMrootHash := calcSingleLayerMerkleRoot(sorTxMPList[0:2])
	oldProotHash := calcSingleLayerMerkleRoot(sorTxMPList[2:])
	hashes = append(hashes, oldMrootHash)
	hashes = append(hashes, oldProotHash)
	oldMProotHash := GetMerkleRoot(hashes)

	newMProotHash, childMainParaHash := CalcMultiLayerMerkleInfo(cfg, 1, sorTxMPList)

	assert.Equal(t, newMProotHash, oldMProotHash)

	assert.Equal(t, len(childMainParaHash), 2)
	assert.Equal(t, childMainParaHash[0].ChildHash, oldMrootHash)
	assert.Equal(t, childMainParaHash[0].StartIndex, int32(0))
	assert.Equal(t, childMainParaHash[0].Title, types.MainChainName)
	assert.Equal(t, childMainParaHash[0].GetTxCount(), int32(2))

	assert.Equal(t, childMainParaHash[1].ChildHash, oldProotHash)
	assert.Equal(t, childMainParaHash[1].StartIndex, int32(2))
	assert.Equal(t, childMainParaHash[1].Title, "user.p.test.")
	assert.Equal(t, childMainParaHash[1].GetTxCount(), int32(1))

	//构造一个主链和三个平行链的交易列表
	var txMThreePList types.Transactions
	tx111111, tx112211, tx113211 := modifyTxExec(tx12, tx22, tx32, "coins", "user.p.test.none", "user.p.para.oracle")
	txMThreePList.Txs = append(txMThreePList.Txs, tx111111)
	txMThreePList.Txs = append(txMThreePList.Txs, tx112211)
	txMThreePList.Txs = append(txMThreePList.Txs, tx113211)

	tx211111, tx212211, tx213211 := modifyTxExec(tx12, tx22, tx32, "valnode", "user.p.test.lottery", "user.p.para.relay")
	txMThreePList.Txs = append(txMThreePList.Txs, tx211111)
	txMThreePList.Txs = append(txMThreePList.Txs, tx212211)
	txMThreePList.Txs = append(txMThreePList.Txs, tx213211)

	tx311111, tx312211, tx313211 := modifyTxExec(tx12, tx22, tx32, "unfreeze", "user.p.test.hashlock", "user.p.para.echo")
	txMThreePList.Txs = append(txMThreePList.Txs, tx311111)
	txMThreePList.Txs = append(txMThreePList.Txs, tx312211)
	txMThreePList.Txs = append(txMThreePList.Txs, tx313211)

	sorTxMThreePList := types.TransactionSort(txMThreePList.Txs)

	var mThreePhashes [][]byte

	//主链的三笔交易的子roothash
	mainRootHash := calcSingleLayerMerkleRoot(sorTxMThreePList[0:3])
	mThreePhashes = append(mThreePhashes, mainRootHash)

	//para平行链的三笔交易的子roothash
	paraRootHash := calcSingleLayerMerkleRoot(sorTxMThreePList[3:6])
	mThreePhashes = append(mThreePhashes, paraRootHash)

	//test平行链的三笔交易的子roothash
	testRootHash := calcSingleLayerMerkleRoot(sorTxMThreePList[6:])
	mThreePhashes = append(mThreePhashes, testRootHash)

	oldMThreeProotHash := GetMerkleRoot(mThreePhashes)

	newMThreeProotHash, childMThreePHash := CalcMultiLayerMerkleInfo(cfg, 1, sorTxMThreePList)

	tempRootHash := CalcMerkleRoot(cfg, 1, sorTxMThreePList)
	assert.Equal(t, newMThreeProotHash, tempRootHash)

	assert.Equal(t, newMThreeProotHash, oldMThreeProotHash)

	assert.Equal(t, len(childMThreePHash), 3)
	assert.Equal(t, childMThreePHash[0].ChildHash, mainRootHash)
	assert.Equal(t, childMThreePHash[0].StartIndex, int32(0))
	assert.Equal(t, childMThreePHash[0].Title, types.MainChainName)
	assert.Equal(t, childMThreePHash[0].GetTxCount(), int32(3))

	assert.Equal(t, childMThreePHash[1].ChildHash, paraRootHash)
	assert.Equal(t, childMThreePHash[1].StartIndex, int32(3))
	assert.Equal(t, childMThreePHash[1].Title, "user.p.para.")
	assert.Equal(t, childMThreePHash[1].GetTxCount(), int32(3))

	assert.Equal(t, childMThreePHash[2].ChildHash, testRootHash)
	assert.Equal(t, childMThreePHash[2].StartIndex, int32(6))
	assert.Equal(t, childMThreePHash[2].Title, "user.p.test.")
	assert.Equal(t, childMThreePHash[2].GetTxCount(), int32(3))

	roothash1, childHash1 := CalcMultiLayerMerkleInfo(cfg, 0, sorTxMThreePList)
	assert.Nil(t, roothash1)
	assert.Nil(t, childHash1)
}

func modifyTxExec(tx1, tx2, tx3 *types.Transaction, tx1exec, tx2exec, tx3exec string) (*types.Transaction, *types.Transaction, *types.Transaction) {
	tx11 := types.CloneTx(tx1)
	tx12 := types.CloneTx(tx2)
	tx13 := types.CloneTx(tx3)

	tx11.Execer = []byte(tx1exec)
	tx12.Execer = []byte(tx2exec)
	tx13.Execer = []byte(tx3exec)

	return tx11, tx12, tx13
}
