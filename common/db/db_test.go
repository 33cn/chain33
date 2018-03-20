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
	require.Equal(t, list, [][]byte{[]byte("my_key/3"), []byte("my_key/4")})

	t.Log("test IteratorScan 0")
	list = it.IteratorScan([]byte("my"), []byte("my_key/3"), 100, 0)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/3"), []byte("my_key/2"), []byte("my_key/1"), []byte("my_"), []byte("my")})
}

// 边界测试
func testDBBoundary(t *testing.T, db DB) {
return
	a, _ := hex.DecodeString("f")
	b, _ := hex.DecodeString("ff")
	c, _ := hex.DecodeString("fff")
	d, _ := hex.DecodeString("ffff")
        db.Set(a, []byte("0xf"))
        db.Set(b, []byte("0xff"))
        db.Set(c, []byte("0xfff"))
        db.Set(d, []byte("0xffff"))

	var v []byte
	_ = v
	it := NewListHelper(db)

	// f为prefix
	t.Log("PrefixScan")
	list := it.PrefixScan(a)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScanFromFirst")
	list = it.IteratorScanFromFirst(a, 2)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScanFromLast")
	list = it.IteratorScanFromLast(a, 100)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("0xffff"), []byte("0xff")})

	t.Log("IteratorScan 1")
	list = it.IteratorScan(a, []byte("0xff"), 100, 1)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScan 0")
	list = it.IteratorScan(a, []byte("0xff"), 100, 0)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	// ff为prefix
	t.Log("PrefixScan")
	list = it.PrefixScan(b)
	for _, v = range list {
		t.Log(string(v))
	}
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScanFromFirst")
	list = it.IteratorScanFromFirst(b, 2)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScanFromLast")
	list = it.IteratorScanFromLast(b, 100)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("0xffff"), []byte("0xff")})

	t.Log("IteratorScan 1")
	list = it.IteratorScan(b, []byte("0xff"), 100, 1)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScan 0")
	list = it.IteratorScan(b, []byte("0xff"), 100, 0)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})
}

