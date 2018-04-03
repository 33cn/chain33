package db

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

// 迭代测试
func testDBIterator(t *testing.T, db DB) {
	t.Log("test Set")
	db.Set([]byte("aaaaaa/1"), []byte("aaaaaa/1"))
	db.Set([]byte("my_key/1"), []byte("my_key/1"))
	db.Set([]byte("my_key/2"), []byte("my_key/2"))
	db.Set([]byte("my_key/3"), []byte("my_key/3"))
	db.Set([]byte("my_key/4"), []byte("my_key/4"))
	db.Set([]byte("my"), []byte("my"))
	db.Set([]byte("my_"), []byte("my_"))
	db.Set([]byte("zzzzzz/1"), []byte("zzzzzz/1"))
	b, err := hex.DecodeString("ff")
	require.NoError(t, err)
	db.Set(b, []byte("0xff"))

	t.Log("test Get")
	v := db.Get([]byte("aaaaaa/1"))
	require.Equal(t, string(v), "aaaaaa/1")

	t.Log("test PrefixScan")
	it := NewListHelper(db)
	list := it.PrefixScan(nil)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("aaaaaa/1"), []byte("my"), []byte("my_"), []byte("my_key/1"), []byte("my_key/2"), []byte("my_key/3"), []byte("my_key/4"), []byte("zzzzzz/1"), []byte("0xff")})

	t.Log("test IteratorScanFromFirst")
	list = it.IteratorScanFromFirst([]byte("my"), 2)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my"), []byte("my_")})

	t.Log("test IteratorScanFromLast")
	list = it.IteratorScanFromLast([]byte("my"), 100)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/4"), []byte("my_key/3"), []byte("my_key/2"), []byte("my_key/1"), []byte("my_"), []byte("my")})

	t.Log("test IteratorScan 1")
	list = it.IteratorScan([]byte("my"), []byte("my_key/3"), 100, 1)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/4")})

	t.Log("test IteratorScan 0")
	list = it.IteratorScan([]byte("my"), []byte("my_key/3"), 100, 0)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/2"), []byte("my_key/1"), []byte("my_"), []byte("my")})
}

// 边界测试
func testDBBoundary2(t *testing.T, db DB) {
	a, _ := hex.DecodeString("ff")
	b, _ := hex.DecodeString("ffff")
	c, _ := hex.DecodeString("ffffff")
	d, _ := hex.DecodeString("ffffffff")
	db.Set(a, []byte("0xff"))
	db.Set(b, []byte("0xffff"))
	db.Set(c, []byte("0xffffff"))
	db.Set(d, []byte("0xffffffff"))

	values := [][]byte{[]byte("0xff"), []byte("0xffff"), []byte("0xffffff"), []byte("0xffffffff")}
	values_reverse := [][]byte{[]byte("0xffffffff"), []byte("0xffffff"), []byte("0xffff"), []byte("0xff")}
	var v []byte
	_ = v
	it := NewListHelper(db)

	// f为prefix
	t.Log("PrefixScan")
	list := it.PrefixScan(a)
	for i, v := range list {
		t.Log(i, string(v))
	}
	require.Equal(t, list, values)

	t.Log("IteratorScanFromFirst")
	list = it.IteratorScanFromFirst(a, 2)
	for i, v := range list {
		t.Log(i, string(v))
	}
	require.Equal(t, list, values[0:2])
	t.Log("IteratorScanFromLast")
	list = it.IteratorScanFromLast(a, 100)
	for i, v := range list {
		t.Log(i, string(v))
	}
	require.Equal(t, list, values_reverse)

	t.Log("IteratorScan 1") //seek 第二个
	list = it.IteratorScan(a, b, 100, 1)
	for i, v := range list {
		t.Log(i, string(v))
	}
	require.Equal(t, list, values[1:])

	t.Log("IteratorScan 0")
	list = it.IteratorScan(a, c, 100, 0)
	for i, v := range list {
		t.Log(i, string(v))
	}
	require.Equal(t, list, values_reverse[1:])
}

func testDBBoundary(t *testing.T, db DB) {
	a, _ := hex.DecodeString("0f")
	c, _ := hex.DecodeString("0fff")
	b, _ := hex.DecodeString("ff")
	d, _ := hex.DecodeString("ffff")
	db.Set(a, []byte("0x0f"))
	db.Set(c, []byte("0x0fff"))
	db.Set(b, []byte("0xff"))
	db.Set(d, []byte("0xffff"))

	var v []byte
	_ = v
	it := NewListHelper(db)

	// f为prefix
	t.Log("PrefixScan")
	list := it.PrefixScan(a)
	require.Equal(t, list, [][]byte{[]byte("0x0f"), []byte("0x0fff")})

	t.Log("IteratorScanFromFirst")
	list = it.IteratorScanFromFirst(a, 2)
	require.Equal(t, list, [][]byte{[]byte("0x0f"), []byte("0x0fff")})

	t.Log("IteratorScanFromLast")
	list = it.IteratorScanFromLast(a, 100)
	require.Equal(t, list, [][]byte{[]byte("0x0fff"), []byte("0x0f")})

	t.Log("IteratorScan 1")
	list = it.IteratorScan(a, a, 100, 1)
	require.Equal(t, list, [][]byte{[]byte("0x0fff")})

	t.Log("IteratorScan 0")
	list = it.IteratorScan(a, a, 100, 0)
	require.Equal(t, list, [][]byte(nil))

	// ff为prefix
	t.Log("PrefixScan")
	list = it.PrefixScan(b)
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScanFromFirst")
	list = it.IteratorScanFromFirst(b, 2)
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScanFromLast")
	list = it.IteratorScanFromLast(b, 100)
	require.Equal(t, list, [][]byte{[]byte("0xffff"), []byte("0xff")})

	t.Log("IteratorScan 1")
	list = it.IteratorScan(b, b, 100, 1)
	require.Equal(t, list, [][]byte{[]byte("0xffff")})

	t.Log("IteratorScan 0")
	list = it.IteratorScan(b, d, 100, 0)
	require.Equal(t, list, [][]byte{[]byte("0xff")})
}
