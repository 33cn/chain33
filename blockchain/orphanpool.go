// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"sync"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
)

var (
	maxOrphanBlocks = 10240 //最大孤儿block数量，考虑到同步阶段孤儿block会很多
)

const orphanExpirationTime = time.Second * 600 // 孤儿过期时间设置为10分钟

// 孤儿节点，就是本节点的父节点未知的block
type orphanBlock struct {
	block      *types.Block
	expiration time.Time
	broadcast  bool
	pid        string
	sequence   int64
}

// OrphanPool 孤儿节点的存储以blockhash作为map的索引。hash转换成string
type OrphanPool struct {
	orphanLock   sync.RWMutex
	orphans      map[string]*orphanBlock
	prevOrphans  map[string][]*orphanBlock
	oldestOrphan *orphanBlock
	param        *types.Chain33Config
}

// NewOrphanPool new
func NewOrphanPool(param *types.Chain33Config) *OrphanPool {
	op := &OrphanPool{
		orphans:     make(map[string]*orphanBlock),
		prevOrphans: make(map[string][]*orphanBlock),
		param:       param,
	}
	return op
}

// IsKnownOrphan 判断本节点是不是已知的孤儿节点
func (op *OrphanPool) IsKnownOrphan(hash []byte) bool {
	op.orphanLock.RLock()
	_, exists := op.orphans[string(hash)]
	op.orphanLock.RUnlock()

	return exists
}

// GetOrphanRoot 获取本孤儿节点的祖先节点hash在孤儿链中，没有的话就返回孤儿节点本身hash
func (op *OrphanPool) GetOrphanRoot(hash []byte) []byte {
	op.orphanLock.RLock()
	defer op.orphanLock.RUnlock()

	// Keep looping while the parent of each orphaned block is known and is an orphan itself.
	orphanRoot := hash
	prevHash := hash
	for {
		orphan, exists := op.orphans[string(prevHash)]
		if !exists {
			break
		}
		orphanRoot = prevHash
		prevHash = orphan.block.GetParentHash()
	}

	return orphanRoot
}

// RemoveOrphanBlockByHash 通过指定的区块hash
// 删除孤儿区块从OrphanPool中，以及prevOrphans中的index.
func (op *OrphanPool) RemoveOrphanBlockByHash(hash []byte) {
	op.orphanLock.Lock()
	defer op.orphanLock.Unlock()

	orphan, exists := op.orphans[string(hash)]
	if exists && orphan != nil {
		chainlog.Debug("RemoveOrphanBlockByHash:", "height", orphan.block.Height, "hash", common.ToHex(hash))
		op.removeOrphanBlock(orphan)
	}
}

// RemoveOrphanBlock 删除孤儿节点从OrphanPool中，以及prevOrphans中的index
func (op *OrphanPool) RemoveOrphanBlock(orphan *orphanBlock) {
	op.orphanLock.Lock()
	defer op.orphanLock.Unlock()

	chainlog.Debug("RemoveOrphanBlock:", "height", orphan.block.Height, "hash", common.ToHex(orphan.block.Hash(op.param)))

	op.removeOrphanBlock(orphan)
}

// RemoveOrphanBlock2 删除孤儿节点从OrphanPool中，以及prevOrphans中的index
func (op *OrphanPool) RemoveOrphanBlock2(block *types.Block, expiration time.Time, broadcast bool, pid string, sequence int64) {
	b := &orphanBlock{
		block:      block,
		expiration: expiration,
		broadcast:  broadcast,
		pid:        pid,
		sequence:   sequence,
	}
	op.RemoveOrphanBlock(b)
}

// 删除孤儿节点从OrphanPool中，以及prevOrphans中的index
func (op *OrphanPool) removeOrphanBlock(orphan *orphanBlock) {
	chainlog.Debug("removeOrphanBlock:", "orphan.block.height", orphan.block.Height, "orphan.block.hash", common.ToHex(orphan.block.Hash(op.param)))

	// 从orphan pool中删除孤儿节点
	orphanHash := orphan.block.Hash(op.param)
	delete(op.orphans, string(orphanHash))

	// 删除parent hash中本孤儿节点的index
	prevHash := orphan.block.GetParentHash()
	orphans := op.prevOrphans[string(prevHash)]
	for i := 0; i < len(orphans); i++ {
		hash := orphans[i].block.Hash(op.param)
		if bytes.Equal(hash, orphanHash) {
			copy(orphans[i:], orphans[i+1:])
			orphans[len(orphans)-1] = nil
			orphans = orphans[:len(orphans)-1]
			i--
		}
	}
	op.prevOrphans[string(prevHash)] = orphans

	// parent hash对应的子孤儿节点都已经删除，也需要删除本parent hash
	if len(op.prevOrphans[string(prevHash)]) == 0 {
		delete(op.prevOrphans, string(prevHash))
	}
}

