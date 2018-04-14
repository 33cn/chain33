package mavl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	. "gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

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
	return bbuf.Bytes()
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
	db := db.NewDB("mavltree", "leveldb", "datastore", 100)

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

//测试hash，save,load以及节点value值的更新功能
func TestPersistence(t *testing.T) {
	db := db.NewDB("mavltree", "leveldb", "datastore", 100)

	records := make(map[string]string)

	recordbaks := make(map[string]string)

	for i := 0; i < 10; i++ {
		records[randstr(20)] = randstr(20)
	}

	t1 := NewTree(db, true)

	for key, value := range records {
		t1.Set([]byte(key), []byte(value))
		//t.Log("TestPersistence tree1 set", "key", key, "value", value)
		recordbaks[key] = randstr(20)
	}

	hash := t1.Hash()
	t1.Save()

	t.Log("TestPersistence", "roothash1", hash)

	// Load a tree
	t2 := NewTree(db, true)
	t2.Load(hash)

	for key, value := range records {
		_, t2value, _ := t2.Get([]byte(key))
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
		t2.Set([]byte(key), []byte(value))
		//t.Log("TestPersistence insert new node treee2", "key", string(key), "value", string(value))
	}

	hash2 := t2.Hash()
	t2.Save()
	t.Log("TestPersistence", "roothash2", hash2)

	// 重新加载hash

	t11 := NewTree(db, true)
	t11.Load(hash)

	t.Log("------tree11------TestPersistence---------")
	for key, value := range records {
		_, t2value, _ := t11.Get([]byte(key))
		if string(t2value) != value {
			t.Fatalf("tree11 Invalid value. Expected %v, got %v", value, t2value)
		}
	}
	//重新加载hash2
	t22 := NewTree(db, true)
	t22.Load(hash2)
	t.Log("------tree22------TestPersistence---------")

	//有5个key对应的value值有变化
	for key, value := range records {
		_, t2value, _ := t22.Get([]byte(key))
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
		_, t2value, _ := t22.Get([]byte(key))
		if string(t2value) != value {
			t.Logf("tree2222 Invalid value. Expected %v, got %v,key %v", string(value), string(t2value), string(key))
		}
	}
	db.Close()
}

//测试key:value对的proof证明功能
func TestIAVLProof(t *testing.T) {

	db := db.NewDB("mavltree", "leveldb", "datastore", 100)

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
	db := db.NewDB("mavltree", "leveldb", "datastore", 100)

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
	storeSet.StateHash = nil
	newhash := SetKVPair(db, &storeSet, true)

	//打印指定roothash的tree
	t.Log("TestSetAndGetKVPair newhash tree")
	PrintTreeLeaf(db, newhash)

	//删除5个节点
	storeDel.StateHash = newhash
	delhash, _ := DelKVPair(db, &storeDel)
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
	newhash2 := SetKVPair(db, &storeSet2, true)

	t.Log("TestSetAndGetKVPair newhash2 tree")
	PrintTreeLeaf(db, newhash2)

	t.Log("TestSetAndGetKVPair delhash tree again !!!")
	PrintTreeLeaf(db, delhash)

	t.Log("TestSetAndGetKVPair newhash tree again !!!")
	PrintTreeLeaf(db, newhash)
	db.Close()
}

func TestGetAndVerifyKVPairProof(t *testing.T) {
	db := db.NewDB("mavltree", "leveldb", "datastore", 100)

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
	storeSet.StateHash = nil
	newhash := SetKVPair(db, &storeSet, true)

	i = 0
	for i = 0; i < total; i++ {
		var keyvalue types.KeyValue

		proof := GetKVPairProof(db, newhash, storeGet.Keys[i])

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
	db := db.NewDB("mavltree", "leveldb", "datastore", 100)
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

func BenchmarkSetMerkleAvlTree(b *testing.B) {
	b.StopTimer()

	db := db.NewDB("test", "leveldb", "./", 100)
	t := NewTree(db, true)

	for i := 0; i < 10000; i++ {
		key := i2b(int32(RandInt32()))
		t.Set(key, nil)
		if i%1000 == 999 {
			t.Save()
		}
	}
	t.Save()

	fmt.Println("BenchmarkSetMerkleAvlTree, starting")

	runtime.GC()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ri := i2b(int32(RandInt32()))
		t.Set(ri, nil)
		if i%1000 == 999 {
			t.Save()
		}
	}
	t.Save()
	db.Close()
}

func BenchmarkGetMerkleAvlTree(b *testing.B) {
	b.StopTimer()

	db := db.NewDB("test", "leveldb", "./", 100)
	t := NewTree(db, true)
	var key []byte
	for i := 0; i < 10000; i++ {
		key = i2b(int32(i))
		t.Set(key, nil)
		if i%100 == 99 {
			t.Save()
		}
	}
	t.Save()
	fmt.Println("BenchmarkGetMerkleAvlTree, starting")

	runtime.GC()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _, exit := t.Get(i2b(int32(i % 10000)))
		if !exit {
			fmt.Println("BenchmarkGetMerkleAvlTree no exit!")
		}
	}
	db.Close()
}
