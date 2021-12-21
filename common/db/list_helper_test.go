package db

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListHelper_List(t *testing.T) {
	dir, err := ioutil.TempDir("", "listdb")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	ldb, err := NewGoLevelDB("level", dir, 128)
	require.Nil(t, err)
	defer ldb.Close()
	testListDB(t, ldb)
	bdb, err := NewGoBadgerDB("badger", dir, 128)
	require.Nil(t, err)
	defer bdb.Close()
	testListDB(t, bdb)
}

func testListDB(t *testing.T, db DB) {

	ldb := NewListHelper(db)
	db.Set([]byte("key1"), []byte("value1"))
	db.Set([]byte("key4"), []byte("value2"))
	db.Set([]byte("key7"), []byte("value3"))
	data := ldb.List([]byte("key"), []byte("key0"), 0, ListASC)
	require.Equal(t, 3, len(data))
	data = ldb.List([]byte("key"), []byte("key1"), 0, ListASC)
	require.Equal(t, 2, len(data))
	data = ldb.List([]byte("key"), []byte("key3"), 0, ListASC)
	require.Equal(t, 2, len(data))
	data = ldb.List([]byte("key"), []byte("key4"), 0, ListASC)
	require.Equal(t, 1, len(data))
	data = ldb.List([]byte("key"), []byte("key7"), 0, ListASC)
	require.Equal(t, 0, len(data))
	data = ldb.List([]byte("key"), []byte("key8"), 0, ListDESC)
	require.Equal(t, 3, len(data))
	data = ldb.List([]byte("key"), []byte("key7"), 0, ListDESC)
	require.Equal(t, 2, len(data))
	data = ldb.List([]byte("key"), []byte("key5"), 0, ListDESC)
	require.Equal(t, 2, len(data))
	data = ldb.List([]byte("key"), []byte("key4"), 0, ListDESC)
	require.Equal(t, 1, len(data))
	data = ldb.List([]byte("key"), []byte("key1"), 0, ListDESC)
	require.Equal(t, 0, len(data))
}
