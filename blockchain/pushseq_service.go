// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

//ProcListBlockSeqCB 列出所有已经设置的seq callback
func (chain *BlockChain) ProcListBlockSeqCB() (*types.BlockSeqCBs, error) {
	return chain.pushservice.ListCallback()
}

//ProcGetSeqCBLastNum 获取指定name的callback已经push的最新seq num
func (chain *BlockChain) ProcGetSeqCBLastNum(name string) int64 {
	return chain.pushservice.GetLastPushSeq(name)
}

//ProcAddBlockSeqCB 添加seq callback
func (chain *BlockChain) ProcAddBlockSeqCB(cb *types.BlockSeqCB) ([]*types.Sequence, error) {
	if cb == nil {
		chainlog.Error("ProcAddBlockSeqCB input hash is null")
		return nil, types.ErrInvalidParam
	}

	if !chain.isRecordBlockSequence {
		chainlog.Error("ProcAddBlockSeqCB not support sequence")
		return nil, types.ErrRecordBlockSequence
	}
	return chain.pushservice.AddCallback(chain.pushseq, cb)
}

// 推送服务
// 1. 需要一个store， 读取seq 相关信息: 包括 seq -> block/height/hash
// 1. 需要一个store， 读写推送相关信息： 包含 注册信息和推送进度
// 1. 一组rpc， 进行推送管理
// 1. 一组真实推送的模块： pushseq文件

// SequenceStore 第一store： 满足获得 seq -> block 的信息获得
// 实现接口先用现有的blockstroe， 先分开代码
type SequenceStore interface {
	LoadBlockLastSequence() (int64, error)
	// seq -> block sequence
	GetBlockSequence(seq int64) (*types.BlockSequence, error)
	// hash -> block header
	GetBlockHeaderByHash(hash []byte) (*types.Header, error)
	// seq -> block, size
	LoadBlockBySequence(seq int64) (*types.BlockDetail, int, error)
	// get last header
	LastHeader() *types.Header
	// hash -> seq
	GetSequenceByHash(hash []byte) (int64, error)
}

// PushWorkNotify 两类notify
// 1. last sequence 变化
// 2. callback 变化
// 函数名不变， 可以先不改pushseq的代码
type PushWorkNotify interface {
	AddTask(cb *types.BlockSeqCB)
	UpdateSeq(seq int64)
}

// PushService rpc接口转发
// 外部接口通过 rpc -> queue -> chain 过来， 接口不变
// TODO 以后或许推送服务整个可以插件化
type PushService interface {
	Add()
	List()
	Get()
}

// PushService1 实现
// 放一个chain的指针，简单的分开代码
type PushService1 struct {
	seqStore  SequenceStore
	pushStore *PushSeqStore1
}

func newPushService(seqStore SequenceStore, bcStore CommonStore) *PushService1 {
	return &PushService1{seqStore: seqStore, pushStore: &PushSeqStore1{store: bcStore}}
}

// ListCallback List Callback
func (push *PushService1) ListCallback() (*types.BlockSeqCBs, error) {
	cbs, err := push.pushStore.ListCB()
	if err != nil {
		chainlog.Error("ProcListBlockSeqCB", "err", err.Error())
		return nil, err
	}
	var listSeqCBs types.BlockSeqCBs

	listSeqCBs.Items = append(listSeqCBs.Items, cbs...)

	return &listSeqCBs, nil
}

// GetLastPushSeq 获取指定name的callback已经push的最新seq num
func (push *PushService1) GetLastPushSeq(name string) int64 {
	return push.pushStore.GetLastPushSeq(name)
}

