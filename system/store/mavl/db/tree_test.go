// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mavl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"sort"
	"testing"

	"os"

	. "github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	log.SetLogLevel("error")
}

const (
	strChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" // 62 characters
)

// Constructs an alphanumeric string of given length.
func RandStr(length int) string {
	chars := []byte{}
MAIN_LOOP:
	for {
		val := rand.Int63()
		for i := 0; i < 10; i++ {
			v := int(val & 0x3f) // rightmost 6 bits
			if v >= 62 {         // only 62 characters in strChars
				val >>= 6
				continue
			} else {
				chars = append(chars, strChars[v])
				if len(chars) == length {
					break MAIN_LOOP
				}
				val >>= 6
			}
		}
	}

	return string(chars)
}

func randstr(length int) string {
	return RandStr(length)
}

func RandInt32() uint32 {
	return uint32(rand.Uint32())
}

func i2b(i int32) []byte {
	bbuf := bytes.NewBuffer([]byte{})
	binary.Write(bbuf, binary.BigEndian, i)
	return Sha256(bbuf.Bytes())
}

func b2i(bz []byte) int {
	var x int
	bbuf := bytes.NewBuffer(bz)
	binary.Read(bbuf, binary.BigEndian, &x)
	return x
}

// 测试set和get功能
func TestBasic(t *testing.T) {
	var tree = NewTree(nil, true)
	up := tree.Set([]byte("1"), []byte("one"))
	if up {
		t.Error("Did not expect an update (should have been create)")
	}
	up = tree.Set([]byte("2"), []byte("two"))
	if up {
		t.Error("Did not expect an update (should have been create)")
	}
	up = tree.Set([]byte("2"), []byte("TWO"))
	if !up {
		t.Error("Expected an update")
	}
	up = tree.Set([]byte("5"), []byte("five"))
	if up {
		t.Error("Did not expect an update (should have been create)")
	}
	hash := tree.Hash()

	t.Log("TestBasic", "roothash", hash)

	//PrintMAVLNode(tree.root)

	// Test 0x00
	{
		idx, val, exists := tree.Get([]byte{0x00})
		if exists {
			t.Errorf("Expected no value to exist")
		}
		if idx != 0 {
			t.Errorf("Unexpected idx %x", idx)
		}
		if string(val) != "" {
			t.Errorf("Unexpected value %v", string(val))
		}
	}

	// Test "1"
	{
		idx, val, exists := tree.Get([]byte("1"))
		if !exists {
			t.Errorf("Expected value to exist")
		}
		if idx != 0 {
			t.Errorf("Unexpected idx %x", idx)
		}
		if string(val) != "one" {
			t.Errorf("Unexpected value %v", string(val))
		}
	}

	// Test "2"
	{
		idx, val, exists := tree.Get([]byte("2"))
		if !exists {
			t.Errorf("Expected value to exist")
		}
		if idx != 1 {
			t.Errorf("Unexpected idx %x", idx)
		}
		if string(val) != "TWO" {
			t.Errorf("Unexpected value %v", string(val))
		}
	}

	// Test "4"
	{
		idx, val, exists := tree.Get([]byte("4"))
		if exists {
			t.Errorf("Expected no value to exist")
		}
		if idx != 2 {
			t.Errorf("Unexpected idx %x", idx)
		}
		if string(val) != "" {
			t.Errorf("Unexpected value %v", string(val))
		}
	}
}
func TestTreeHeightAndSize(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)

	db := db.NewDB("mavltree", "leveldb", dir, 100)

	// Create some random key value pairs
	records := make(map[string]string)

	count := 14
	for i := 0; i < count; i++ {
		records[randstr(20)] = randstr(20)
	}

	// Construct some tree and save it
	t1 := NewTree(db, true)

	for key, value := range records {
		t1.Set([]byte(key), []byte(value))
	}

	for key, value := range records {
		index, t2value, _ := t1.Get([]byte(key))
		if string(t2value) != value {
			t.Log("TestTreeHeightAndSize", "index", index, "key", []byte(key))
		}
	}
	t1.Hash()
	//PrintMAVLNode(t1.root)
	t1.Save()
	if int32(count) != t1.Size() {
		t.Error("TestTreeHeightAndSize Size != count", "treesize", t1.Size(), "count", count)
	}
	//t.Log("TestTreeHeightAndSize", "treeheight", t1.Height(), "leafcount", count)
	//t.Log("TestTreeHeightAndSize", "treesize", t1.Size())
	db.Close()
}

//获取共享的老数据
func TestSetAndGetOld(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)
	dbm := db.NewDB("mavltree", "leveldb", dir, 100)
	t1 := NewTree(dbm, true)
	t1.Set([]byte("1"), []byte("1"))
	t1.Set([]byte("2"), []byte("2"))

	hash := t1.Hash()
	t1.Save()

	//t2 在t1的基础上再做修改
	t2 := NewTree(dbm, true)
	t2.Load(hash)
	t2.Set([]byte("2"), []byte("22"))
	hash = t2.Hash()
	t2.Save()

	t3 := NewTree(dbm, true)
	t3.Load(hash)
	_, v, _ := t3.Get([]byte("1"))
	assert.Equal(t, []byte("1"), v)

	_, v, _ = t3.Get([]byte("2"))
	assert.Equal(t, []byte("22"), v)
}

//开启mvcc 和 prefix 要保证 hash 不变
func TestHashSame(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)
	dbm := db.NewDB("mavltree", "leveldb", dir, 100)

	t1 := NewTree(dbm, true)
	t1.Set([]byte("1"), []byte("1"))
	t1.Set([]byte("2"), []byte("2"))

	hash1 := t1.Hash()

	EnableMVCC(true)
	defer EnableMVCC(false)
	EnableMavlPrefix(true)
	defer EnableMavlPrefix(false)
	//t2 在t1的基础上再做修改
	t2 := NewTree(dbm, true)
	t2.Set([]byte("1"), []byte("1"))
	t2.Set([]byte("2"), []byte("2"))
	hash2 := t2.Hash()
	assert.Equal(t, hash1, hash2)
}

