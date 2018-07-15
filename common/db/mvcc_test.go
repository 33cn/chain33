package db

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestPrefix(t *testing.T) {
	assert.Equal(t, []byte(".-mvcc-."), mvccPrefix)
	assert.Equal(t, []byte(".-mvcc-.m."), mvccMeta)
	assert.Equal(t, []byte(".-mvcc-.d."), mvccData)
}

func TestPad(t *testing.T) {
	assert.Equal(t, []byte("00000000000000000001"), pad(1))
}

func getMVCC() *MVCCHelper {
	dir, err := ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}
	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	if err != nil {
		panic(err)
	}
	return NewMVCC(leveldb)
}

func closeMVCC(m *MVCCHelper) {
	m.db.Close()
}

func TestSetVersion(t *testing.T) {
	m := getMVCC()
	defer closeMVCC(m)
	hash := common.Sha256([]byte("1"))
	err := m.SetVersion(hash, int64(1))
	assert.Nil(t, err)
	v, err := m.GetVersion(hash)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), v)

	err = m.DelVersion(hash)
	assert.Nil(t, err)
	_, err = m.GetVersion(hash)
	assert.Equal(t, err, types.ErrNotFound)
}

func TestGetSaveKV(t *testing.T) {
	m := getMVCC()
	defer closeMVCC(m)
	kv := m.GetSaveKV([]byte("key"), []byte("value"), 1)
	assert.Equal(t, []byte(".-mvcc-.d.key.00000000000000000001"), kv.GetKey())
	assert.Equal(t, []byte("value"), kv.GetValue())
}

func TestGetDelKV(t *testing.T) {
	m := getMVCC()
	defer closeMVCC(m)
	kv := m.GetDelKV([]byte("key"), 1)
	assert.Equal(t, []byte(".-mvcc-.d.key.00000000000000000001"), kv.GetKey())
	assert.Equal(t, []byte(nil), kv.GetValue())
}

func TestVersionString(t *testing.T) {
	s, err := getVersionString([]byte(".-mvcc-.d.key.00000000000000000001"))
	assert.Nil(t, err)
	assert.Equal(t, "00000000000000000001", s)

	v, err := getVersion([]byte(".-mvcc-.d.key.00330721199901011111"))
	assert.Nil(t, err)
	assert.Equal(t, int64(330721199901011111), v)
}

func TestVersionSetAndGet(t *testing.T) {
	m := getMVCC()
	defer closeMVCC(m)
	b, err := m.GetV([]byte("key"), 0)
	assert.Equal(t, err, types.ErrNotFound)
	assert.Nil(t, b)

	//模拟这样的一个场景
	//0,k0,v0
	//1,k1,v1,k0,v01
	//2,k2,v2,k0,v02
	//3,k3,v3,k0,v03
	//4,kv,v4
	m.SetVersion(common.Sha256([]byte("0")), 0)
	m.SetV([]byte("k0"), []byte("v0"), 0)
	m.SetVersion(common.Sha256([]byte("1")), 1)
	m.SetV([]byte("k1"), []byte("v1"), 1)
	m.SetV([]byte("k0"), []byte("v01"), 1)
	m.SetVersion(common.Sha256([]byte("2")), 2)
	m.SetV([]byte("k2"), []byte("v2"), 2)
	m.SetVersion(common.Sha256([]byte("3")), 3)
	m.SetV([]byte("k3"), []byte("v3"), 3)
	m.SetV([]byte("k0"), []byte("v03"), 3)
	m.SetVersion(common.Sha256([]byte("4")), 4)
	m.SetV([]byte("k4"), []byte("v4"), 4)

	m.Trash(0)
	_, err = m.GetVersion(common.Sha256([]byte("5")))
	assert.Equal(t, err, types.ErrNotFound)

	version, err := m.GetVersion(common.Sha256([]byte("4")))
	assert.Nil(t, err)
	assert.Equal(t, version, int64(4))

	v, err := m.GetV([]byte("k0"), version)
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v03"))

	v, err = m.GetV([]byte("k4"), version)
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v4"))

	v, err = m.GetV([]byte("k0"), 0)
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v0"))

	v, err = m.GetV([]byte("k0"), 1)
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v01"))

	v, err = m.GetV([]byte("k0"), 2)
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v01"))

	v, err = m.GetV([]byte("k0"), 3)
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v03"))
}
