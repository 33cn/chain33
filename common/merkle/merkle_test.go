// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package merkle

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/33cn/chain33/common"
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

//测试三个交易的roothash以及branch
//"height": 99960,
//  "merkleroot": "34d5a57822efa653019edfee29b9586a0d0d807572275b45f39a7e9c25614bf9",
//  "tx": [
//    "f89c65bdcd695e4acc621256085f20d7c093097e04a1ce34b606a5829cbaf2c6",
//    "1818bef9c6aeed09de0ed999b5f2868b3555084437e1c63f29d5f37b69bb214f",
//    "d43a40a2db5bad2bd176c27911ed86d97bff734425953b19c8cf77910b21020d"
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

//原来的版本更快，这个方案只是做一个性能测试的对比
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