func TestHashSame2(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)

	db1 := db.NewDB("test1", "leveldb", dir, 100)
	prevHash := make([]byte, 32)
	strs1 := make(map[string]bool)
	for i := 0; i < 5; i++ {
		prevHash, err = saveBlock(db1, int64(i), prevHash, 1000, false)
		assert.Nil(t, err)
		str := Bytes2Hex(prevHash)
		fmt.Println("unable", str)
		strs1[str] = true
	}

	db2 := db.NewDB("test2", "leveldb", dir, 100)
	prevHash = prevHash[:0]
	EnableMVCC(true)
	defer EnableMVCC(false)
	EnableMavlPrefix(true)
	defer EnableMavlPrefix(false)
	for i := 0; i < 5; i++ {
		prevHash, err = saveBlock(db2, int64(i), prevHash, 1000, false)
		assert.Nil(t, err)
		str := Bytes2Hex(prevHash)
		fmt.Println("enable", str)
		if ok := strs1[str]; !ok {
			t.Error("enable Prefix have a different hash")
		}
	}

}

//测试hash，save,load以及节点value值的更新功能
func TestPersistence(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)

	dbm := db.NewDB("mavltree", "leveldb", dir, 100)

	records := make(map[string]string)

	recordbaks := make(map[string]string)

	for i := 0; i < 10; i++ {
		records[randstr(20)] = randstr(20)
	}

	mvccdb := db.NewMVCC(dbm)

	t1 := NewTree(dbm, true)

	for key, value := range records {
		//t1.Set([]byte(key), []byte(value))
		kindsSet(t1, mvccdb, []byte(key), []byte(value), 0, enableMvcc)
		//t.Log("TestPersistence tree1 set", "key", key, "value", value)
		recordbaks[key] = randstr(20)
	}

	hash := t1.Hash()
	t1.Save()

	t.Log("TestPersistence", "roothash1", hash)

	// Load a tree
	t2 := NewTree(dbm, true)
	t2.Load(hash)

	for key, value := range records {
		//_, t2value, _ := t2.Get([]byte(key))
		_, t2value, _ := kindsGet(t2, mvccdb, []byte(key), 0, enableMvcc)
		if string(t2value) != value {
			t.Fatalf("Invalid value. Expected %v, got %v", value, t2value)
		}
	}

	// update 5个key的value在hash2 tree中，验证这个5个key在hash和hash2中的值不一样
	var count int
	for key, value := range recordbaks {
		count++
		if count > 5 {
			break
		}
		//t2.Set([]byte(key), []byte(value))
		kindsSet(t2, mvccdb, []byte(key), []byte(value), 1, enableMvcc)
		//t.Log("TestPersistence insert new node treee2", "key", string(key), "value", string(value))

	}

	hash2 := t2.Hash()
	t2.Save()
	t.Log("TestPersistence", "roothash2", hash2)

	// 重新加载hash

	t11 := NewTree(dbm, true)
	t11.Load(hash)

	t.Log("------tree11------TestPersistence---------")
	for key, value := range records {
		//_, t2value, _ := t11.Get([]byte(key))
		_, t2value, _ := kindsGet(t11, mvccdb, []byte(key), 0, enableMvcc)
		if string(t2value) != value {
			t.Fatalf("tree11 Invalid value. Expected %v, got %v", value, t2value)
		}
	}
	//重新加载hash2
	t22 := NewTree(dbm, true)
	t22.Load(hash2)
	t.Log("------tree22------TestPersistence---------")

	//有5个key对应的value值有变化
	for key, value := range records {
		//_, t2value, _ := t22.Get([]byte(key))
		_, t2value, _ := kindsGet(t22, mvccdb, []byte(key), 0, enableMvcc)
		if string(t2value) != value {
			t.Log("tree22 value update.", "oldvalue", string(value), "newvalue", string(t2value), "key", string(key))
		}
	}
	count = 0
	for key, value := range recordbaks {
		count++
		if count > 5 {
			break
		}
		//_, t2value, _ := t22.Get([]byte(key))
		_, t2value, _ := kindsGet(t22, mvccdb, []byte(key), 1, enableMvcc)
		if string(t2value) != value {
			t.Logf("tree2222 Invalid value. Expected %v, got %v,key %v", string(value), string(t2value), string(key))
		}
	}
	dbm.Close()
}

func kindsGet(t *Tree, mvccdb *db.MVCCHelper, key []byte, version int64, enableMvcc bool) (index int32, value []byte, exists bool) {
	if enableMvcc {
		if mvccdb != nil {
			value, err := mvccdb.GetV(key, version)
			if err != nil {
				return 0, nil, false
			}
			return 0, value, true
		}
	} else {
		if t != nil {
			return t.Get(key)
		}
	}
	return 0, nil, false
}

func kindsSet(t *Tree, mvccdb *db.MVCCHelper, key []byte, value []byte, version int64, enableMvcc bool) (updated bool) {
	if enableMvcc {
		if mvccdb != nil {
			err := mvccdb.SetV(key, value, version)
			if err != nil {
				panic(fmt.Errorf("mvccdb cant setv %s", err.Error()))
			}
		}
	}
	return t.Set(key, value)
}