// AddOrphanBlock adds the passed block (which is already determined to be
// an orphan prior calling this function) to the orphan pool.  It lazily cleans
// up any expired blocks so a separate cleanup poller doesn't need to be run.
// It also imposes a maximum limit on the number of outstanding orphan
// blocks and will remove the oldest received orphan block if the limit is
// exceeded.
func (op *OrphanPool) AddOrphanBlock(broadcast bool, block *types.Block, pid string, sequence int64) {

	chainlog.Debug("addOrphanBlock:", "block.height", block.Height, "block.hash", common.ToHex(block.Hash(op.param)))

	op.orphanLock.Lock()
	defer op.orphanLock.Unlock()

	// 删除过期的孤儿节点从孤儿池中
	for _, oBlock := range op.orphans {
		if types.Now().After(oBlock.expiration) {
			chainlog.Debug("addOrphanBlock:removeOrphanBlock expiration", "block.height", oBlock.block.Height, "block.hash", common.ToHex(oBlock.block.Hash(op.param)))

			op.removeOrphanBlock(oBlock)
			continue
		}
		// 更新最早的孤儿block
		if op.oldestOrphan == nil || oBlock.expiration.Before(op.oldestOrphan.expiration) {
			op.oldestOrphan = oBlock
		}
	}

	// 孤儿池超过最大限制时，删除最早的一个孤儿block
	if (len(op.orphans) + 1) > maxOrphanBlocks {
		op.removeOrphanBlock(op.oldestOrphan)
		chainlog.Debug("addOrphanBlock:removeOrphanBlock maxOrphanBlocks ", "block.height", op.oldestOrphan.block.Height, "block.hash", common.ToHex(op.oldestOrphan.block.Hash(op.param)))

		op.oldestOrphan = nil
	}

	// 将本孤儿节点插入孤儿池中，并启动过期定时器
	expiration := types.Now().Add(orphanExpirationTime)
	oBlock := &orphanBlock{
		block:      block,
		expiration: expiration,
		broadcast:  broadcast,
		pid:        pid,
		sequence:   sequence,
	}
	op.orphans[string(block.Hash(op.param))] = oBlock

	// 将本孤儿节点添加到其父hash对应的map列表中，方便快速查找
	prevHash := block.GetParentHash()
	op.prevOrphans[string(prevHash)] = append(op.prevOrphans[string(prevHash)], oBlock)
}

// getChildOrphanCount 获取父hash对应的子孤儿节点的个数,内部函数
func (op *OrphanPool) getChildOrphanCount(hash string) int {
	return len(op.prevOrphans[hash])
}

// getChildOrphan 获取子孤儿连，内部函数
func (op *OrphanPool) getChildOrphan(hash string, index int) *orphanBlock {
	if index >= len(op.prevOrphans[hash]) {
		return nil
	}
	return op.prevOrphans[hash][index]
}

// ProcessOrphans 孤儿链的处理,将本hash对应的子block插入chain中
func (op *OrphanPool) ProcessOrphans(hash []byte, b *BlockChain) error {
	chainlog.Debug("ProcessOrphans:parent", "hash", common.ToHex(hash))
	op.orphanLock.Lock()
	defer op.orphanLock.Unlock()

	processHashes := make([]string, 0, 100)
	processHashes = append(processHashes, string(hash))
	for len(processHashes) > 0 {
		processHash := processHashes[0]
		processHashes[0] = "" // Prevent GC leak.
		processHashes = processHashes[1:]

		//  处理以processHash为父hash的所有子block
		count := b.orphanPool.getChildOrphanCount(processHash)
		for i := 0; i < count; i++ {
			orphan := b.orphanPool.getChildOrphan(processHash, i)
			if orphan == nil {
				chainlog.Debug("ProcessOrphans", "Found a nil entry at index", i, "orphan dependency list for block", common.ToHex([]byte(processHash)))
				continue
			}

			// 从孤儿池中删除此孤儿节点
			orphanHash := orphan.block.Hash(op.param)
			b.orphanPool.removeOrphanBlock(orphan)
			i--

			chainlog.Debug("processOrphans:maybeAcceptBlock", "height", orphan.block.GetHeight(), "hash", common.ToHex(orphan.block.Hash(op.param)))
			// 尝试将此孤儿节点添加到主链
			_, _, err := b.maybeAcceptBlock(orphan.broadcast, &types.BlockDetail{Block: orphan.block}, orphan.pid, orphan.sequence)
			if err != nil {
				return err
			}
			processHashes = append(processHashes, string(orphanHash))
		}
	}
	return nil
}
