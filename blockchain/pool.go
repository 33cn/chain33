package blockchain

import (
	"sync"

	"code.aliyun.com/chain33/chain33/types"
)

type BlockPool struct {
	mtx      sync.Mutex
	synblock chan struct{}
	// 存储需要缓存的block
	recvBlocks map[int64]*types.Block
}

var poollog = chainlog.New("submodule", "pool")

func NewBlockPool() *BlockPool {
	bp := &BlockPool{
		recvBlocks: make(map[int64]*types.Block),
		synblock:   make(chan struct{}, 1),
	}
	return bp
}

//从缓存中删除已经加载到db中的block
func (pool *BlockPool) DelBlock(height int64) {
	pool.mtx.Lock()
	defer pool.mtx.Unlock()

	delete(pool.recvBlocks, height)
	//poollog.Info("DelBlock", "Height", height)
}

// 暂时添加block到缓存中
func (pool *BlockPool) AddBlock(block *types.Block) {
	pool.mtx.Lock()
	defer pool.mtx.Unlock()

	blockinfo := pool.recvBlocks[block.Height]
	if blockinfo != nil {
		poollog.Info("AddBlock existed", "Height", block.Height)
		return
	}
	pool.recvBlocks[block.Height] = block
	//poollog.Info("AddBlock", "Height", block.Height)
}

// chain 模块获取pool中缓存的block 加载到db中
func (pool *BlockPool) GetBlock(height int64) *types.Block {
	pool.mtx.Lock()
	defer pool.mtx.Unlock()

	block := pool.recvBlocks[height]
	if block == nil {
		//poollog.Debug("GetBlock does not exist", "Height", height)
		return nil
	}
	//poollog.Debug("GetBlock", "Height", height)
	return pool.recvBlocks[height]
}