//测试key:value对的proof证明功能
func TestIAVLProof(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)

	db := db.NewDB("mavltree", "leveldb", dir, 100)

	var tree = NewTree(db, true)

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("TestIAVLProof key:%d!", i)
		value := fmt.Sprintf("TestIAVLProof value:%d!", i)
		tree.Set([]byte(key), []byte(value))
	}

	// Persist the items so far
	hash1 := tree.Save()

	// Add more items so it's not all persisted
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("TestIAVLProof KEY:%d!", i)
		value := fmt.Sprintf("TestIAVLProof VALUE:%d!", i)
		tree.Set([]byte(key), []byte(value))
	}

	rootHashBytes := tree.Hash()
	hashetr := ToHex(rootHashBytes)
	hashbyte, _ := FromHex(hashetr)

	t.Log("TestIAVLProof", "rootHashBytes", rootHashBytes)
	t.Log("TestIAVLProof", "hashetr", hashetr)
	t.Log("TestIAVLProof", "hashbyte", hashbyte)

	var KEY9proofbyte []byte

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("TestIAVLProof KEY:%d!", i)
		value := fmt.Sprintf("TestIAVLProof VALUE:%d!", i)
		keyBytes := []byte(key)
		valueBytes := []byte(value)
		_, KEY9proofbyte, _ = tree.Proof(keyBytes)
		value2, proof := tree.ConstructProof(keyBytes)
		if !bytes.Equal(value2, valueBytes) {
			t.Log("TestIAVLProof", "value2", string(value2), "value", string(valueBytes))
		}
		if proof != nil {
			istrue := proof.Verify([]byte(key), []byte(value), rootHashBytes)
			if !istrue {
				t.Error("TestIAVLProof Verify fail", "keyBytes", string(keyBytes), "valueBytes", string(valueBytes), "roothash", rootHashBytes)
			}
		}
	}

	t.Log("TestIAVLProof test Persistence data----------------")
	tree = NewTree(db, true)

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("TestIAVLProof key:%d!", i)
		value := fmt.Sprintf("TestIAVLProof value:%d!", i)
		keyBytes := []byte(key)
		valueBytes := []byte(value)

		value2, proofbyte, _ := tree.Proof(keyBytes)
		if !bytes.Equal(value2, valueBytes) {
			t.Log("TestIAVLProof", "value2", string(value2), "value", string(valueBytes))
		}
		if proofbyte != nil {

			leafNode := types.LeafNode{Key: keyBytes, Value: valueBytes, Height: 0, Size: 1}
			leafHash := leafNode.Hash()

			proof, err := ReadProof(rootHashBytes, leafHash, proofbyte)
			if err != nil {
				t.Log("TestIAVLProof ReadProof err ", "err", err)
			}
			istrue := proof.Verify([]byte(key), []byte(value), rootHashBytes)
			if !istrue {
				t.Log("TestIAVLProof Verify fail!", "keyBytes", string(keyBytes), "valueBytes", string(valueBytes), "roothash", rootHashBytes)
			}
		}
	}

	roothash := tree.Save()

	//key：value对的proof，hash1中不存在,roothash中存在
	index := 9
	key := fmt.Sprintf("TestIAVLProof KEY:%d!", index)
	value := fmt.Sprintf("TestIAVLProof VALUE:%d!", index)
	keyBytes := []byte(key)
	valueBytes := []byte(value)

	leafNode := types.LeafNode{Key: keyBytes, Value: valueBytes, Height: 0, Size: 1}
	leafHash := leafNode.Hash()

	// verify proof in tree1
	t.Log("TestIAVLProof  Verify key proof in tree1 ", "keyBytes", string(keyBytes), "valueBytes", string(valueBytes), "roothash", hash1)

	proof, err := ReadProof(hash1, leafHash, KEY9proofbyte)
	if err != nil {
		t.Log("TestIAVLProof ReadProof err ", "err", err)
	}
	istrue := proof.Verify(keyBytes, valueBytes, hash1)
	if !istrue {
		t.Log("TestIAVLProof  key not in tree ", "keyBytes", string(keyBytes), "valueBytes", string(valueBytes), "roothash", hash1)
	}

	// verify proof in tree2
	t.Log("TestIAVLProof  Verify key proof in tree2 ", "keyBytes", string(keyBytes), "valueBytes", string(valueBytes), "roothash", roothash)

	proof, err = ReadProof(roothash, leafHash, KEY9proofbyte)
	if err != nil {
		t.Log("TestIAVLProof ReadProof err ", "err", err)
	}
	istrue = proof.Verify(keyBytes, valueBytes, roothash)
	if istrue {
		t.Log("TestIAVLProof  key in tree2 ", "keyBytes", string(keyBytes), "valueBytes", string(valueBytes), "roothash", roothash)
	}
	db.Close()
}

