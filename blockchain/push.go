package blockchain

import (
	"bytes"
	"compress/gzip"
	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	notRunning               = int32(1)
	running                  = int32(2)
	PushBlock                = int32(0)
	PushTxReceipt            = int32(1)
	waitNewBlock             = 5 * time.Second
	pushMaxSeq               = 10
	pushMaxSize              = 1 * 1024 * 1024
	maxPushSubscriber        = int(100)
	subscribeStatusActive    = int32(1)
	subscribeStatusNotActive = int32(2)
	postFailSleepSecond      = int32(60)
)

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

type SequenceStore interface {
	LoadBlockLastSequence() (int64, error)
	// seqUpdateChan -> block sequence
	GetBlockSequence(seq int64) (*types.BlockSequence, error)
	// hash -> block header
	GetBlockHeaderByHash(hash []byte) (*types.Header, error)
	// seqUpdateChan -> block, size
	LoadBlockBySequence(seq int64) (*types.BlockDetail, int, error)
	// get last header
	LastHeader() *types.Header
	// hash -> seqUpdateChan
	GetSequenceByHash(hash []byte) (int64, error)
}

type PostService interface {
	PostData(subscribe *types.PushSubscribeReq, postdata []byte, seq int64) (err error)
}

//当前的实现是为每个订阅者单独启动一个协程goroute，然后单独为每个subscriber分别过滤指定类型的交易，
//进行归类，这种方式集成了区块推送方式的处理机制，但是对于订阅者数量大的情况，势必会浪费cpu的开销，
//数据库的读取开销不会额外增加明星，因为会有cach
//TODO：后续需要考虑将区块推送和交易执行回执推送进行重构，提高并行推送效率
//pushNotify push Notify
type pushNotify struct {
	subscribe           *types.PushSubscribeReq
	seqUpdateChan       chan int64
	status              int32
	postFailSleepSecond int32
}

type Push struct {
	store               CommonStore
	sequenceStore       SequenceStore
	tasks               map[string]*pushNotify
	mu                  sync.Mutex
	postService         PostService
	cfg                 *types.Chain33Config
	postFailSleepSecond int32
}

type PushClient struct {
	client *http.Client
}

type PushType int32

func (pushType PushType) string() string {
	return []string{"PushBlock", "PushTxReceipt", "NotSupported"}[pushType]
}

func (pushClient *PushClient) PostData(subscribe *types.PushSubscribeReq, postdata []byte, seq int64) (err error) {
	//post data in body
	chainlog.Info("postData begin", "seq", seq, "subscribe name", subscribe.Name)
	var buf bytes.Buffer
	g := gzip.NewWriter(&buf)
	if _, err = g.Write(postdata); err != nil {
		return err
	}
	if err = g.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", subscribe.URL, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := pushClient.client.Do(req)
	if err != nil {
		chainlog.Info("postData", "Do err", err)
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return err
	}
	if string(body) != "ok" && string(body) != "OK" {
		chainlog.Error("postData fail", "name:", subscribe.Name, "URL", subscribe.URL,
			"Contract:", subscribe.Contract, "body", string(body))
		_ = resp.Body.Close()
		return types.ErrPushSeqPostData
	}
	chainlog.Debug("postData success", "name", subscribe.Name, "URL", subscribe.URL,
		"Contract:", subscribe.Contract, "updateSeq", seq)
	return resp.Body.Close()
}

//ProcAddBlockSeqCB 添加seq callback
func (chain *BlockChain) procSubscribePush(subscribe *types.PushSubscribeReq) error {
	if !chain.enablePushSubscribe {
		chainlog.Error("Push is not enabled for subscribed")
		return types.ErrPushNotSupport
	}

	if !chain.isRecordBlockSequence {
		chainlog.Error("procSubscribePush not support sequence")
		return types.ErrRecordBlockSequence
	}

	if subscribe == nil {
		chainlog.Error("procSubscribePush para is null")
		return types.ErrInvalidParam
	}

	if chain.client.GetConfig().IsEnable("reduceLocaldb") && subscribe.Type == PushTxReceipt {
		chainlog.Error("Tx receipts are reduced on this node")
		return types.ErrTxReceiptReduced
	}

	return chain.push.addSubscriber(subscribe)
}

