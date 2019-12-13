package blockchain

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"

	bmocks "github.com/33cn/chain33/blockchain/mocks"
)

func Test_recommendSeqs(t *testing.T) {
	seqs := recommendSeqs(2, 10)
	assert.Equal(t, 3, len(seqs))

	seqs = recommendSeqs(20, 10)
	assert.Equal(t, 6, len(seqs))
	assert.Equal(t, []int64{20, 19, 18, 17, 16, 0}, seqs)

	seqs = recommendSeqs(10000, 6)
	assert.Equal(t, 6, len(seqs))
	assert.Equal(t, []int64{10000, 9999, 9998, 9898, 9798, 9698}, seqs)

	seqs = recommendSeqs(200, 6)
	assert.Equal(t, 5, len(seqs))
	assert.Equal(t, []int64{200, 199, 198, 98, 0}, seqs)
}

func Test_loadSequanceForAddCallback(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	store := &BlockStore{
		db:           dbm.NewDB("blockchain", "leveldb", dir, 100),
		saveSequence: true,
	}
	cb := &types.BlockSeqCB{
		Name:          "D7",
		URL:           "http://192.1.1.1:101",
		Encode:        "json",
		LastSequence:  11,
		LastHeight:    11,
		LastBlockHash: "0xae42f51d3e21ef6169ab2c40892b88359d069008178040cde07e87fc8f37fdd0",
	}
	loadSequanceForAddCallback(store, cb)
}

func Test_PushServiceRPC(t *testing.T) {
	seqStore := new(bmocks.SequenceStore)
	pushStore := new(bmocks.CommonStore)
	work := new(bmocks.PushWorkNotify)

	cb1 := types.BlockSeqCB{
		Name:     "cb1",
		URL:      "url1",
		Encode:   "json",
		IsHeader: false,
	}

	s := newPushService(seqStore, pushStore)
	work.On("AddTask", mock.Anything).Return()
	pushStore.On("PrefixCount", mock.Anything).Return(int64(0), nil).Once()
	pushStore.On("SetSync", calcSeqCBKey([]byte(cb1.Name)), types.Encode(&cb1)).Return(nil)
	seq1, err1 := s.AddCallback(work, &cb1)
	assert.Nil(t, err1)
	assert.Nil(t, seq1)

	pushStore.On("PrefixCount", seqCBPrefix).Return(int64(1), nil)
	pushStore.On("List", seqCBPrefix).Return([][]byte{types.Encode(&cb1)}, nil)
	pushSeqs, err := s.ListCallback()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(pushSeqs.GetItems()))
	assert.Equal(t, cb1.URL, pushSeqs.Items[0].URL)

	pushStore.On("GetKey", calcSeqCBLastNumKey([]byte(cb1.Name))).Return(types.Encode(&types.Int64{Data: 3}), nil)
	seqNum := s.GetLastPushSeq(cb1.Name)
	assert.Equal(t, int64(3), seqNum)
}

func Test_PushServiceAdd(t *testing.T) {
	seqStore := new(bmocks.SequenceStore)
	pushStore := new(bmocks.CommonStore)
	work := new(bmocks.PushWorkNotify)

	cb1 := types.BlockSeqCB{
		Name:     "cb1",
		URL:      "url1",
		Encode:   "json",
		IsHeader: false,
	}

	cb2 := types.BlockSeqCB{
		Name:          "cb2",
		URL:           "url2",
		Encode:        "json",
		IsHeader:      true,
		LastSequence:  int64(2),
		LastHeight:    int64(2),
		LastBlockHash: common.ToHex([]byte("match2")),
	}

	cb3 := types.BlockSeqCB{
		Name:          "cb3",
		URL:           "url2",
		Encode:        "json",
		IsHeader:      true,
		LastSequence:  int64(3),
		LastHeight:    int64(3),
		LastBlockHash: common.ToHex([]byte("not-match")),
	}

	cb4 := types.BlockSeqCB{
		Name:          "cb4",
		URL:           "url2",
		Encode:        "json",
		IsHeader:      true,
		LastSequence:  int64(7),
		LastHeight:    int64(7),
		LastBlockHash: common.ToHex([]byte("not-match")),
	}

	cb5 := types.BlockSeqCB{
		Name:          "cb5",
		URL:           "url2",
		Encode:        "json",
		IsHeader:      true,
		LastSequence:  int64(-1),
		LastHeight:    int64(2),
		LastBlockHash: common.ToHex([]byte("match2")),
	}

	s := newPushService(seqStore, pushStore)

	// 模拟只注册了cb1
	work.On("AddTask", mock.Anything).Return()
	pushStore.On("GetKey", calcSeqCBKey([]byte(cb1.Name))).Return(nil, nil)
	pushStore.On("GetKey", mock.Anything).Return(nil, types.ErrNotFound)
	pushStore.On("PrefixCount", mock.Anything).Return(int64(1), nil)
	pushStore.On("SetSync", calcSeqCBKey([]byte(cb1.Name)), types.Encode(&cb1)).Return(nil)

	// seq 匹配的情况
	seqStore.On("LastHeader").Return(&types.Header{Height: 6, Hash: []byte("match6")})
	seqStore.On("LoadBlockLastSequence").Return(int64(6), nil)
	for i := 0; i <= 6; i++ {
		sequence := types.BlockSequence{Hash: []byte(fmt.Sprintf("match%d", i)), Type: 1}
		seqStore.On("GetBlockSequence", int64(i)).Return(&sequence, nil)
		seqStore.On("GetSequenceByHash", []byte(fmt.Sprintf("match%d", i))).Return(int64(i), nil)
		header := types.Header{Height: int64(i), Hash: []byte(fmt.Sprintf("match%d", i))}
		seqStore.On("GetBlockHeaderByHash", []byte(fmt.Sprintf("match%d", i))).Return(&header, nil)
	}

	work.On("AddTask", mock.Anything).Return()
	pushStore.On("SetSync", calcSeqCBLastNumKey([]byte(cb2.Name)), mock.Anything).Return(nil)
	pushStore.On("SetSync", calcSeqCBKey([]byte(cb2.Name)), mock.Anything).Return(nil)

	seq1, err1 := s.AddCallback(work, &cb2)
	assert.Nil(t, err1)
	assert.Nil(t, seq1)

	// seq 不匹配的情况
	_, err1 = s.AddCallback(work, &cb3)
	assert.Equal(t, types.ErrSequenceNotMatch, err1)

	// seq 过大
	seq1, err1 = s.AddCallback(work, &cb4)
	assert.Equal(t, types.ErrSequenceTooBig, err1)
	assert.Nil(t, seq1)

	seq1, err1 = s.AddCallback(work, &cb5)
	assert.Equal(t, types.ErrSequenceNotMatch, err1)
	assert.Equal(t, 3, len(seq1))
	assert.Equal(t, int64(2), seq1[0].Sequence)
	assert.Equal(t, int64(2), seq1[0].Height)
}