func TestSetAndGetKVPair(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)

	db := db.NewDB("mavltree", "leveldb", dir, 100)

	var storeSet types.StoreSet
	var storeGet types.StoreGet
	var storeDel types.StoreGet

	total := 10
	storeSet.KV = make([]*types.KeyValue, total)
	storeGet.Keys = make([][]byte, total)
	storeDel.Keys = make([][]byte, total-5)

	records := make(map[string]string)

	for i := 0; i < total; i++ {
		records[randstr(20)] = randstr(20)
	}
	i := 0
	for key, value := range records {
		var keyvalue types.KeyValue
		keyvalue.Key = []byte(key)
		keyvalue.Value = []byte(value)
		if i < total {
			storeSet.KV[i] = &keyvalue
			storeGet.Keys[i] = []byte(key)
			if i < total-5 {
				storeDel.Keys[i] = []byte(key)
			}
		}
		i++
	}
	// storeSet hash is nil
	storeSet.StateHash = emptyRoot[:]
	newhash, err := SetKVPair(db, &storeSet, true)
	assert.Nil(t, err)
	//打印指定roothash的tree
	t.Log("TestSetAndGetKVPair newhash tree")
	PrintTreeLeaf(db, newhash)

	//删除5个节点
	storeDel.StateHash = newhash
	delhash, _, err := DelKVPair(db, &storeDel)
	assert.Nil(t, err)
	//打印指定roothash的tree
	t.Log("TestSetAndGetKVPair delhash tree")
	PrintTreeLeaf(db, delhash)

	// 在原来的基础上再次插入10个节点

	var storeSet2 types.StoreSet
	var storeGet2 types.StoreGet

	total = 10
	storeSet2.KV = make([]*types.KeyValue, total)
	storeGet2.Keys = make([][]byte, total)

	records2 := make(map[string]string)

	for j := 0; j < total; j++ {
		records2[randstr(20)] = randstr(20)
	}
	i = 0
	for key, value := range records2 {
		var keyvalue types.KeyValue
		keyvalue.Key = []byte(key)
		keyvalue.Value = []byte(value)
		if i < total {
			storeSet2.KV[i] = &keyvalue
			storeGet2.Keys[i] = []byte(key)
		}
		i++
	}
	// storeSet hash is newhash
	storeSet2.StateHash = delhash
	newhash2, err := SetKVPair(db, &storeSet2, true)
	assert.Nil(t, err)
	t.Log("TestSetAndGetKVPair newhash2 tree")
	PrintTreeLeaf(db, newhash2)

	t.Log("TestSetAndGetKVPair delhash tree again !!!")
	PrintTreeLeaf(db, delhash)

	t.Log("TestSetAndGetKVPair newhash tree again !!!")
	PrintTreeLeaf(db, newhash)
	db.Close()
}

func TestGetAndVerifyKVPairProof(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)

	db := db.NewDB("mavltree", "leveldb", dir, 100)

	var storeSet types.StoreSet
	var storeGet types.StoreGet

	total := 10
	storeSet.KV = make([]*types.KeyValue, total)
	storeGet.Keys = make([][]byte, total)

	records := make(map[string]string)

	for i := 0; i < total; i++ {
		records[randstr(20)] = randstr(20)
	}
	i := 0
	for key, value := range records {
		var keyvalue types.KeyValue
		keyvalue.Key = []byte(key)
		keyvalue.Value = []byte(value)
		if i < total {
			storeSet.KV[i] = &keyvalue
			storeGet.Keys[i] = []byte(key)
		}
		i++
	}
	// storeSet hash is nil
	storeSet.StateHash = emptyRoot[:]
	newhash, err := SetKVPair(db, &storeSet, true)
	assert.Nil(t, err)
	for i = 0; i < total; i++ {
		var keyvalue types.KeyValue

		proof, err := GetKVPairProof(db, newhash, storeGet.Keys[i])
		assert.Nil(t, err)
		keyvalue.Key = storeGet.Keys[i]
		keyvalue.Value = []byte(records[string(storeGet.Keys[i])])
		exit := VerifyKVPairProof(db, newhash, keyvalue, proof)
		if !exit {
			t.Log("TestGetAndVerifyKVPairProof  Verify proof fail!", "keyvalue", keyvalue.String(), "newhash", newhash)
		}
	}
	db.Close()
}

type traverser struct {
	Values []string
}

func (t *traverser) view(key, value []byte) bool {
	t.Values = append(t.Values, string(value))
	return false
}

// 迭代测试
func TestIterateRange(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)

	db := db.NewDB("mavltree", "leveldb", dir, 100)
	tree := NewTree(db, true)

	type record struct {
		key   string
		value string
	}

	records := []record{
		{"abc", "abc"},
		{"low", "low"},
		{"fan", "fan"},
		{"foo", "foo"},
		{"foobaz", "foobaz"},
		{"good", "good"},
		{"foobang", "foobang"},
		{"foobar", "foobar"},
		{"food", "food"},
		{"foml", "foml"},
	}
	keys := make([]string, len(records))
	for i, r := range records {
		keys[i] = r.key
	}
	sort.Strings(keys)

	for _, r := range records {
		updated := tree.Set([]byte(r.key), []byte(r.value))
		if updated {
			t.Error("should have not been updated")
		}
	}

	// test traversing the whole node works... in order
	list := []string{}
	tree.Iterate(func(key []byte, value []byte) bool {
		list = append(list, string(value))
		return false
	})
	require.Equal(t, list, []string{"abc", "fan", "foml", "foo", "foobang", "foobar", "foobaz", "food", "good", "low"})

	trav := traverser{}
	tree.IterateRange([]byte("foo"), []byte("goo"), true, trav.view)
	require.Equal(t, trav.Values, []string{"foo", "foobang", "foobar", "foobaz", "food"})

	trav = traverser{}
	tree.IterateRangeInclusive([]byte("foo"), []byte("good"), true, trav.view)
	require.Equal(t, trav.Values, []string{"foo", "foobang", "foobar", "foobaz", "food", "good"})

	trav = traverser{}
	tree.IterateRange(nil, []byte("flap"), true, trav.view)
	require.Equal(t, trav.Values, []string{"abc", "fan"})

	trav = traverser{}
	tree.IterateRange([]byte("foob"), nil, true, trav.view)
	require.Equal(t, trav.Values, []string{"foobang", "foobar", "foobaz", "food", "good", "low"})

	trav = traverser{}
	tree.IterateRange([]byte("very"), nil, true, trav.view)
	require.Equal(t, trav.Values, []string(nil))

	trav = traverser{}
	tree.IterateRange([]byte("fooba"), []byte("food"), true, trav.view)
	require.Equal(t, trav.Values, []string{"foobang", "foobar", "foobaz"})

	trav = traverser{}
	tree.IterateRange([]byte("fooba"), []byte("food"), false, trav.view)
	require.Equal(t, trav.Values, []string{"foobaz", "foobar", "foobang"})

	trav = traverser{}
	tree.IterateRange([]byte("g"), nil, false, trav.view)
	require.Equal(t, trav.Values, []string{"low", "good"})
}