//ProcListPush 列出所有已经设置的推送订阅
func (chain *BlockChain) ProcListPush() (*types.PushSubscribes, error) {
	if !chain.isRecordBlockSequence {
		return nil, types.ErrRecordBlockSequence
	}
	if !chain.enablePushSubscribe {
		return nil, types.ErrPushNotSupport
	}

	values, err := chain.push.store.List(pushPrefix)
	if err != nil {
		return nil, err
	}
	var listSeqCBs types.PushSubscribes
	for _, value := range values {
		var onePush types.PushSubscribeReq
		err := types.Decode(value, &onePush)
		if err != nil {
			return nil, err
		}
		listSeqCBs.Pushes = append(listSeqCBs.Pushes, &onePush)
	}
	return &listSeqCBs, nil
}

// GetLastPushSeq Seq的合法值从0开始的，所以没有获取到或者获取失败都应该返回-1
func (chain *BlockChain) ProcGetLastPushSeq(name string) int64 {
	lastSeqbytes, err := chain.push.store.GetKey(calcLastPushSeqNumKey(name))
	if lastSeqbytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("getSeqCBLastNum", "error", err)
		}
		return -1
	}
	n, err := decodeHeight(lastSeqbytes)
	if err != nil {
		return -1
	}
	storeLog.Error("getSeqCBLastNum", "name", name, "num", n)

	return n
}

func newpush(commonStore CommonStore, seqStore SequenceStore, cfg *types.Chain33Config) *Push {
	tasks := make(map[string]*pushNotify)

	pushClient := &PushClient{
		client: &http.Client{Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}},
	}
	service := &Push{store: commonStore,
		sequenceStore:       seqStore,
		tasks:               tasks,
		postService:         pushClient,
		cfg:                 cfg,
		postFailSleepSecond: postFailSleepSecond,
	}
	service.init()

	return service
}

//初始化: 从数据库读出seq的数目
func (push *Push) init() {
	var subscribes []*types.PushSubscribeReq
	values, err := push.store.List(pushPrefix)
	if err != nil && err != dbm.ErrNotFoundInDb {
		chainlog.Error("Push init", "err", err)
		return
	}
	if 0 == len(values) {
		return
	}
	for _, value := range values {
		var pushWithStatus types.PushWithStatus
		err := types.Decode(value, &pushWithStatus)
		if err != nil {
			chainlog.Error("Push init", "Failed to decode subscribe due to err:", err)
			return
		}
		if pushWithStatus.Status == subscribeStatusActive {
			subscribes = append(subscribes, pushWithStatus.Push)
		}

	}
	for _, subscribe := range subscribes {
		push.addTask(subscribe)
	}
}

func (push *Push) addSubscriber(subscribe *types.PushSubscribeReq) error {
	if subscribe == nil {
		chainlog.Error("addSubscriber input para is null")
		return types.ErrInvalidParam
	}

	//如果需要配置起始的块的信息，则为了保持一致性，三项缺一不可
	if subscribe.LastBlockHash != "" || subscribe.LastSequence != 0 || subscribe.LastHeight != 0 {
		if subscribe.LastBlockHash == "" || subscribe.LastSequence == 0 || subscribe.LastHeight == 0 {
			chainlog.Error("addSubscriber ErrInvalidParam", "seqUpdateChan", subscribe.LastSequence, "height", subscribe.LastHeight, "hash", subscribe.LastBlockHash)
			return types.ErrInvalidParam
		}
	}

	//如果该用户已经注册了订阅请求，则只是确认是否需用重新启动，否则就直接返回
	if exist, subscribeInDB := push.hasSubscriberExist(subscribe); exist {
		if subscribeInDB.URL != subscribe.URL || subscribeInDB.Type != subscribe.Type {
			return types.ErrNotAllowModifyPush
		}
		//使用保存在数据库中的push配置，而不是最新的配置信息
		return push.check2ResumePush(subscribeInDB)
	}

	if len(push.tasks) >= int(maxPushSubscriber) {
		chainlog.Error("addSubscriber too many push subscriber")
		return types.ErrTooManySeqCB
	}

	//处理需要从指定高度开始推送的订阅请求
	if subscribe.LastSequence > 0 {
		sequence, err := push.sequenceStore.GetBlockSequence(subscribe.LastSequence)
		if err != nil {
			chainlog.Error("addSubscriber continue-seqUpdateChan-push", "load-1", err)
			return err
		}

		// 注册点，在节点上存在
		// 同一高度，不一定同一个hash，有分叉的可能；但同一个hash必定同一个高度
		reloadHash := common.ToHex(sequence.Hash)
		if subscribe.LastBlockHash == reloadHash {
			// 先填入last seqUpdateChan， 而不是从0开始
			err = push.setLastPushSeq(subscribe.Name, subscribe.LastSequence)
			if err != nil {
				chainlog.Error("addSubscriber", "setLastPushSeq", err)
				return err
			}
			return push.persisAndStart(subscribe)
		}
	}

	return push.persisAndStart(subscribe)
}

