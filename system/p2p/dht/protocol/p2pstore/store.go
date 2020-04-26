package p2pstore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/ipfs/go-datastore"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

const (
	LocalChunkInfoKey = "local-chunk-info"
	ChunkNameSpace    = "chunk"
	AlphaValue        = 3
	Backup            = 3
)

type LocalChunkInfo struct {
	*types.ChunkInfoMsg
	Time time.Time
}

// 保存chunk到本地p2pStore，同时更新本地chunk列表
func (p *Protocol) addChunkBlock(info *types.ChunkInfoMsg, bodys *types.BlockBodys) error {
	//先检查数据是不是正在保存
	if _, ok := p.saving.LoadOrStore(hex.EncodeToString(info.ChunkHash), nil); ok {
		return nil
	}
	defer p.saving.Delete(hex.EncodeToString(info.ChunkHash))

	err := p.addLocalChunkInfo(info)
	if err != nil {
		return err
	}
	return p.DB.Put(genChunkKey(info.ChunkHash), types.Encode(bodys))
}

// 更新本地chunk保存时间，只更新索引即可
func (p *Protocol) updateChunk(req *types.ChunkInfoMsg) error {
	m, err := p.getLocalChunkInfoMap()
	if err != nil {
		return err
	}
	mapKey := hex.EncodeToString(req.ChunkHash)
	if _, ok := m[mapKey]; ok {
		m[mapKey].Time = time.Now()
		return p.saveLocalChunkInfoMap(m)
	}

	return types2.ErrNotFound
}

// 获取本地chunk数据
//	本地不存在，返回not found
//  本地存在：
//		数据未过期：返回数据
//		数据已过期：返回数据并从数据库中删除，同时返回数据已过期的error
func (p *Protocol) getChunkBlock(hash []byte) (*types.BlockBodys, error) {
	m, err := p.getLocalChunkInfoMap()
	if err != nil {
		log.Error("getChunkBlock", "getLocalChunkInfoMap error", err)
		return nil, err
	}
	info, ok := m[hex.EncodeToString(hash)]
	if !ok {
		return nil, types2.ErrNotFound
	}

	b, err := p.DB.Get(genChunkKey(hash))
	if err != nil {
		return nil, err
	}
	var bodys types.BlockBodys
	err = types.Decode(b, &bodys)
	if err != nil {
		return nil, err
	}
	if time.Since(info.Time) > types2.ExpiredTime {
		go func() {
			delete(m, hex.EncodeToString(hash))
			err := p.saveLocalChunkInfoMap(m)
			if err != nil {
				log.Error("getChunkBlock", "saveLocalChunkInfoMap error", err)
			}
			err = p.DB.Delete(genChunkKey(hash))
			if err != nil {
				log.Error("getChunkBlock", "delete chunk error", err, "hash", hex.EncodeToString(hash))
			}
		}()
		err = types2.ErrExpired
	}

	return &bodys, err

}

func (p *Protocol) deleteChunkBlock(hash []byte) error {
	err := p.deleteLocalChunkInfo(hash)
	if err != nil {
		return err
	}
	return p.DB.Delete(genChunkKey(hash))
}

// 保存一个本地chunk hash列表，用于遍历本地数据
func (p *Protocol) addLocalChunkInfo(info *types.ChunkInfoMsg) error {
	hashMap, err := p.getLocalChunkInfoMap()
	if err != nil {
		return err
	}

	if _, ok := hashMap[hex.EncodeToString(info.ChunkHash)]; ok {
		return nil
	}

	hashMap[hex.EncodeToString(info.ChunkHash)] = &LocalChunkInfo{
		ChunkInfoMsg: info,
		Time:         time.Now(),
	}
	return p.saveLocalChunkInfoMap(hashMap)
}

func (p *Protocol) deleteLocalChunkInfo(hash []byte) error {
	hashMap, err := p.getLocalChunkInfoMap()
	if err != nil {
		return err
	}
	delete(hashMap, hex.EncodeToString(hash))
	return p.saveLocalChunkInfoMap(hashMap)
}

func (p *Protocol) getLocalChunkInfoMap() (map[string]*LocalChunkInfo, error) {

	value, err := p.DB.Get(datastore.NewKey(LocalChunkInfoKey))
	if err != nil {
		return make(map[string]*LocalChunkInfo), nil
	}

	var chunkInfoMap map[string]*LocalChunkInfo
	err = json.Unmarshal(value, &chunkInfoMap)
	if err != nil {
		return nil, err
	}

	return chunkInfoMap, nil
}

func (p *Protocol) saveLocalChunkInfoMap(m map[string]*LocalChunkInfo) error {
	value, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return p.DB.Put(datastore.NewKey(LocalChunkInfoKey), value)
}

// 适配libp2p，按路径格式生成数据的key值，便于区分多种数据类型的命名空间，以及key值合法性校验
func genChunkPath(hash []byte) string {
	return fmt.Sprintf("/%s/%s", ChunkNameSpace, hex.EncodeToString(hash))
}

func genChunkKey(hash []byte) datastore.Key {
	return datastore.NewKey(genChunkPath(hash))
}

func genDHTID(chunkHash []byte) kb.ID {
	return kb.ConvertKey(genChunkPath(chunkHash))
}
