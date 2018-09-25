// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package mpt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	comTy "gitlab.33.cn/chain33/chain33/types"
)

var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big3   = big.NewInt(3)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(256)
	Big257 = big.NewInt(257)
)

func init() {
	rand.Seed(time.Now().UnixNano())
	spew.Config.Indent = "    "
	spew.Config.DisableMethods = false
}

// Used for testing
func newEmpty() *Trie {
	memdb, _ := dbm.NewGoMemDB("gomemdb", "", 128)
	trie, _ := New(common.Hash{}, NewDatabase(memdb))
	return trie
}

func TestEmptyTrie(t *testing.T) {
	var trie Trie
	res := trie.Hash()
	exp := emptyRoot
	if res != common.Hash(exp) {
		t.Errorf("expected %x got %x", exp, res)
	}
}

func TestNull(t *testing.T) {
	var trie Trie
	key := make([]byte, 32)
	value := []byte("test")
	trie.Update(key, value)
	if !bytes.Equal(trie.Get(key), value) {
		t.Fatal("wrong value")
	}
}

func TestMissingRoot(t *testing.T) {
	memdb, _ := dbm.NewGoMemDB("gomemdb", "", 128)
	trie, err := New(common.HexToHash("0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33"), NewDatabase(memdb))
	if trie != nil {
		t.Error("New returned non-nil trie for invalid root")
	}
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("New returned wrong error: %v", err)
	}
}

func TestMissingNodeDisk(t *testing.T)    { testMissingNode(t, false) }
func TestMissingNodeMemonly(t *testing.T) { testMissingNode(t, true) }