func (push *Push) hasSubscriberExist(subscribe *types.PushSubscribeReq) (bool, *types.PushSubscribeReq) {
	value, err := push.store.GetKey(calcPushKey(subscribe.Name))
	if err == nil {
		var subscribeExist types.PushSubscribeReq
		err = types.Decode(value, &subscribeExist)
		return err == nil, &subscribeExist
	}
	return false, nil
}

func (push *Push) subscriberCount() int64 {
	return push.store.PrefixCount(pushPrefix)
}

//向数据库添加交易回执订阅信息
func (push *Push) persisAndStart(subscribe *types.PushSubscribeReq) error {
	if len(subscribe.Name) > 128 || len(subscribe.URL) > 1024 || len(subscribe.URL) == 0 {
		storeLog.Error("Invalid para to persisAndStart due to wrong length", "len(subscribe.Name)=", len(subscribe.Name),
			"len(subscribe.URL)=", len(subscribe.URL), "len(subscribe.Contract)=", len(subscribe.Contract))
		return types.ErrInvalidParam
	}
	key := calcPushKey(subscribe.Name)
	storeLog.Info("persisAndStart", "key", string(key), "subscribe", subscribe)
	push.addTask(subscribe)

	pushWithStatus := &types.PushWithStatus{
		Push:   subscribe,
		Status: subscribeStatusActive,
	}

	return push.store.SetSync(key, types.Encode(pushWithStatus))
}

func (push *Push) check2ResumePush(subscribe *types.PushSubscribeReq) error {
	if len(subscribe.Name) > 128 || len(subscribe.URL) > 1024 || len(subscribe.Contract) > 128 {
		storeLog.Error("Invalid para to persisAndStart due to wrong length", "len(subscribe.Name)=", len(subscribe.Name),
			"len(subscribe.URL)=", len(subscribe.URL), "len(subscribe.Contract)=", len(subscribe.Contract))
		return types.ErrInvalidParam
	}
	push.mu.Lock()
	defer push.mu.Unlock()

	keyStr := string(calcPushKey(subscribe.Name))
	storeLog.Info("check2ResumePush", "key", keyStr, "subscribe", subscribe)

	notify := push.tasks[keyStr]
	//有可能因为连续发送失败已经导致将其从推送任务中删除了
	if nil == notify {
		push.tasks[keyStr] = &pushNotify{
			subscribe:     subscribe,
			seqUpdateChan: make(chan int64, 10),
			status:        notRunning,
		}

		push.runTask(push.tasks[keyStr])
		return nil
	}

	if running == atomic.LoadInt32(&notify.status) {
		storeLog.Info("Is already in state:running", "postFailSleepSecond", atomic.LoadInt32(&notify.postFailSleepSecond))
		atomic.CompareAndSwapInt32(&notify.postFailSleepSecond, notify.postFailSleepSecond, 0)
		return nil
	}

	push.runTask(push.tasks[keyStr])
	return nil
}

//只更新本cb的seq值，每次add一个新cb时如果刷新所有的cb，会耗时很长在初始化时
func (push *Push) updateLastSeq(nameContract string) {
	last, err := push.sequenceStore.LoadBlockLastSequence()
	if err != nil {
		chainlog.Error("LoadBlockLastSequence", "err", err)
		return
	}

	notify := push.tasks[nameContract]
	notify.seqUpdateChan <- last
	chainlog.Debug("updateLastSeq", "last", last, "notify.seqUpdateChan", len(notify.seqUpdateChan))
}

