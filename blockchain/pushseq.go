package blockchain

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/33cn/chain33/types"
)

const (
	pushMaxSeq  = 100
	pushMaxSize = 100 * 1024 * 1024
)

//pushNotify push Notify
type pushNotify struct {
	cb  chan *types.BlockSeqCB
	seq chan int64
}

//push seq data to out
type pushseq struct {
	store  *BlockStore
	cmds   map[string]pushNotify
	mu     sync.Mutex
	client *http.Client
}

func newpushseq(store *BlockStore) *pushseq {
	cmds := make(map[string]pushNotify)
	return &pushseq{store: store, cmds: cmds, client: &http.Client{}}
}

//初始化: 从数据库读出seq的数目
func (p *pushseq) init() {
	cbs, err := p.store.listSeqCB()
	if err != nil {
		chainlog.Error("listSeqCB", "err", err)
		return
	}
	for _, cb := range cbs {
		p.addTask(cb)
	}
}

//只更新本cb的seq值，每次add一个新cb时如果刷新所有的cb，会耗时很长在初始化时
func (p *pushseq) updateLastSeq(name string) {
	last, err := p.store.LoadBlockLastSequence()
	if err != nil {
		chainlog.Error("LoadBlockLastSequence", "err", err)
		return
	}

	notify := p.cmds[name]
	notify.seq <- last
}

//每个name 有一个task
func (p *pushseq) addTask(cb *types.BlockSeqCB) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if notify, ok := p.cmds[cb.Name]; ok {
		notify.cb <- cb
		if cb.URL == "" {
			chainlog.Debug("delete callback", "cb", cb)
			delete(p.cmds, cb.Name)
		}
		return
	}
	p.cmds[cb.Name] = pushNotify{
		cb:  make(chan *types.BlockSeqCB, 10),
		seq: make(chan int64, 10),
	}
	p.cmds[cb.Name].cb <- cb
	p.runTask(p.cmds[cb.Name])

	//更新最新的seq
	p.updateLastSeq(cb.Name)

	chainlog.Debug("runTask callback", "cb", cb)
}

func (p *pushseq) updateSeq(seq int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, notify := range p.cmds {
		//如果有seq, 那么先读一个出来
		select {
		case <-notify.seq:
		default:
		}
		//再写入seq（一定不会block，因为加了lock，不存在两个同时写channel的情况）
		notify.seq <- seq
	}
}

func (p *pushseq) trigeRun(run chan struct{}, sleep time.Duration) {
	if sleep > 0 {
		time.Sleep(sleep)
	}
	go func() {
		select {
		case run <- struct{}{}:
		default:
		}
	}()
}

func (p *pushseq) runTask(input pushNotify) {
	go func(in pushNotify) {
		var lastseq int64 = -1
		var maxseq int64 = -1
		var cb *types.BlockSeqCB
		var run = make(chan struct{}, 10)
		for {
			select {
			case cb = <-in.cb:
				if cb.URL == "" {
					return
				}
				p.trigeRun(run, 0)
			case maxseq = <-in.seq:
				p.trigeRun(run, 0)
			case <-run:
				if cb == nil {
					p.trigeRun(run, time.Second)
					continue
				}
				if lastseq == -1 {
					lastseq = p.store.getSeqCBLastNum([]byte(cb.Name))
				}
				if lastseq >= maxseq {
					p.trigeRun(run, 100*time.Millisecond)
					continue
				}
				seqCount := pushMaxSeq
				if lastseq+int64(seqCount) > maxseq {
					seqCount = 1
				}
				data, updateSeq, err := p.getSeqs(cb, lastseq+1, seqCount, pushMaxSize)
				if err != nil {
					chainlog.Error("getDataBySeq", "err", err, "seq", lastseq+1, "maxSeq", seqCount)
					p.trigeRun(run, 1000*time.Millisecond)
					continue
				}
				err = p.postData(cb, data, updateSeq)
				if err != nil {
					chainlog.Error("postdata", "err", err)
					//sleep 60s
					p.trigeRun(run, 60000*time.Millisecond)
					continue
				}
				//update seqid
				lastseq = lastseq + int64(seqCount)
				p.trigeRun(run, 0)
			}
		}
	}(input)
}