func testMissingNode(t *testing.T, memonly bool) {
	memdb, _ := dbm.NewGoMemDB("gomemdb", "", 128)
	triedb := NewDatabase(memdb)

	trie, _ := New(common.Hash{}, triedb)
	updateString(trie, "120000", "qwerqwerqwerqwerqwerqwerqwerqwerxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	updateString(trie, "123456", "asdfasdfasdfasdfasdfasdfasdfasdfxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	root, _ := trie.Commit(nil)
	if !memonly {
		triedb.Commit(root, true)
	}
	trie, _ = New(root, triedb)
	data, err := trie.TryGet([]byte("120000"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	assert.Equal(t, []byte("qwerqwerqwerqwerqwerqwerqwerqwerxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"), data)
	trie, _ = New(root, triedb)
	_, err = trie.TryGet([]byte("120099"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = New(root, triedb)
	_, err = trie.TryGet([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = New(root, triedb)
	err = trie.TryUpdate([]byte("120099"), []byte("zxcvzxcvzxcvzxcvzxcvzxcvzxcvzxcvxxxxxxxxxxxxxxxxxxxx"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = New(root, triedb)
	err = trie.TryDelete([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	hash := common.HexToHash("9e2800ae476b4acbbdf630ec058098017784a876de4ca272899a32120eb0c879")
	if memonly {
		delete(triedb.nodes, hash)
	} else {
		memdb.Delete(hash[:])
	}

	trie, _ = New(root, triedb)
	_, err = trie.TryGet([]byte("120000"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
	trie, _ = New(root, triedb)
	_, err = trie.TryGet([]byte("120099"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
	trie, _ = New(root, triedb)
	_, err = trie.TryGet([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = New(root, triedb)
	err = trie.TryUpdate([]byte("120099"), []byte("zxcv"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
	trie, _ = New(root, triedb)
	err = trie.TryDelete([]byte("123456"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
}

func TestInsert(t *testing.T) {
	trie := newEmpty()

	updateString(trie, "doe", "reindeer")
	updateString(trie, "dog", "puppy")
	updateString(trie, "dogglesworth", "cat")

	exp := common.HexToHash("3b2d78cb0319a48452b99d27825e8db84d6df5e1a9a3acf7c000b2c8a3154bed")
	root := trie.Hash()
	if root != exp {
		t.Errorf("exp %x got %x", exp, root)
	}

	trie = newEmpty()
	updateString(trie, "A", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	exp = common.HexToHash("2ca388c689f5fb595f5d9837cb319564da9d12468763e001596ee20d109f030d")
	root, err := trie.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	if root != exp {
		t.Errorf("exp %x got %x", exp, root)
	}
}

func TestGet(t *testing.T) {
	trie := newEmpty()
	updateString(trie, "doe", "reindeer")
	updateString(trie, "dog", "puppy")
	updateString(trie, "dogglesworth", "cat")

	for i := 0; i < 2; i++ {
		res := getString(trie, "dog")
		if !bytes.Equal(res, []byte("puppy")) {
			t.Errorf("expected puppy got %x", res)
		}

		unknown := getString(trie, "unknown")
		if unknown != nil {
			t.Errorf("expected nil got %x", unknown)
		}

		if i == 1 {
			return
		}
		trie.Commit(nil)
	}
}

func TestDelete(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		if val.v != "" {
			updateString(trie, val.k, val.v)
		} else {
			deleteString(trie, val.k)
		}
	}

	hash := trie.Hash()
	exp := common.HexToHash("29f72de68f3db599f9b170f17813438d40e83bfdacb4ba95d80dbd1d2af55304")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestEmptyValues(t *testing.T) {
	trie := newEmpty()

	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		updateString(trie, val.k, val.v)
	}

	hash := trie.Hash()
	exp := common.HexToHash("29f72de68f3db599f9b170f17813438d40e83bfdacb4ba95d80dbd1d2af55304")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
	_, err := trie.Commit(nil)
	assert.Nil(t, err)
}

func TestReplication(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		updateString(trie, val.k, val.v)
	}
	exp, err := trie.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}

	// create a new trie on top of the database and check that lookups work.
	trie2, err := New(exp, trie.db)
	if err != nil {
		t.Fatalf("can't recreate trie at %x: %v", exp, err)
	}
	for _, kv := range vals {
		if string(getString(trie2, kv.k)) != kv.v {
			t.Errorf("trie2 doesn't have %q => %q", kv.k, kv.v)
		}
	}
	hash, err := trie2.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	if hash != exp {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}

	// perform some insertions on the new trie.
	vals2 := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		// {"shaman", "horse"},
		// {"doge", "coin"},
		// {"ether", ""},
		// {"dog", "puppy"},
		// {"somethingveryoddindeedthis is", "myothernodedata"},
		// {"shaman", ""},
	}
	for _, val := range vals2 {
		updateString(trie2, val.k, val.v)
	}
	if hash := trie2.Hash(); hash != exp {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}
}

func TestLargeValue(t *testing.T) {
	trie := newEmpty()
	trie.Update([]byte("key1"), []byte{99, 99, 99, 99})
	trie.Update([]byte("key2"), bytes.Repeat([]byte{1}, 32))
	trie.Hash()
}

type countingDB struct {
	dbm.DB
	gets map[string]int
}

func (db *countingDB) Get(key []byte) ([]byte, error) {
	db.gets[string(key)]++
	return db.DB.Get(key)
}

// TestCacheUnload checks that decoded nodes are unloaded after a
// certain number of commit operations.
func TestCacheUnload(t *testing.T) {
	// Create test trie with two branches.
	trie := newEmpty()
	key1 := "---------------------------------"
	key2 := "---some other branch"
	updateString(trie, key1, "this is the branch of key1.")
	updateString(trie, key2, "this is the branch of key2.")

	root, _ := trie.Commit(nil)
	trie.db.Commit(root, true)

	// Commit the trie repeatedly and access key1.
	// The branch containing it is loaded from DB exactly two times:
	// in the 0th and 6th iteration.
	db := &countingDB{DB: trie.db.db, gets: make(map[string]int)}
	trie, _ = New(root, NewDatabase(db))
	trie.SetCacheLimit(5)
	for i := 0; i < 12; i++ {
		getString(trie, key1)
		trie.Commit(nil)
	}
	// Check that it got loaded two times.
	for dbkey, count := range db.gets {
		if count != 2 {
			t.Errorf("db key %x loaded %d times, want %d times", []byte(dbkey), count, 2)
		}
	}
}

// randTest performs random trie operations.
// Instances of this test are created by Generate.
type randTest []randTestStep

type randTestStep struct {
	op    int
	key   []byte // for opUpdate, opDelete, opGet
	value []byte // for opUpdate
	err   error  // for debugging
}

const (
	opUpdate = iota
	opDelete
	opGet
	opCommit
	opHash
	opReset
	opItercheckhash
	opCheckCacheInvariant
	opMax // boundary value, not an actual op
)

func (randTest) Generate(r *rand.Rand, size int) reflect.Value {
	var allKeys [][]byte
	genKey := func() []byte {
		if len(allKeys) < 2 || r.Intn(100) < 10 {
			// new key
			key := make([]byte, r.Intn(50))
			r.Read(key)
			allKeys = append(allKeys, key)
			return key
		}
		// use existing key
		return allKeys[r.Intn(len(allKeys))]
	}

	var steps randTest
	for i := 0; i < size; i++ {
		step := randTestStep{op: r.Intn(opMax)}
		switch step.op {
		case opUpdate:
			step.key = genKey()
			step.value = make([]byte, 8)
			binary.BigEndian.PutUint64(step.value, uint64(i))
		case opGet, opDelete:
			step.key = genKey()
		}
		steps = append(steps, step)
	}
	return reflect.ValueOf(steps)
}

func runRandTest(rt randTest) bool {

	memdb, _ := dbm.NewGoMemDB("gomemdb", "", 128)
	triedb := NewDatabase(memdb)

	tr, _ := New(common.Hash{}, triedb)
	values := make(map[string]string) // tracks content of the trie

	for i, step := range rt {
		switch step.op {
		case opUpdate:
			tr.Update(step.key, step.value)
			values[string(step.key)] = string(step.value)
		case opDelete:
			tr.Delete(step.key)
			delete(values, string(step.key))
		case opGet:
			v := tr.Get(step.key)
			want := values[string(step.key)]
			if string(v) != want {
				rt[i].err = fmt.Errorf("mismatch for key 0x%x, got 0x%x want 0x%x", step.key, v, want)
			}
		case opCommit:
			_, rt[i].err = tr.Commit(nil)
		case opHash:
			tr.Hash()
		case opReset:
			hash, err := tr.Commit(nil)
			if err != nil {
				rt[i].err = err
				return false
			}
			newtr, err := New(hash, triedb)
			if err != nil {
				rt[i].err = err
				return false
			}
			tr = newtr
		case opItercheckhash:
			checktr, _ := New(common.Hash{}, triedb)
			it := NewIterator(tr.NodeIterator(nil))
			for it.Next() {
				checktr.Update(it.Key, it.Value)
			}
			if tr.Hash() != checktr.Hash() {
				rt[i].err = fmt.Errorf("hash mismatch in opItercheckhash")
			}
		case opCheckCacheInvariant:
			rt[i].err = checkCacheInvariant(tr.root, nil, tr.cachegen, false, 0)
		}
		// Abort the test on error.
		if rt[i].err != nil {
			return false
		}
	}
	return true
}

func checkCacheInvariant(n, parent node, parentCachegen uint16, parentDirty bool, depth int) error {
	var children []node
	var flag nodeFlag
	switch n := n.(type) {
	case *shortNode:
		flag = n.flags
		children = []node{n.Val}
	case *fullNode:
		flag = n.flags
		children = n.Children[:]
	default:
		return nil
	}

	errorf := func(format string, args ...interface{}) error {
		msg := fmt.Sprintf(format, args...)
		msg += fmt.Sprintf("\nat depth %d node %s", depth, spew.Sdump(n))
		msg += fmt.Sprintf("parent: %s", spew.Sdump(parent))
		return errors.New(msg)
	}
	if flag.gen > parentCachegen {
		return errorf("cache invariant violation: %d > %d\n", flag.gen, parentCachegen)
	}
	if depth > 0 && !parentDirty && flag.dirty {
		return errorf("cache invariant violation: %d > %d\n", flag.gen, parentCachegen)
	}
	for _, child := range children {
		if err := checkCacheInvariant(child, n, flag.gen, flag.dirty, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func TestRandom(t *testing.T) {
	if err := quick.Check(runRandTest, nil); err != nil {
		if cerr, ok := err.(*quick.CheckError); ok {
			t.Fatalf("random test iteration %d failed: %s", cerr.Count, spew.Sdump(cerr.In))
		}
		t.Fatal(err)
	}
}

func BenchmarkGet(b *testing.B)      { benchGet(b, false) }
func BenchmarkGetDB(b *testing.B)    { benchGet(b, true) }
func BenchmarkUpdateBE(b *testing.B) { benchUpdate(b, binary.BigEndian) }
func BenchmarkUpdateLE(b *testing.B) { benchUpdate(b, binary.LittleEndian) }

const benchElemCount = 20000

func benchGet(b *testing.B, commit bool) {
	trie := new(Trie)
	if commit {
		_, tmpdb := tempDB()
		trie, _ = New(common.Hash{}, tmpdb)
	}
	k := make([]byte, 32)
	for i := 0; i < benchElemCount; i++ {
		binary.LittleEndian.PutUint64(k, uint64(i))
		trie.Update(k, k)
	}
	binary.LittleEndian.PutUint64(k, benchElemCount/2)
	if commit {
		trie.Commit(nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Get(k)
	}
	b.StopTimer()

	//if commit {
	//	ldb := trie.db.db.(*ethdb.LDBDatabase)
	//	ldb.Close()
	//	os.RemoveAll(ldb.Path())
	//}
}

func benchUpdate(b *testing.B, e binary.ByteOrder) *Trie {
	trie := newEmpty()
	k := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		e.PutUint64(k, uint64(i))
		trie.Update(k, k)
	}
	return trie
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randBytes2(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

// Benchmarks the trie hashing. Since the trie caches the result of any operation,
// we cannot use b.N as the number of hashing rouns, since all rounds apart from
// the first one will be NOOP. As such, we'll use b.N as the number of account to
// insert into the trie before measuring the hashing.
func BenchmarkHash(b *testing.B) {
	// Make the random benchmark deterministic
	random := rand.New(rand.NewSource(0))

	// Create a realistic account trie to hash
	addresses := make([][20]byte, b.N)
	for i := 0; i < len(addresses); i++ {
		for j := 0; j < len(addresses[i]); j++ {
			addresses[i][j] = byte(random.Intn(256))
		}
	}
	accounts := make([][]byte, len(addresses))
	for i := 0; i < len(accounts); i++ {
		accounts[i] = randBytes2(128)
	}
	// Insert the accounts into the trie and hash it
	trie := newEmpty()
	for i := 0; i < len(addresses); i++ {
		trie.Update(common.ShaKeccak256(addresses[i][:]), accounts[i])
	}
	b.ResetTimer()
	b.ReportAllocs()
	trie.Hash()
}

func tempDB() (string, *Database) {
	dir, err := ioutil.TempDir("", "trie-bench")
	if err != nil {
		panic(fmt.Sprintf("can't create temporary directory: %v", err))
	}
	//diskdb, err := ethdb.NewLDBDatabase(dir, 256, 0)
	//if err != nil {
	//	panic(fmt.Sprintf("can't create temporary database: %v", err))
	//}
	diskdb, _ := dbm.NewGoLevelDB("goleveldb", dir, 128)
	if err != nil {
		panic(fmt.Sprintf("can't create temporary database: %v", err))
	}
	return dir, NewDatabase(diskdb)
}

func getString(trie *Trie, k string) []byte {
	return trie.Get([]byte(k))
}

func updateString(trie *Trie, k, v string) {
	trie.Update([]byte(k), []byte(v))
}

func deleteString(trie *Trie, k string) {
	trie.Delete([]byte(k))
}

func set10000(t assert.TestingT, root common.Hash, db dbm.DB, n int) (common.Hash, map[string]string) {
	database := NewDatabase(db)
	trie, _ := New(root, database)
	keys := make(map[string]string)
	for i := 0; i < n; i++ {
		k := string(randBytes2(10)) + fmt.Sprint(i)
		v := string(randBytes2(10)) + fmt.Sprint(i)
		keys[k] = v
		updateString(trie, k, v)
	}
	root, err := trie.Commit(nil)
	assert.Nil(t, err)
	trie.db.Commit(root, true)
	return root, keys
}

func get10000(t assert.TestingT, root common.Hash, db dbm.DB, keys map[string]string) {
	database := NewDatabase(db)
	t1, _ := New(root, database)
	for k, v := range keys {
		value, err := t1.TryGet([]byte(k))
		assert.Nil(t, err)
		assert.Equal(t, string(value), v)
	}
}

func BenchmarkInsert10000(b *testing.B) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(b, err)
	b.Log(dir)
	db := dbm.NewDB("test", "leveldb", dir, 100)
	root := common.Hash{}
	count := 0
	for i := 0; i < b.N; i++ {
		root, _ = set10000(b, root, db, 10000)
		count += 10000
		if count >= b.N {
			break
		}
	}
}

func BenchmarkInsertGet10000(b *testing.B) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(b, err)
	b.Log(dir)
	db := dbm.NewDB("test", "leveldb", dir, 100)
	root := common.Hash{}
	var kv map[string]string
	var prevkv map[string]string
	for i := 0; i < b.N; i++ {
		root, kv = set10000(b, root, db, 10000)
		get10000(b, root, db, kv)
		if prevkv != nil {
			for k := range kv {
				if _, ok := prevkv[k]; ok {
					panic("repeat k")
				}
			}
			get10000(b, root, db, prevkv)
		}
		prevkv = kv
	}
}

func BenchmarkGet10000(b *testing.B) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(b, err)
	b.Log(dir)
	db := dbm.NewDB("test", "leveldb", dir, 100)
	root := common.Hash{}
	kvs := make(map[string]string)
	insertN := b.N/10000 + 1
	for i := 0; i < insertN; i++ {
		var kv map[string]string
		root, kv = set10000(b, root, db, 10000)
		for k, v := range kv {
			kvs[k] = v
		}
	}
	b.ResetTimer()
	database := NewDatabase(db)
	t1, _ := New(root, database)
	i := 0
	for k, v := range kvs {
		i++
		if i == b.N {
			break
		}
		value, err := t1.TryGet([]byte(k))
		assert.Nil(b, err)
		assert.Equal(b, v, string(value))
	}
}

func TestInsert10000(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)
	db := dbm.NewDB("test", "leveldb", dir, 100)
	root := common.Hash{}
	root, kv1 := set10000(t, root, db, 10000)
	get10000(t, root, db, kv1)

	root, kv2 := set10000(t, root, db, 10000)
	get10000(t, root, db, kv1)
	get10000(t, root, db, kv2)
}

func TestInsert1(t *testing.T) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(t, err)
	t.Log(dir)
	db := dbm.NewDB("test", "leveldb", dir, 100)
	root := common.Hash{}
	root, kv1 := set10000(t, root, db, 2)
	root, kv2 := set10000(t, root, db, 2)

	get10000(t, root, db, kv1)
	get10000(t, root, db, kv2)
}

func BenchmarkDBGet(b *testing.B) {
	dir, err := ioutil.TempDir("", "datastore")
	require.NoError(b, err)
	b.Log(dir)
	db := dbm.NewDB("test", "leveldb", dir, 100)
	prevHash := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		prevHash, err = saveBlock(db, int64(i), prevHash, 1000, false)
		assert.Nil(b, err)
	}
	b.ResetTimer()
	t, _ := NewEx(common.BytesToHash(prevHash), NewDatabase(db))
	for i := 0; i < b.N*1000; i++ {
		key := i2b(int32(i))
		value := common.Sha256(key)
		v, err := t.TryGet(key)
		assert.Nil(b, err)
		assert.Equal(b, value, v)
	}
}

func i2b(i int32) []byte {
	bbuf := bytes.NewBuffer([]byte{})
	binary.Write(bbuf, binary.BigEndian, i)
	return common.Sha256(bbuf.Bytes())
}

func genKV(height int64, txN int64) (kvs []*comTy.KeyValue) {
	for i := int64(0); i < txN; i++ {
		n := height*1000 + i
		key := i2b(int32(n))
		value := common.Sha256(key)
		kvs = append(kvs, &comTy.KeyValue{Key: key, Value: value})
	}
	return kvs
}

func saveBlock(dbm dbm.DB, height int64, hash []byte, txN int64, mvcc bool) (newHash []byte, err error) {
	t, err := NewEx(common.BytesToHash(hash), NewDatabase(dbm))
	if nil != err {
		return nil, nil
	}
	kvs := genKV(height, txN)
	for _, kv := range kvs {
		t.Update(kv.Key, kv.Value)
	}
	rehash, err := t.Commit(nil)
	if nil != err {
		return nil, nil
	}
	err = t.Commit2Db(rehash, false)
	if nil != err {
		return nil, nil
	}
	newHash = rehash[:]
	//if mvcc {
	//	mvccdb := db.NewMVCC(dbm)
	//	newkvs, err := mvccdb.AddMVCC(kvs, newHash, hash, height)
	//	if err != nil {
	//		return nil, err
	//	}
	//	batch := dbm.NewBatch(true)
	//	for _, kv := range newkvs {
	//		batch.Set(kv.Key, kv.Value)
	//	}
	//	err = batch.Write()
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	return newHash, nil
}