// addTask 每个name 有一个task, 通知新增推送
func (push *Push) addTask(subscribe *types.PushSubscribeReq) {
	push.mu.Lock()
	defer push.mu.Unlock()
	keyStr := string(calcPushKey(subscribe.Name))
	push.tasks[keyStr] = &pushNotify{
		subscribe:     subscribe,
		seqUpdateChan: make(chan int64, 10),
		status:        notRunning,
	}

	push.runTask(push.tasks[keyStr])
	//更新最新的seq
	push.updateLastSeq(keyStr)
	chainlog.Debug("runTask to push tx receipt", "subscribe", subscribe)
}

// UpdateSeq sequence 更新通知
func (push *Push) UpdateSeq(seq int64) {
	push.mu.Lock()
	defer push.mu.Unlock()
	for _, notify := range push.tasks {
		//再写入seq（一定不会block，因为加了lock，不存在两个同时写channel的情况）
		if len(notify.seqUpdateChan) < 10 {
			chainlog.Info("new block Update Seq notified", "subscribe", notify.subscribe.Name, "current sequence", seq, "length", len(notify.seqUpdateChan))
			notify.seqUpdateChan <- seq
		}
		chainlog.Info("new block UpdateSeq", "subscribe", notify.subscribe.Name, "current sequence", seq, "length", len(notify.seqUpdateChan))
	}
}

