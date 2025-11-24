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
)

// LocalChunkInfo wraps local chunk key with time.
type LocalChunkInfo struct {
	*types.ChunkInfoMsg
	Time time.Time
}

// 保存chunk到本地p2pStore，同时更新本地chunk列表
func (p *Protocol) addChunkBlock(info *types.ChunkInfoMsg, bodys *types.BlockBodys) error {
	if int64(len(bodys.Items)) != info.End-info.Start+1 {
		return types2.ErrLength
	}
	p.addLocalChunkInfo(info)
	for i := info.Start; i <= info.End; i++ {
		key := genChunkDBKey(i)
		if err := p.DB.Set(key, types.Encode(bodys.Items[i-info.Start])); err != nil {
			return err
		}
	}
	return nil
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
	if _, err := p.DB.Get(genChunkDBKey(req.Start)); err == nil {
		p.localChunkInfo[mapKey] = LocalChunkInfo{
			ChunkInfoMsg: req,
			Time:         time.Now(),
		}
		return nil
	}

	return types2.ErrNotFound
}

func (p *Protocol) deleteChunkBlock(msg *types.ChunkInfoMsg) error {
	if exist := p.deleteLocalChunkInfo(msg.ChunkHash); !exist {
		return nil
	}
	batch := p.DB.NewBatch(true)
	start, end := genChunkDBKey(msg.Start), genChunkDBKey(msg.End+1)
	it := p.DB.Iterator(start, end, false)
	defer it.Close()
	for it.Next(); it.Valid(); it.Next() {
		batch.Delete(it.Key())
	}
	_ = batch.Write()
	_ = p.DB.CompactRange(start, end)
	return nil
}

// 获取本地chunk数据
func (p *Protocol) loadChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, error) {

	// TODO: 处理特殊情况
	// 新节点从其他节点拷贝了p2pstore
	//if _, ok := p.getChunkInfoByHash(req.ChunkHash); !ok {
	//	return nil, types2.ErrNotFound
	//}
	var bodys []*types.BlockBody
	it := p.DB.Iterator(genChunkDBKey(req.Start), genChunkDBKey(req.End+1), false)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
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
func (p *Protocol) addLocalChunkInfo(info *types.ChunkInfoMsg) {
	p.localChunkInfoMutex.Lock()
	defer p.localChunkInfoMutex.Unlock()

	p.localChunkInfo[hex.EncodeToString(info.ChunkHash)] = LocalChunkInfo{
		ChunkInfoMsg: info,
		Time:         time.Now(),
	}
}

func (p *Protocol) deleteLocalChunkInfo(hash []byte) bool {
	p.localChunkInfoMutex.Lock()
	defer p.localChunkInfoMutex.Unlock()
	if _, exist := p.localChunkInfo[hex.EncodeToString(hash)]; !exist {
		return false
	}
	delete(p.localChunkInfo, hex.EncodeToString(hash))
	return true
}

func (p *Protocol) initLocalChunkInfoMap() {
	_ = p.DB.DeleteSync([]byte(LocalChunkInfoKey))
	p.localChunkInfo = make(map[string]LocalChunkInfo)
	start := time.Now()
	records, err := p.getChunkRecordFromBlockchain(&types.ReqChunkRecords{Start: 0, End: 0})
	if err != nil {
		return
	}
	if records == nil || len(records.Infos) != 1 {
		panic("invalid record")
	}
	chunkLen := records.Infos[0].End - records.Infos[0].Start + 1
	for i := int64(0); ; i++ {
		records, err := p.getChunkRecordFromBlockchain(&types.ReqChunkRecords{Start: i, End: i})
		if err != nil {
			break
		}
		info := records.Infos[0]
		key := genChunkDBKey(i * chunkLen)
		if _, err := p.DB.Get(key); err == nil {
			p.localChunkInfoMutex.Lock()
			p.localChunkInfo[hex.EncodeToString(info.ChunkHash)] = LocalChunkInfo{
				ChunkInfoMsg: &types.ChunkInfoMsg{
					ChunkHash: info.ChunkHash,
					Start:     info.Start,
					End:       info.End,
				},
				Time: start,
			}
			p.localChunkInfoMutex.Unlock()
		}
	}
	log.Info("initLocalChunkInfoMap", "time cost", time.Since(start), "len", len(p.localChunkInfo))
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