func TestIAVLPrint(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)
	dbm := db.NewDB("test", "leveldb", dir, 100)
	prevHash := make([]byte, 32)
	tree := NewTree(dbm, true)
	tree.Load(prevHash)
	PrintNode(tree.root)
	kvs := genKVShort(0, 10)
	for i, kv := range kvs {
		tree.Set(kv.Key, kv.Value)
		println("insert", i)
		PrintNode(tree.root)
		tree.Hash()
		//println("insert.hash", i)
		//PrintNode(tree.root)
	}
}

func TestPruningTree(t *testing.T) {
	const txN = 5    // 每个块交易量
	const preB = 500 // 一轮区块数
	const round = 5  // 更新叶子节点次数
	const preDel = preB / 10
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)
	db := db.NewDB("test", "leveldb", dir, 100)
	prevHash := make([]byte, 32)

	EnableMavlPrefix(true)
	defer EnableMavlPrefix(false)
	EnablePrune(true)
	defer EnablePrune(false)
	SetPruneHeight(preDel)
	defer SetPruneHeight(0)

	for j := 0; j < round; j++ {
		for i := 0; i < preB; i++ {
			setPruning(pruningStateStart)
			prevHash, err = saveUpdateBlock(db, int64(i), prevHash, txN, j, int64(j*preB+i))
			assert.Nil(t, err)
			m := int64(j*preB + i)
			if m/int64(preDel) > 1 && m%int64(preDel) == 0 {
				pruningTree(db, m)
			}
		}
		fmt.Printf("round %d over \n", j)
	}
	te := NewTree(db, true)
	err = te.Load(prevHash)
	if err == nil {
		fmt.Printf("te.Load \n")
		for i := 0; i < preB*txN; i++ {
			key := []byte(fmt.Sprintf("my_%018d", i))
			vIndex := round - 1
			value := []byte(fmt.Sprintf("my_%018d_%d", i, vIndex))
			_, v, exist := te.Get(key)
			assert.Equal(t, exist, true)
			assert.Equal(t, value, v)
		}
	}
	PruningTreePrintDB(db, []byte(leafKeyCountPrefix))
	PruningTreePrintDB(db, []byte(hashNodePrefix))
	PruningTreePrintDB(db, []byte(leafNodePrefix))
}

func genUpdateKV(height int64, txN int64, vIndex int) (kvs []*types.KeyValue) {
	for i := int64(0); i < txN; i++ {
		n := height*txN + i
		key := fmt.Sprintf("my_%018d", n)
		value := fmt.Sprintf("my_%018d_%d", n, vIndex)
		//fmt.Printf("index: %d  i: %d  %v \n", vIndex, i, value)
		kvs = append(kvs, &types.KeyValue{Key: []byte(key), Value: []byte(value)})
	}
	return kvs
}

func saveUpdateBlock(dbm db.DB, height int64, hash []byte, txN int64, vIndex int, blockHeight int64) (newHash []byte, err error) {
	t := NewTree(dbm, true)
	t.Load(hash)
	t.SetBlockHeight(blockHeight)
	kvs := genUpdateKV(height, txN, vIndex)
	for _, kv := range kvs {
		t.Set(kv.Key, kv.Value)
	}
	newHash = t.Save()
	return newHash, nil
}

func TestMaxLevalDbValue(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)

	db := db.NewDB("mavltree", "leveldb", dir, 100)
	value := make([]byte, 1024*1024*5)
	for i := 0; i < 5; i++ {
		db.Set([]byte(fmt.Sprintf("mmm%d", i)), value)
	}

	for i := 0; i < 5; i++ {
		_, e := db.Get([]byte(fmt.Sprintf("mmm%d", i)))
		assert.NoError(t, e, "")
		//fmt.Println(v)
	}
}