//seq= data.Seqs[0].Num+int64(len(data.Seqs))-1
func (p *pushseq) postData(cb *types.BlockSeqCB, postdata []byte, seq int64) (err error) {
	//post data in body
	var buf bytes.Buffer
	g := gzip.NewWriter(&buf)
	if _, err = g.Write(postdata); err != nil {
		return err
	}
	if err = g.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", cb.URL, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "ok" && string(body) != "OK" {
		chainlog.Error("postData fail", "cb.name", cb.Name, "body", string(body))
		return types.ErrPushSeqPostData
	}
	chainlog.Debug("postData success", "cb.name", cb.Name, "updateSeq", seq)
	return p.store.setSeqCBLastNum([]byte(cb.Name), seq)
}

func (p *pushseq) getBlockDataBySeq(seq int64) (*types.BlockSeq, int, error) {
	seqdata, err := p.store.GetBlockSequence(seq)
	if err != nil {
		return nil, 0, err
	}
	detail, blockSize, err := p.store.LoadBlockBySequence(seq)
	if err != nil {
		return nil, 0, err
	}
	return &types.BlockSeq{Num: seq, Seq: seqdata, Detail: detail}, blockSize, nil
}
func (p *pushseq) getSeqs(cb *types.BlockSeqCB, seq int64, seqCount, maxSize int) ([]byte, int64, error) {
	if cb.IsHeader {
		return p.getHeaderSeqs(cb.Encode, seq, seqCount, maxSize)
	}
	return p.getBlockSeqs(cb.Encode, seq, seqCount, maxSize)
}

func (p *pushseq) getBlockSeqs(encode string, seq int64, seqCount, maxSize int) ([]byte, int64, error) {
	seqs := &types.BlockSeqs{}
	totalSize := 0
	for i := 0; i < seqCount; i++ {
		seq, size, err := p.getBlockDataBySeq(seq + int64(i))
		if err != nil {
			return nil, -1, err
		}
		if totalSize == 0 || totalSize+size < maxSize {
			seqs.Seqs = append(seqs.Seqs, seq)
			totalSize += size
		} else {
			break
		}
	}
	updateSeq := seqs.Seqs[0].Num + int64(len(seqs.Seqs)) - 1

	var postdata []byte
	var err error
	if encode == "json" {
		postdata, err = types.PBToJSON(seqs)
		if err != nil {
			return nil, -1, err
		}
	} else {
		postdata = types.Encode(seqs)
	}
	return postdata, updateSeq, nil
}

func (p *pushseq) getHeaderSeqs(encode string, seq int64, seqCount, maxSize int) ([]byte, int64, error) {
	seqs := &types.HeaderSeqs{}
	totalSize := 0
	for i := 0; i < seqCount; i++ {
		seq, size, err := p.getHeaderDataBySeq(seq + int64(i))
		if err != nil {
			return nil, -1, err
		}
		if totalSize == 0 || totalSize+size < maxSize {
			seqs.Seqs = append(seqs.Seqs, seq)
			totalSize += size
		} else {
			break
		}
	}
	updateSeq := seqs.Seqs[0].Num + int64(len(seqs.Seqs)) - 1

	var postdata []byte
	var err error

	if encode == "json" {
		postdata, err = types.PBToJSON(seqs)
		if err != nil {
			return nil, -1, err
		}
	} else {
		postdata = types.Encode(seqs)
	}
	return postdata, updateSeq, nil
}

func (p *pushseq) getHeaderDataBySeq(seq int64) (*types.HeaderSeq, int, error) {
	seqdata, err := p.store.GetBlockSequence(seq)
	if err != nil {
		return nil, 0, err
	}
	header, err := p.store.GetBlockHeaderByHash(seqdata.Hash)
	if err != nil {
		return nil, 0, err
	}
	return &types.HeaderSeq{Num: seq, Seq: seqdata, Header: header}, header.Size(), nil
}