func (push *Push) trigeRun(run chan struct{}, sleep time.Duration, name string) {
	chainlog.Info("trigeRun", name, "name", "sleep", sleep, "run len", len(run))
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

func (push *Push) runTask(input *pushNotify) {
	go func(in *pushNotify) {
		var lastProcessedseq int64 = -1
		var lastesBlockSeq int64 = -1
		var run = make(chan struct{}, 10)
		var continueFailCount int32 = 0

		atomic.CompareAndSwapInt32(&input.postFailSleepSecond, input.postFailSleepSecond, 0)
		subscribe := in.subscribe
		push.trigeRun(run, 0, in.subscribe.Name)
		lastProcessedseq = push.getLastPushSeq(subscribe)
		atomic.StoreInt32(&in.status, running)
		chainlog.Debug("runTask to push tx receipt", "subscribe", subscribe, "in.seqUpdateChan", in.seqUpdateChan)
		for {
			select {
			case lastesBlockSeq = <-in.seqUpdateChan:
				chainlog.Debug("Get new block Seq", "name", in.subscribe.Name, "lastesBlockSeq", lastesBlockSeq, "len", len(in.seqUpdateChan), "in", in.seqUpdateChan)
				cnt := len(in.seqUpdateChan)
				for i := 0; i < cnt; i++ {
					lastesBlockSeq = <-in.seqUpdateChan
					chainlog.Debug("Get more new block Seq ", "name", in.subscribe.Name, "lastesBlockSeq", lastesBlockSeq, "len", len(in.seqUpdateChan), "in", in.seqUpdateChan)
				}

				chainlog.Debug("runTask to push tx receipt in seqUpdateChan", "name", in.subscribe.Name, "lastesBlockSeq", lastesBlockSeq, "len", len(in.seqUpdateChan), "in", in.seqUpdateChan)
				push.trigeRun(run, 0, in.subscribe.Name)
			case <-run:
				atomic.AddInt32(&input.postFailSleepSecond, -1)
				if atomic.LoadInt32(&input.postFailSleepSecond) > 0 {
					//每次sleep 1s，重复被激活
					push.trigeRun(run, 1000*time.Millisecond, in.subscribe.Name)
					chainlog.Debug("Another 1 second sleep for post fail", "postFailSleepSecond", input.postFailSleepSecond, "name", in.subscribe.Name)
					continue
				}
				atomic.CompareAndSwapInt32(&input.postFailSleepSecond, input.postFailSleepSecond, 0)
				
				//没有更新的区块，则不进行处理，同时等待一定的时间
				if lastProcessedseq >= lastesBlockSeq {
					chainlog.Debug("lastProcessedseq >= lastesBlockSeq", "lastProcessedseq", lastProcessedseq, "lastesBlockSeq", lastesBlockSeq)
					push.trigeRun(run, waitNewBlock, in.subscribe.Name)
					continue
				}
				//确定一次推送的数量，如果需要更新的数量少于门限值，则一次只推送一个区块的交易数据
				seqCount := pushMaxSeq
				if lastProcessedseq+int64(seqCount) > lastesBlockSeq {
					seqCount = 1
				}
				data, updateSeq, err := push.getPushData(subscribe, lastProcessedseq+1, seqCount, pushMaxSize)
				if err != nil {
					chainlog.Error("getPushData", "err", err, "seqCurrent", lastProcessedseq+1, "maxSeq", seqCount,
						"Name", subscribe.Name, "pushType:", PushType(subscribe.Type).string())
					push.trigeRun(run, 1000*time.Millisecond, in.subscribe.Name)
					continue
				}

				if data != nil {
					err = push.postService.PostData(subscribe, data, updateSeq)
					if err != nil {
						continueFailCount++
						chainlog.Error("postdata failed", "err", err, "lastProcessedseq", lastProcessedseq,
							"Name", subscribe.Name, "pushType:", PushType(subscribe.Type).string(), "continueFailCount", continueFailCount)
						if continueFailCount >= 3 {
							atomic.StoreInt32(&in.status, notRunning)
							chainlog.Error("postdata failed exceed 3 times", "Name", subscribe.Name, "in.status", atomic.LoadInt32(&in.status))

							pushWithStatus := &types.PushWithStatus{
								Push:   subscribe,
								Status: subscribeStatusNotActive,
							}

							key := calcPushKey(subscribe.Name)
							push.mu.Lock()
							delete(push.tasks, string(key))
							push.mu.Unlock()
							_ = push.store.SetSync(key, types.Encode(pushWithStatus))
							return
						}
						//sleep 60s，每次1s，总计60次，在每次结束时，等待接收方重新进行请求推送
						push.trigeRun(run, 1000*time.Millisecond, in.subscribe.Name)
						atomic.CompareAndSwapInt32(&input.postFailSleepSecond, input.postFailSleepSecond, push.postFailSleepSecond)
						continue
					}
					_ = push.setLastPushSeq(subscribe.Name, updateSeq)
				}
				continueFailCount = 0
				//update seqid
				lastProcessedseq = updateSeq
				//push.trigeRun(run, 0)
			}
		}
	}(input)
}

func (push *Push) getPushData(subscribe *types.PushSubscribeReq, startSeq int64, seqCount, maxSize int) ([]byte, int64, error) {
	if subscribe.Type == PushBlock {
		return push.getBlocks(subscribe, startSeq, seqCount, maxSize)
	}
	return push.getTxReceipts(subscribe, startSeq, seqCount, maxSize)
}

func (push *Push) getTxReceipts(subscribe *types.PushSubscribeReq, startSeq int64, seqCount, maxSize int) ([]byte, int64, error) {
	txReceipts := &types.TxReceipts4Subscribe{}
	totalSize := 0
	actualIterCount := 0
	for i := startSeq; i < startSeq+int64(seqCount); i++ {
		chainlog.Info("getTxReceipts", "startSeq:", i)
		seqdata, err := push.sequenceStore.GetBlockSequence(i)
		if err != nil {
			return nil, -1, err
		}
		detail, _, err := push.sequenceStore.LoadBlockBySequence(i)
		if err != nil {
			return nil, -1, err
		}

		txReceiptsPerBlk := &types.TxReceipts4SubscribePerBlk{}
		chainlog.Info("getTxReceipts", "height:", detail.Block.Height, "tx numbers:", len(detail.Block.Txs), "Receipts numbers:", len(detail.Receipts))
		for txIndex, tx := range detail.Block.Txs {
			if subscribe.Contract[string(tx.Execer)] {
				chainlog.Info("getTxReceipts", "txIndex:", txIndex)
				txReceiptsPerBlk.Tx = append(txReceiptsPerBlk.Tx, tx)
				txReceiptsPerBlk.ReceiptData = append(txReceiptsPerBlk.ReceiptData, detail.Receipts[txIndex])
				//txReceiptsPerBlk.KV = append(txReceiptsPerBlk.KV, detail.KV[txIndex])
			}
		}
		if len(txReceiptsPerBlk.Tx) > 0 {
			txReceiptsPerBlk.Height = detail.Block.Height
			txReceiptsPerBlk.BlockHash = detail.Block.Hash(push.cfg)
			txReceiptsPerBlk.ParentHash = detail.Block.ParentHash
			txReceiptsPerBlk.PreviousHash = []byte{}
			txReceiptsPerBlk.AddDelType = int32(seqdata.Type)
			txReceiptsPerBlk.SeqNum = i
		}
		size := types.Size(txReceiptsPerBlk)
		if len(txReceiptsPerBlk.Tx) > 0 && totalSize+size < maxSize {
			txReceipts.TxReceipts = append(txReceipts.TxReceipts, txReceiptsPerBlk)
			totalSize += size
			chainlog.Debug("get Tx Receipts subscribed for pushing", "Name", subscribe.Name, "contract:", subscribe.Contract,
				"height=", txReceiptsPerBlk.Height)
		} else if totalSize+size > maxSize {
			break
		}
		actualIterCount++
	}

	updateSeq := startSeq + int64(actualIterCount) - 1
	if len(txReceipts.TxReceipts) == 0 {
		return nil, updateSeq, nil
	}
	chainlog.Info("getTxReceipts", "updateSeq", updateSeq, "actualIterCount", actualIterCount)

	var postdata []byte
	var err error
	if subscribe.Encode == "json" {
		postdata, err = types.PBToJSON(txReceipts)
		if err != nil {
			return nil, -1, err
		}
	} else {
		postdata = types.Encode(txReceipts)
	}

	return postdata, updateSeq, nil
}

func (push *Push) getBlocks(subscribe *types.PushSubscribeReq, seq int64, seqCount, maxSize int) ([]byte, int64, error) {
	if subscribe.IsHeader {
		return push.getHeaderSeqs(subscribe.Encode, seq, seqCount, maxSize)
	}
	return push.getBlockSeqs(subscribe.Encode, seq, seqCount, maxSize)
}

func (push *Push) getBlockDataBySeq(seq int64) (*types.BlockSeq, int, error) {
	seqdata, err := push.sequenceStore.GetBlockSequence(seq)
	if err != nil {
		return nil, 0, err
	}
	detail, blockSize, err := push.sequenceStore.LoadBlockBySequence(seq)
	if err != nil {
		return nil, 0, err
	}
	return &types.BlockSeq{Num: seq, Seq: seqdata, Detail: detail}, blockSize, nil
}

func (push *Push) getBlockSeqs(encode string, seq int64, seqCount, maxSize int) ([]byte, int64, error) {
	seqs := &types.BlockSeqs{}
	totalSize := 0
	for i := 0; i < seqCount; i++ {
		seq, size, err := push.getBlockDataBySeq(seq + int64(i))
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

func (push *Push) getHeaderSeqs(encode string, seq int64, seqCount, maxSize int) ([]byte, int64, error) {
	seqs := &types.HeaderSeqs{}
	totalSize := 0
	for i := 0; i < seqCount; i++ {
		seq, size, err := push.getHeaderDataBySeq(seq + int64(i))
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

func (push *Push) getHeaderDataBySeq(seq int64) (*types.HeaderSeq, int, error) {
	seqdata, err := push.sequenceStore.GetBlockSequence(seq)
	if err != nil {
		return nil, 0, err
	}
	header, err := push.sequenceStore.GetBlockHeaderByHash(seqdata.Hash)
	if err != nil {
		return nil, 0, err
	}
	return &types.HeaderSeq{Num: seq, Seq: seqdata, Header: header}, header.Size(), nil
}

// GetLastPushSeq Seq的合法值从0开始的，所以没有获取到或者获取失败都应该返回-1
func (push *Push) getLastPushSeq(subscribe *types.PushSubscribeReq) int64 {
	seqbytes, err := push.store.GetKey(calcLastPushSeqNumKey(subscribe.Name))
	if seqbytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("getLastPushSeq", "error", err)
		}
		return -1
	}
	n, err := decodeHeight(seqbytes)
	if err != nil {
		return -1
	}
	storeLog.Info("getLastPushSeq", "name", subscribe.Name,
		"Contract:", subscribe.Contract, "num", n)

	return n
}

func (push *Push) setLastPushSeq(name string, num int64) error {
	return push.store.SetSync(calcLastPushSeqNumKey(name), types.Encode(&types.Int64{Data: num}))
}
