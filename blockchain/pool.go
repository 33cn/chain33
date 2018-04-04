package blockchain

import (
	"sync"

	"gitlab.33.cn/chain33/chain33/types"
)

type BlockPool struct {
	mtx      sync.Mutex
	synblock chan struct{}
	// 存储需要缓存的block
	recvBlocks map[int64]*types.BlockDetail
	broadcast  map[int64]bool
}

var poollog = chainlog.New("submodule", "pool")

func NewBlockPool() *BlockPool {
	bp := &BlockPool{
		recvBlocks: make(map[int64]*types.BlockDetail),
		broadcast:  make(map[int64]bool),
		synblock:   make(chan struct{}, 1),
	}
	return bp
}

//从缓存中删除已经加载到db中的block
func (pool *BlockPool) DelBlock(height int64) {
	pool.mtx.Lock()
	delete(pool.recvBlocks, height)
	delete(pool.broadcast, height)
	pool.mtx.Unlock()
	//poollog.Info("DelBlock", "Height", height)
}

// 暂时添加block到缓存中
func (pool *BlockPool) AddBlock(blockdetail *types.BlockDetail, broadcast bool) {
	pool.mtx.Lock()
	defer pool.mtx.Unlock()
	block := blockdetail.Block
	blockinfo := pool.recvBlocks[block.Height]
	if blockinfo != nil {
		poollog.Info("AddBlock existed", "Height", block.Height)
		return
	}
	pool.broadcast[block.Height] = broadcast
	pool.recvBlocks[block.Height] = blockdetail
	//poollog.Error("AddBlock", "Height", block.Height)
}

// chain 模块获取pool中缓存的block 加载到db中
func (pool *BlockPool) GetBlock(height int64) (*types.BlockDetail, bool) {
	pool.mtx.Lock()
	defer pool.mtx.Unlock()

	blockdetail := pool.recvBlocks[height]
	if blockdetail == nil || blockdetail.Block == nil {
		//poollog.Debug("GetBlock does not exist", "Height", height)
		return nil, false
	}
	//poollog.Debug("GetBlock", "Height", height)
	return pool.recvBlocks[height], pool.broadcast[height]
}