func TestPruningFirstLevelNode(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)
	defer os.RemoveAll(dir)

	db1 := db.NewDB("mavltree", "leveldb", dir, 100)
	//add node data
	nodes1 := []Node{
		{key: []byte("11111111"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44d"), height: 1},
		{key: []byte("11111111"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44a"), height: 5000},
		{key: []byte("11111111"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44b"), height: 10000},
		{key: []byte("11111111"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44c"), height: 30000},
		{key: []byte("11111111"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44e"), height: 40000},
		{key: []byte("11111111"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44f"), height: 450000},
	}

	nodes2 := []Node{
		{key: []byte("22222222"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc45d"), height: 1},
		{key: []byte("22222222"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc45a"), height: 5000},
		{key: []byte("22222222"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc45b"), height: 10000},
		{key: []byte("22222222"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc45c"), height: 30000},
		{key: []byte("22222222"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc45e"), height: 40000},
		{key: []byte("22222222"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc45f"), height: 450000},
	}

	hashNodes1 := []types.PruneData{
		{Hashs: [][]byte{[]byte("113"), []byte("114"), []byte("115"), []byte("116"), []byte("117"), []byte("118")}},
		{Hashs: [][]byte{[]byte("123"), []byte("124"), []byte("125"), []byte("126"), []byte("127"), []byte("128")}},
		{Hashs: [][]byte{[]byte("133"), []byte("134"), []byte("135"), []byte("136"), []byte("137"), []byte("138")}},
		{Hashs: [][]byte{[]byte("143"), []byte("144"), []byte("145"), []byte("146"), []byte("147"), []byte("148")}},
		{Hashs: [][]byte{[]byte("153"), []byte("154"), []byte("155"), []byte("156"), []byte("157"), []byte("158")}},
		{Hashs: [][]byte{[]byte("163"), []byte("164"), []byte("165"), []byte("166"), []byte("167"), []byte("168")}},
	}

	hashNodes2 := []types.PruneData{
		{Hashs: [][]byte{[]byte("213"), []byte("214"), []byte("215"), []byte("216"), []byte("217"), []byte("218")}},
		{Hashs: [][]byte{[]byte("223"), []byte("224"), []byte("225"), []byte("226"), []byte("227"), []byte("228")}},
		{Hashs: [][]byte{[]byte("233"), []byte("234"), []byte("235"), []byte("236"), []byte("237"), []byte("238")}},
		{Hashs: [][]byte{[]byte("243"), []byte("244"), []byte("245"), []byte("246"), []byte("247"), []byte("248")}},
		{Hashs: [][]byte{[]byte("253"), []byte("254"), []byte("255"), []byte("256"), []byte("257"), []byte("258")}},
		{Hashs: [][]byte{[]byte("263"), []byte("264"), []byte("265"), []byte("266"), []byte("267"), []byte("268")}},
	}

	batch := db1.NewBatch(true)
	for i, node := range nodes1 {
		k := genLeafCountKey(node.key, node.hash, int64(node.height), len(node.hash))
		data := &types.PruneData{
			Hashs: hashNodes1[i].Hashs,
		}
		v, err := proto.Marshal(data)
		if err != nil {
			panic(err)
		}
		// 保存索引节点
		batch.Set(k, v)
		// 保存叶子节点
		batch.Set(node.hash, node.key)
		// 保存hash节点
		for _, hash := range data.Hashs {
			batch.Set(hash, hash)
		}
	}
	for i, node := range nodes2 {
		k := genLeafCountKey(node.key, node.hash, int64(node.height), len(node.hash))
		data := &types.PruneData{
			Hashs: hashNodes2[i].Hashs,
		}
		v, err := proto.Marshal(data)
		if err != nil {
			panic(err)
		}
		// 保存索引节点
		batch.Set(k, v)
		// 保存叶子节点
		batch.Set(node.hash, node.key)
		// 保存hash节点
		for _, hash := range data.Hashs {
			batch.Set(hash, hash)
		}
	}
	batch.Write()
	db1.Close()

	SetPruneHeight(5000)
	db2 := db.NewDB("mavltree", "leveldb", dir, 100)

	var existHashs [][]byte
	var noExistHashs [][]byte

	//当前高度设置为10000,只能删除高度为1的节点
	pruningFirstLevel(db2, 10000)

	for i, node := range nodes1 {
		if i >= 1 {
			existHashs = append(existHashs, node.hash)
			existHashs = append(existHashs, hashNodes1[i].Hashs...)
		} else {
			noExistHashs = append(noExistHashs, node.hash)
			noExistHashs = append(noExistHashs, hashNodes1[i].Hashs...)
		}
	}
	for i, node := range nodes2 {
		if i >= 1 {
			existHashs = append(existHashs, node.hash)
			existHashs = append(existHashs, hashNodes1[i].Hashs...)
		} else {
			noExistHashs = append(noExistHashs, node.hash)
			noExistHashs = append(noExistHashs, hashNodes1[i].Hashs...)
		}
	}
	verifyNodeExist(t, db2, existHashs, noExistHashs)

	//当前高度设置为20000, 删除高度为1000的
	pruningFirstLevel(db2, 20000)

	existHashs = existHashs[:0][:0]
	existHashs = noExistHashs[:0][:0]
	for i, node := range nodes1 {
		if i >= 2 {
			existHashs = append(existHashs, node.hash)
			existHashs = append(existHashs, hashNodes1[i].Hashs...)
		} else {
			noExistHashs = append(noExistHashs, node.hash)
			noExistHashs = append(noExistHashs, hashNodes1[i].Hashs...)
		}
	}
	for i, node := range nodes2 {
		if i >= 2 {
			existHashs = append(existHashs, node.hash)
			existHashs = append(existHashs, hashNodes1[i].Hashs...)
		} else {
			noExistHashs = append(noExistHashs, node.hash)
			noExistHashs = append(noExistHashs, hashNodes1[i].Hashs...)
		}
	}
	verifyNodeExist(t, db2, existHashs, noExistHashs)

	//目前还剩下 10000 30000 40000 450000
	//当前高度设置为510001, 将高度为10000的加入二级节点,删除30000 40000节点
	pruningFirstLevel(db2, 510001)
	existHashs = existHashs[:0][:0]
	existHashs = noExistHashs[:0][:0]
	for i, node := range nodes1 {
		if i >= 5 {
			existHashs = append(existHashs, node.hash)
			existHashs = append(existHashs, hashNodes1[i].Hashs...)
		} else if i == 2 {

		} else {
			noExistHashs = append(noExistHashs, node.hash)
			noExistHashs = append(noExistHashs, hashNodes1[i].Hashs...)
		}
	}
	for i, node := range nodes2 {
		if i >= 5 {
			existHashs = append(existHashs, node.hash)
			existHashs = append(existHashs, hashNodes1[i].Hashs...)
		} else if i == 2 {

		} else {
			noExistHashs = append(noExistHashs, node.hash)
			noExistHashs = append(noExistHashs, hashNodes1[i].Hashs...)
		}
	}
	verifyNodeExist(t, db2, existHashs, noExistHashs)

	//检查转换成二级裁剪高度的节点
	var secLevelNodes []*Node
	secLevelNodes = append(secLevelNodes, &nodes1[2])
	secLevelNodes = append(secLevelNodes, &nodes2[2])
	VerifySecLevelCountNodeExist(t, db2, secLevelNodes)
}

func verifyNodeExist(t *testing.T, dbm db.DB, existHashs [][]byte, noExistHashs [][]byte) {
	for _, hash := range existHashs {
		_, err := dbm.Get(hash)
		if err != nil {
			require.NoError(t, fmt.Errorf("this node should exist %s", string(hash)))
		}
	}

	for _, hash := range noExistHashs {
		v, err := dbm.Get(hash)
		if err == nil || len(v) > 0 {
			require.NoError(t, fmt.Errorf("this node should not exist %s", string(hash)))
		}
	}
}

func VerifySecLevelCountNodeExist(t *testing.T, dbm db.DB, nodes []*Node) {
	for _, node := range nodes {
		_, err := dbm.Get(genOldLeafCountKey(node.key, node.hash, int64(node.height), len(node.hash)))
		if err != nil {
			require.NoError(t, fmt.Errorf("this node should exist key: %s, hash: %s", string(node.key), string(node.hash)))
		}
	}
}

func TestPruningSecondLevelNode(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)
	defer os.RemoveAll(dir)

	db1 := db.NewDB("mavltree", "leveldb", dir, 100)
	//add node data
	nodes1 := []Node{
		{key: []byte("11111111"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44d"), height: 1},
		{key: []byte("11111111"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44a"), height: 5000},
		{key: []byte("11111111"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44b"), height: 10000},
	}

	nodes2 := []Node{
		{key: []byte("22222222"), hash: []byte("d95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc45d"), height: 1},
	}

	hashNodes1 := []types.PruneData{
		{Hashs: [][]byte{[]byte("113"), []byte("114"), []byte("115"), []byte("116"), []byte("117"), []byte("118")}},
		{Hashs: [][]byte{[]byte("123"), []byte("124"), []byte("125"), []byte("126"), []byte("127"), []byte("128")}},
		{Hashs: [][]byte{[]byte("133"), []byte("134"), []byte("135"), []byte("136"), []byte("137"), []byte("138")}},
	}

	hashNodes2 := []types.PruneData{
		{Hashs: [][]byte{[]byte("213"), []byte("214"), []byte("215"), []byte("216"), []byte("217"), []byte("218")}},
	}

	batch := db1.NewBatch(true)
	for i, node := range nodes1 {
		k := genOldLeafCountKey(node.key, node.hash, int64(node.height), len(node.hash))
		data := &types.PruneData{
			Hashs: hashNodes1[i].Hashs,
		}
		v, err := proto.Marshal(data)
		if err != nil {
			panic(err)
		}
		// 保存索引节点
		batch.Set(k, v)
		// 保存叶子节点
		batch.Set(node.hash, node.key)
		// 保存hash节点
		for _, hash := range data.Hashs {
			batch.Set(hash, hash)
		}
	}
	for i, node := range nodes2 {
		k := genOldLeafCountKey(node.key, node.hash, int64(node.height), len(node.hash))
		data := &types.PruneData{
			Hashs: hashNodes2[i].Hashs,
		}
		v, err := proto.Marshal(data)
		if err != nil {
			panic(err)
		}
		// 保存索引节点
		batch.Set(k, v)
		// 保存叶子节点
		batch.Set(node.hash, node.key)
		// 保存hash节点
		for _, hash := range data.Hashs {
			batch.Set(hash, hash)
		}
	}
	batch.Write()
	db1.Close()

	SetPruneHeight(5000)
	db2 := db.NewDB("mavltree", "leveldb", dir, 100)

	var existHashs [][]byte
	var noExistHashs [][]byte

	//当前高度设置为1500010,只能删除高度为1的节点
	pruningSecondLevel(db2, 1500010)

	for i, node := range nodes1 {
		if i >= 2 {
			existHashs = append(existHashs, node.hash)
			existHashs = append(existHashs, hashNodes1[i].Hashs...)
		} else {
			noExistHashs = append(noExistHashs, node.hash)
			noExistHashs = append(noExistHashs, hashNodes1[i].Hashs...)
		}
	}
	verifyNodeExist(t, db2, existHashs, noExistHashs)

	//检查转换成二级裁剪高度的节点
	var secLevelNodes []*Node
	secLevelNodes = append(secLevelNodes, &nodes2[0])
	VerifyThreeLevelCountNodeExist(t, db2, secLevelNodes)
}

func VerifyThreeLevelCountNodeExist(t *testing.T, dbm db.DB, nodes []*Node) {
	for _, node := range nodes {
		v, err := dbm.Get(genOldLeafCountKey(node.key, node.hash, int64(node.height), len(node.hash)))
		if err == nil || len(v) > 0 {
			require.NoError(t, fmt.Errorf("this node should not exist key:%s hash:%s", string(node.key), string(node.hash)))
		}
	}
}

func TestGetHashNode(t *testing.T) {
	eHashs := [][]byte{
		[]byte("h44"),
		[]byte("h33"),
		[]byte("h22"),
		[]byte("h11"),
		[]byte("h00"),
	}

	root := &Node{
		key:        []byte("00"),
		hash:       []byte("h00"),
		parentNode: nil,
	}

	node1 := &Node{
		key:        []byte("11"),
		hash:       []byte("h11"),
		parentNode: root,
	}

	node2 := &Node{
		key:        []byte("22"),
		hash:       []byte("h22"),
		parentNode: node1,
	}

	node3 := &Node{
		key:        []byte("33"),
		hash:       []byte("h33"),
		parentNode: node2,
	}

	node4 := &Node{
		key:        []byte("44"),
		hash:       []byte("h44"),
		parentNode: node3,
	}

	leafN := &Node{
		key:        []byte("55"),
		hash:       []byte("h55"),
		parentNode: node4,
	}

	hashs := getHashNode(leafN)
	require.Equal(t, len(eHashs), len(hashs))
	for _, hash := range hashs {
		t.Log("hash is ", string(hash))
		require.Contains(t, eHashs, hash)
	}
}

func TestGetKeyHeightFromLeafCountKey(t *testing.T) {
	key := []byte("123456")
	hash, err := FromHex("0x5f6d625f2d303030303030303030302dd95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44d")
	require.NoError(t, err)
	height := 100001
	hashLen := len(hash)
	hashkey := genLeafCountKey(key, hash, int64(height), hashLen)

	//1
	key1, height1, hash1, err := getKeyHeightFromLeafCountKey(hashkey)
	require.NoError(t, err)
	require.Equal(t, key, key1)
	require.Equal(t, height, height1)
	require.Equal(t, hash, hash1)

	//2
	key = []byte("24525252626988973653")
	hash, err = FromHex("0xd95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44d")
	require.NoError(t, err)
	height = 453
	hashLen = len(hash)
	hashkey = genLeafCountKey(key, hash, int64(height), hashLen)

	key1, height1, hash1, err = getKeyHeightFromLeafCountKey(hashkey)
	require.NoError(t, err)
	require.Equal(t, key, key1)
	require.Equal(t, height, height1)
	require.Equal(t, hash, hash1)
}

func TestGetKeyHeightFromOldLeafCountKey(t *testing.T) {
	key := []byte("123456")
	hash, err := FromHex("0x5f6d625f2d303030303030303030302dd95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44d")
	require.NoError(t, err)
	height := 100001
	hashLen := len(hash)
	hashkey := genOldLeafCountKey(key, hash, int64(height), hashLen)

	//1
	key1, height1, hash1, err := getKeyHeightFromOldLeafCountKey(hashkey)
	require.NoError(t, err)
	require.Equal(t, key, key1)
	require.Equal(t, height, height1)
	require.Equal(t, hash, hash1)

	//2
	key = []byte("24525252626988973653")
	hash, err = FromHex("0xd95f1027b1ecf9013a1cf870a85d967ca828e8faca366a290ec43adcecfbc44d")
	require.NoError(t, err)
	height = 453
	hashLen = len(hash)
	hashkey = genOldLeafCountKey(key, hash, int64(height), hashLen)

	key1, height1, hash1, err = getKeyHeightFromOldLeafCountKey(hashkey)
	require.NoError(t, err)
	require.Equal(t, key, key1)
	require.Equal(t, height, height1)
	require.Equal(t, hash, hash1)
}

func BenchmarkDBSet(b *testing.B) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(b, err)
	b.Log(dir)
	db := db.NewDB("test", "leveldb", dir, 100)
	prevHash := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		prevHash, err = saveBlock(db, int64(i), prevHash, 1000, false)
		assert.Nil(b, err)
	}
}

func BenchmarkDBSetMVCC(b *testing.B) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(b, err)
	b.Log(dir)
	db := db.NewDB("test", "leveldb", dir, 100)
	prevHash := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		prevHash, err = saveBlock(db, int64(i), prevHash, 1000, true)
		assert.Nil(b, err)
	}
}

func BenchmarkDBGet(b *testing.B) {
	//开启MVCC情况下不做测试；BenchmarkDBGetMVCC进行测试
	if enableMvcc {
		b.Skip()
		return
	}
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(b, err)
	b.Log(dir)
	db := db.NewDB("test", "leveldb", dir, 100)
	prevHash := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		prevHash, err = saveBlock(db, int64(i), prevHash, 1000, false)
		assert.Nil(b, err)
		if i%10 == 0 {
			fmt.Println(prevHash)
		}
	}
	b.ResetTimer()
	t := NewTree(db, true)
	t.Load(prevHash)
	for i := 0; i < b.N*1000; i++ {
		key := i2b(int32(i))
		value := Sha256(key)
		_, v, exist := t.Get(key)
		assert.Equal(b, exist, true)
		assert.Equal(b, value, v)
	}
}

func BenchmarkDBGetMVCC(b *testing.B) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	b.Log(dir)
	ldb := db.NewDB("test", "leveldb", dir, 100)
	prevHash := make([]byte, 32)
	EnableMavlPrefix(true)
	defer EnableMavlPrefix(false)
	for i := 0; i < b.N; i++ {
		prevHash, err = saveBlock(ldb, int64(i), prevHash, 1000, true)
		assert.Nil(b, err)
		if i%10 == 0 {
			fmt.Println(prevHash)
		}
	}
	b.ResetTimer()
	mvccdb := db.NewMVCC(ldb)
	for i := 0; i < b.N*1000; i++ {
		key := i2b(int32(i))
		value := Sha256(key)
		v, err := mvccdb.GetV(key, int64(b.N-1))
		assert.Nil(b, err)
		assert.Equal(b, value, v)
	}
}

func genKVShort(height int64, txN int64) (kvs []*types.KeyValue) {
	for i := int64(0); i < txN; i++ {
		n := height*1000 + i
		key := []byte(fmt.Sprintf("k:%d", n))
		value := []byte(fmt.Sprintf("v:%d", n))
		kvs = append(kvs, &types.KeyValue{Key: key, Value: value})
	}
	return kvs
}

func genKV(height int64, txN int64) (kvs []*types.KeyValue) {
	for i := int64(0); i < txN; i++ {
		n := height*txN + i
		key := i2b(int32(n))
		value := Sha256(key)
		kvs = append(kvs, &types.KeyValue{Key: key, Value: value})
	}
	return kvs
}

func saveBlock(dbm db.DB, height int64, hash []byte, txN int64, mvcc bool) (newHash []byte, err error) {
	t := NewTree(dbm, true)
	t.Load(hash)
	kvs := genKV(height, txN)
	for _, kv := range kvs {
		t.Set(kv.Key, kv.Value)
	}
	newHash = t.Save()
	if mvcc {
		mvccdb := db.NewMVCC(dbm)
		newkvs, err := mvccdb.AddMVCC(kvs, newHash, hash, height)
		if err != nil {
			return nil, err
		}
		batch := dbm.NewBatch(true)
		for _, kv := range newkvs {
			batch.Set(kv.Key, kv.Value)
		}
		err = batch.Write()
		if err != nil {
			return nil, err
		}
	}
	return newHash, nil
}
