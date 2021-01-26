package p2pstore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

// prefix key and const parameters
const (
	LocalChunkInfoKey = "local-chunk-info"
	ChunkNameSpace    = "chunk"
	ChunkPrefix       = "chunk-"
	AlphaValue        = 3
	Backup            = 100
)

//LocalChunkInfo wraps local chunk key with time.
type LocalChunkInfo struct {
	*types.ChunkInfoMsg
	Time time.Time
}

// 保存chunk到本地p2pStore，同时更新本地chunk列表
func (p *Protocol) addChunkBlock(info *types.ChunkInfoMsg, bodys *types.BlockBodys) error {
	if int64(len(bodys.Items)) != info.End-info.Start+1 {
		return types2.ErrLength
	}
	for i := info.Start; i <= info.End; i++ {
		key := genChunkDBKey(i)
		if err := p.DB.Set(key, types.Encode(bodys.Items[i-info.Start])); err != nil {
			return err
		}
	}
	return p.addLocalChunkInfo(info)
}

// 更新本地chunk保存时间，只更新索引即可
func (p *Protocol) updateChunk(req *types.ChunkInfoMsg) error {
	mapKey := hex.EncodeToString(req.ChunkHash)
	p.localChunkInfoMutex.Lock()
	defer p.localChunkInfoMutex.Unlock()
	if info, ok := p.localChunkInfo[mapKey]; ok {
		info.Time = time.Now()
		p.localChunkInfo[mapKey] = info
		return nil
	}

	return types2.ErrNotFound
}

func (p *Protocol) deleteChunkBlock(hash []byte) error {
	err := p.deleteLocalChunkInfo(hash)
	if err != nil {
		return err
	}
	batch := p.DB.NewBatch(false)
	it := p.DB.Iterator(hash, append(hash, ':'+1), false)
	defer it.Close()
	for it.Next(); it.Valid(); it.Next() {
		batch.Delete(it.Key())
	}
	return batch.Write()
}

// 获取本地chunk数据
//	本地不存在，返回not found
//  本地存在：
//		数据未过期：返回数据
//		数据已过期：返回数据,然后从数据库删除该数据
func (p *Protocol) getChunkBlock(req *types.ChunkInfoMsg) (*types.BlockBodys, error) {

	if _, ok := p.getChunkInfoByHash(req.ChunkHash); !ok {
		return nil, types2.ErrNotFound
	}
	var bodys []*types.BlockBody
	it := p.DB.Iterator(genChunkDBKey(req.Start), genChunkDBKey(req.End+1), false)
	defer it.Close()
	for it.Next(); it.Valid(); it.Next() {
		var body types.BlockBody
		if err := types.Decode(it.Value(), &body); err != nil {
			return nil, err
		}
		bodys = append(bodys, &body)
	}
	if int64(len(bodys)) != req.End-req.Start+1 {
		return nil, types2.ErrLength
	}

	return &types.BlockBodys{Items: bodys}, nil
}

// 保存一个本地chunk hash列表，用于遍历本地数据
func (p *Protocol) addLocalChunkInfo(info *types.ChunkInfoMsg) error {
	p.localChunkInfoMutex.Lock()
	defer p.localChunkInfoMutex.Unlock()

	p.localChunkInfo[hex.EncodeToString(info.ChunkHash)] = LocalChunkInfo{
		ChunkInfoMsg: info,
		Time:         time.Now(),
	}
	return p.saveLocalChunkInfoMap(p.localChunkInfo)
}

func (p *Protocol) deleteLocalChunkInfo(hash []byte) error {
	p.localChunkInfoMutex.Lock()
	defer p.localChunkInfoMutex.Unlock()
	delete(p.localChunkInfo, hex.EncodeToString(hash))
	return p.saveLocalChunkInfoMap(p.localChunkInfo)
}

func (p *Protocol) initLocalChunkInfoMap() {
	p.localChunkInfoMutex.Lock()
	defer p.localChunkInfoMutex.Unlock()
	p.localChunkInfo = make(map[string]LocalChunkInfo)
	value, err := p.DB.Get([]byte(LocalChunkInfoKey))
	if err != nil {
		log.Error("initLocalChunkInfoMap", "error", err)
		return
	}

	err = json.Unmarshal(value, &p.localChunkInfo)
	if err != nil {
		panic(err)
	}
	for k, v := range p.localChunkInfo {
		info := v
		info.Time = time.Now()
		p.localChunkInfo[k] = info
	}
}

func (p *Protocol) saveLocalChunkInfoMap(m map[string]LocalChunkInfo) error {
	value, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return p.DB.Set([]byte(LocalChunkInfoKey), value)
}

func (p *Protocol) getChunkInfoByHash(hash []byte) (LocalChunkInfo, bool) {
	p.localChunkInfoMutex.RLock()
	defer p.localChunkInfoMutex.RUnlock()
	info, ok := p.localChunkInfo[hex.EncodeToString(hash)]
	return info, ok
}

// 适配libp2p，按路径格式生成数据的key值，便于区分多种数据类型的命名空间，以及key值合法性校验
func genChunkNameSpaceKey(hash []byte) string {
	return fmt.Sprintf("/%s/%s", ChunkNameSpace, hex.EncodeToString(hash))
}

func genDHTID(chunkHash []byte) kb.ID {
	return kb.ConvertKey(genChunkNameSpaceKey(chunkHash))
}

func formatHeight(height int64) string {
	return fmt.Sprintf("%012d", height)
}

func genChunkDBKey(height int64) []byte {
	var key []byte
	key = append(key, ChunkPrefix...)
	key = append(key, formatHeight(height)...)
	return key
}