// AddCallback 添加seq callback
func (push *PushService1) AddCallback(pushseq PushWorkNotify, cb *types.BlockSeqCB) ([]*types.Sequence, error) {
	if cb == nil {
		chainlog.Error("AddCallback input hash is null")
		return nil, types.ErrInvalidParam
	}

	if cb.LastBlockHash != "" || cb.LastSequence != 0 || cb.LastHeight != 0 {
		if cb.LastBlockHash == "" || cb.LastSequence == 0 || cb.LastHeight == 0 {
			chainlog.Error("AddCallback ErrInvalidParam", "seq", cb.LastSequence, "height", cb.LastHeight, "hash", cb.LastBlockHash)
			return nil, types.ErrInvalidParam
		}
	}

	if push.pushStore.CallbackCount() >= MaxSeqCB && !push.pushStore.CallbackExist(cb.Name) {
		chainlog.Error("ProcAddBlockSeqCB too many seq callback")
		return nil, types.ErrTooManySeqCB
	}

	// 在不指定sequence时, 和原来行为保存一直
	if cb.LastSequence == 0 {
		err := push.pushStore.AddCallback(cb)
		if err != nil {
			chainlog.Error("ProcAddBlockSeqCB", "addBlockSeqCB", err)
			return nil, err
		}
		pushseq.AddTask(cb)
		return nil, nil
	}

	// 处理带 last sequence, 推送续传的情况
	chainlog.Debug("ProcAddBlockSeqCB continue-seq-push", "name", cb.Name, "seq", cb.LastSequence,
		"hash", cb.LastBlockHash, "height", cb.LastHeight)
	// name 是否存在， 存在就继续，不需要重新注册了
	if push.pushStore.CallbackExist(cb.Name) {
		chainlog.Info("ProcAddBlockSeqCB continue-seq-push", "exist", cb.Name)
		return nil, nil
	}

	lastHeader := push.seqStore.LastHeader()
	// 续传的情况下， 最好等节点同步过了原先的点， 不然同步好的删除了， 等于重新同步
	if lastHeader.Height < cb.LastHeight {
		chainlog.Error("ProcAddBlockSeqCB continue-seq-push", "last-height", lastHeader.Height, "input-height", cb.LastHeight, "err", types.ErrSequenceTooBig)
		return nil, types.ErrSequenceTooBig
	}

	if cb.LastSequence > 0 {
		// name不存在：Sequence 信息匹配，添加
		sequence, err := push.seqStore.GetBlockSequence(cb.LastSequence)
		if err != nil {
			chainlog.Error("ProcAddBlockSeqCB continue-seq-push", "load-1", err)
			return nil, err
		}

		// 注册点，在节点上存在
		// 同一高度，不一定同一个hash，有分叉的可能；但同一个hash必定同一个高度
		reloadHash := common.ToHex(sequence.Hash)
		if cb.LastBlockHash == reloadHash {
			// 先填入last seq， 而不是从0开始
			err = push.pushStore.SetLastPushSeqSync([]byte(cb.Name), cb.LastSequence)
			if err != nil {
				chainlog.Error("ProcAddBlockSeqCB", "setSeqCBLastNum", err)
				return nil, err
			}
			err = push.pushStore.AddCallback(cb)
			if err != nil {
				chainlog.Error("ProcAddBlockSeqCB", "addBlockSeqCB", err)
				return nil, err
			}
			pushseq.AddTask(cb)
			return nil, nil
		}
	}

	// 支持已经从N height之后才有业务的情况
	// 指定height，hash， 推荐sequence 再来注册
	if cb.LastSequence == -1 {
		hexHash, err := common.FromHex(cb.LastBlockHash)
		if err != nil {
			chainlog.Error("ProcAddBlockSeqCB common.FromHex", "err", err, "hash", cb.LastBlockHash)
			return nil, err
		}
		seq, err := push.seqStore.GetSequenceByHash(hexHash)
		if err != nil {
			chainlog.Error("ProcAddBlockSeqCB GetSequenceByHash", "err", err, "hash", cb.LastBlockHash)
			return nil, err
		}
		cb.LastSequence = seq
	}

	// 注册点，在节点上不存在， 即分叉上
	// name不存在， 但对应的Hash/Height对不上
	return loadSequanceForAddCallback(push.seqStore, cb)
}

// add callback时， name不存在， 但对应的Hash/Height对不上, 加载推荐的开始点
// 1. 在接近的sequence推荐，解决分叉问题
// 2. 跳跃的sequence推荐，解决在极端情况下， 有比较深的分叉， 减少交互的次数
func loadSequanceForAddCallback(store SequenceStore, cb *types.BlockSeqCB) ([]*types.Sequence, error) {
	seqsNumber := recommendSeqs(cb.LastSequence, types.MaxBlockCountPerTime)

	seqs := make([]*types.Sequence, 0)
	for _, i := range seqsNumber {
		seq, err := loadOneSeq(store, i)
		if err != nil {
			continue
		}
		seqs = append(seqs, seq)
	}
	return seqs, types.ErrSequenceNotMatch
}

// 推荐开始点选取
func recommendSeqs(lastSequence, max int64) []int64 {
	count := int64(100)
	skip := int64(100)
	skipTimes := int64(100)
	if count+skipTimes > max {
		count = max / 2
		skipTimes = max / 2
	}

	seqs := make([]int64, 0)

	start := lastSequence - count
	if start < 0 {
		start = 0
	}
	cur := lastSequence
	for ; cur > start; cur-- {
		seqs = append(seqs, cur)
	}

	cur = start + 1 - skip
	for ; cur > 0; cur = cur - skip {
		skipTimes--
		if skipTimes < 0 {
			break
		}
		seqs = append(seqs, cur)
	}
	if cur <= 0 {
		seqs = append(seqs, 0)
	}

	return seqs
}

func loadOneSeq(store SequenceStore, cur int64) (*types.Sequence, error) {
	seq, err := store.GetBlockSequence(cur)
	if err != nil || seq == nil {
		chainlog.Warn("ProcAddBlockSeqCB continue-seq-push", "load-2", err, "seq", cur)
		return nil, err
	}
	header, err := store.GetBlockHeaderByHash(seq.Hash)
	if err != nil || header == nil {
		chainlog.Warn("ProcAddBlockSeqCB continue-seq-push", "load-2", err, "seq", cur, "hash", common.ToHex(seq.Hash))
		return nil, err
	}
	return &types.Sequence{Hash: seq.Hash, Type: seq.Type, Sequence: cur, Height: header.Height}, nil
}

// PushSeqStrore 第二store， 读写推送相关信息的读写
type PushSeqStrore interface {
}

// CommonStore 通用的store 接口
// 修改大一点，可能可以用 db.KVDB
// 先改动小一点， 用store, 如果接口一样可以直接换
type CommonStore interface {
	SetSync(key, value []byte) error
	Set(key, value []byte) error
	GetKey(key []byte) ([]byte, error)
	PrefixCount(prefix []byte) int64
	List(prefix []byte) ([][]byte, error)
}

// PushSeqStore1 store
// 两组接口： 和注册相关的， 和推送进行到seq相关的
type PushSeqStore1 struct {
	store CommonStore
}

// AddCallback push seq callback
func (push *PushSeqStore1) AddCallback(cb *types.BlockSeqCB) error {
	if len(cb.Name) > 128 || len(cb.URL) > 1024 {
		return types.ErrInvalidParam
	}
	storeLog.Info("addBlockSeqCB", "key", string(calcSeqCBKey([]byte(cb.Name))), "value", cb)
	return push.store.SetSync(calcSeqCBKey([]byte(cb.Name)), types.Encode(cb))
}

// CallbackCount Callback Count
func (push *PushSeqStore1) CallbackCount() int64 {
	return push.store.PrefixCount(seqCBPrefix)
}

// CallbackExist Callback Exist
func (push *PushSeqStore1) CallbackExist(name string) bool {
	value, err := push.store.GetKey(calcSeqCBKey([]byte(name)))
	if err == nil {
		var cb types.BlockSeqCB
		err = types.Decode(value, &cb)
		return err == nil
	}
	return false
}

// ListCB List callback
func (push *PushSeqStore1) ListCB() (cbs []*types.BlockSeqCB, err error) {
	values, err := push.store.List(seqCBPrefix)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		var cb types.BlockSeqCB
		err := types.Decode(value, &cb)
		if err != nil {
			return nil, err
		}
		cbs = append(cbs, &cb)
	}
	return cbs, nil
}

// GetLastPushSeq Seq的合法值从0开始的，所以没有获取到或者获取失败都应该返回-1
func (push *PushSeqStore1) GetLastPushSeq(name string) int64 {
	bytes, err := push.store.GetKey(calcSeqCBLastNumKey([]byte(name)))
	if bytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("getSeqCBLastNum", "error", err)
		}
		return -1
	}
	n, err := decodeHeight(bytes)
	if err != nil {
		return -1
	}
	storeLog.Error("getSeqCBLastNum", "name", name, "num", n)

	return n
}

// SetLastPushSeqSync 更新推送进度
func (push *PushSeqStore1) SetLastPushSeqSync(name []byte, num int64) error {
	return push.store.SetSync(calcSeqCBLastNumKey(name), types.Encode(&types.Int64{Data: num}))
}

// SetLastPushSeq 更新推送进度
func (push *PushSeqStore1) SetLastPushSeq(name []byte, num int64) error {
	return push.store.SetSync(calcSeqCBLastNumKey(name), types.Encode(&types.Int64{Data: num}))
}
