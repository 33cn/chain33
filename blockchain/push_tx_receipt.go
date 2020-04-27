package blockchain

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

const (
	NotRunning = iota + 99
	Running
)

//当前的实现是为每个订阅者单独启动一个协程goroute，然后单独为每个subscriber分别过滤指定类型的交易，
//进行归类，这种方式集成了区块推送方式的处理机制，但是对于订阅者数量大的情况，势必会浪费cpu的开销，
//数据库的读取开销不会额外增加明星，因为会有cach
//TODO：后续需要考虑将区块推送和交易执行回执推送进行重构，提高并行推送效率
//pushNotify push Notify
type pushTxReceiptNotify struct {
	subscribe *types.SubscribeTxReceipt
	seq       chan int64
	status    int32
}

type PushTxReceiptService struct {
	store         CommonStore
	sequenceStore SequenceStore
	tasks         map[string]*pushTxReceiptNotify
	mu            sync.Mutex
	client        *http.Client
	cfg           *types.Chain33Config
}

//ProcAddBlockSeqCB 添加seq callback
func (chain *BlockChain) procSubscribeTxReceipt(subscribe *types.SubscribeTxReceipt) error {
	if subscribe == nil {
		chainlog.Error("procSubscribeTxReceipt para is null")
		return types.ErrInvalidParam
	}

	return chain.pushTxReceipt.addTxReceiptSubscriber(subscribe)
}

func (push *PushTxReceiptService) addTxReceiptSubscriber(subscribe *types.SubscribeTxReceipt) error {
	if subscribe == nil {
		chainlog.Error("addTxReceiptSubscriber input para is null")
		return types.ErrInvalidParam
	}

	//如果需要配置起始的块的信息，则为了保持一致性，三项缺一不可
	if subscribe.LastBlockHash != "" || subscribe.LastSequence != 0 || subscribe.LastHeight != 0 {
		if subscribe.LastBlockHash == "" || subscribe.LastSequence == 0 || subscribe.LastHeight == 0 {
			chainlog.Error("addTxReceiptSubscriber ErrInvalidParam", "seq", subscribe.LastSequence, "height", subscribe.LastHeight, "hash", subscribe.LastBlockHash)
			return types.ErrInvalidParam
		}
	}

	//如果该用户已经注册了订阅请求，则只是确认是否需用重新启动，否则就直接返回
	if push.hasSubscriberExist(subscribe) {
		return push.check2ResumePush(subscribe)
	}

	if push.subscriberCount() >= MaxTxReceiptSubscriber {
		chainlog.Error("ProcAddBlockSeqCB too many seq callback")
		return types.ErrTooManySeqCB
	}
	if err := push.addTxSubscriber(subscribe); nil != err {
		return err
	}

	return nil
}

//目前只是最简单地确认需要增加的交易执行回执是否有完全重复，只要不是就ok
//通过name-contract的方式管理一个用户可以订阅多个交易回执的功能
//TODO：后续需要增加删除指定交易类型回执的订阅功能
func (push *PushTxReceiptService) hasSubscriberExist(subscribe *types.SubscribeTxReceipt) bool {
	value, err := push.store.GetKey(calcTxReceiptKey(subscribe.Name, subscribe.Contract))
	if err == nil {
		var subscribeExist types.SubscribeTxReceipt
		err = types.Decode(value, &subscribeExist)
		return err == nil
	}
	return false
}

func (push *PushTxReceiptService) subscriberCount() int64 {
	return push.store.PrefixCount([]byte(txReceiptPrefix))
}

//向数据库添加交易回执订阅信息
func (push *PushTxReceiptService) addTxSubscriber(subscribe *types.SubscribeTxReceipt) error {
	if len(subscribe.Name) > 128 || len(subscribe.URL) > 1024 || len(subscribe.Contract) > 128 {
		storeLog.Error("Invalid para to addTxSubscriber due to wrong length", "len(subscribe.Name)=", len(subscribe.Name),
			"len(subscribe.URL)=", len(subscribe.URL), "len(subscribe.Contract)=", len(subscribe.Contract))
		return types.ErrInvalidParam
	}
	key := calcTxReceiptKey(subscribe.Name, subscribe.Contract)
	storeLog.Info("addTxSubscriber", "key", string(key), "subscribe", subscribe)
	push.addTask(subscribe)
	return push.store.SetSync(key, types.Encode(subscribe))
}

func (push *PushTxReceiptService) check2ResumePush(subscribe *types.SubscribeTxReceipt) error {
	if len(subscribe.Name) > 128 || len(subscribe.URL) > 1024 || len(subscribe.Contract) > 128 {
		storeLog.Error("Invalid para to addTxSubscriber due to wrong length", "len(subscribe.Name)=", len(subscribe.Name),
			"len(subscribe.URL)=", len(subscribe.URL), "len(subscribe.Contract)=", len(subscribe.Contract))
		return types.ErrInvalidParam
	}
	push.mu.Lock()
	defer push.mu.Unlock()

	key := calcTxReceiptKey(subscribe.Name, subscribe.Contract)
	storeLog.Info("check2ResumePush", "key", string(key), "subscribe", subscribe)
	//push.addTask(subscribe)

	keyStr := string(calcTxReceiptKey(subscribe.Name, subscribe.Contract))
	notify := push.tasks[keyStr]
	//TODO:支持直接从运行状态将其删除注册
	if Running == notify.status {
		storeLog.Info("Is already in state:Running")
		return nil
	}

	if subscribe.URL == "" {
		chainlog.Debug("delete Tx Receipt Subscriber", "subscribe", subscribe)
		delete(push.tasks, keyStr)
		_ = push.store.SetSync([]byte(keyStr), nil)
		return nil
	}

	push.runTask(push.tasks[keyStr])
	//更新最新的seq
	//push.updateLastSeq(keyStr)
	return nil
}

///////////////////////////////////////////////
///////////////////////////////////////////////
///////////////////////////////////////////////
func newpushTxReceiptService(commonStore CommonStore, seqStore SequenceStore, cfg *types.Chain33Config) *PushTxReceiptService {
	tasks := make(map[string]*pushTxReceiptNotify)
	service := &PushTxReceiptService{store: commonStore,
		sequenceStore: seqStore,
		tasks:         tasks,
		client: &http.Client{Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}},
		cfg: cfg,
	}
	service.init()

	return service
}

//初始化: 从数据库读出seq的数目
func (push *PushTxReceiptService) init() {
	var subscribes []*types.SubscribeTxReceipt
	values, err := push.store.List([]byte(txReceiptPrefix))
	if err != dbm.ErrNotFoundInDb {
		chainlog.Error("PushTxReceiptService init", "err", err)
		return
	}
	if 0 == len(values) {
		return
	}
	for _, value := range values {
		var subscribe types.SubscribeTxReceipt
		err := types.Decode(value, &subscribe)
		if err != nil {
			chainlog.Error("PushTxReceiptService init", "Failed to decode subscribe due to err:", err)
			return
		}
		subscribes = append(subscribes, &subscribe)
	}
	for _, subscribe := range subscribes {
		push.addTask(subscribe)
	}
}

//只更新本cb的seq值，每次add一个新cb时如果刷新所有的cb，会耗时很长在初始化时
func (push *PushTxReceiptService) updateLastSeq(nameContract string) {
	last, err := push.sequenceStore.LoadBlockLastSequence()
	if err != nil {
		chainlog.Error("LoadBlockLastSequence", "err", err)
		return
	}

	notify := push.tasks[nameContract]
	notify.seq <- last
	chainlog.Debug("updateLastSeq", "last", last, "notify.seq", len(notify.seq))
}

// addTask 每个name 有一个task, 通知新增推送
func (push *PushTxReceiptService) addTask(subscribe *types.SubscribeTxReceipt) {
	push.mu.Lock()
	defer push.mu.Unlock()
	keyStr := string(calcTxReceiptKey(subscribe.Name, subscribe.Contract))
	if _, ok := push.tasks[keyStr]; ok {
		if subscribe.URL == "" {
			chainlog.Debug("delete Tx Receipt Subscriber", "subscribe", subscribe)
			delete(push.tasks, keyStr)
			_ = push.store.SetSync([]byte(keyStr), nil)
		}
		return
	}
	push.tasks[keyStr] = &pushTxReceiptNotify{
		subscribe: subscribe,
		seq:       make(chan int64, 10),
		status:    NotRunning,
	}

	push.runTask(push.tasks[keyStr])
	//更新最新的seq
	push.updateLastSeq(keyStr)
	chainlog.Debug("runTask to push tx receipt", "subscribe", subscribe)
}

// UpdateSeq sequence 更新通知
func (push *PushTxReceiptService) UpdateSeq(seq int64) {
	push.mu.Lock()
	defer push.mu.Unlock()
	for _, notify := range push.tasks {
		////如果有seq, 那么先读一个出来
		//select {
		//case ss := <-notify.seq:
		//	chainlog.Info("UpdateSeq", "ss", ss,"length", len(notify.seq))
		//default:
		//}
		//再写入seq（一定不会block，因为加了lock，不存在两个同时写channel的情况）
		if len(notify.seq) < 10 {
			notify.seq <- seq
			chainlog.Info("UpdateSeq", "insert seq", seq, "length", len(notify.seq))
		}
		chainlog.Info("UpdateSeq", "seq channel", notify.seq, "seq", seq, "length", len(notify.seq))
	}
}

func (push *PushTxReceiptService) trigeRun(run chan struct{}, sleep time.Duration) {
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

func (push *PushTxReceiptService) runTask(input *pushTxReceiptNotify) {
	go func(in *pushTxReceiptNotify) {
		var lastProcessedseq int64 = -1
		var lastesBlockSeq int64 = -1
		var subscribe *types.SubscribeTxReceipt
		var run = make(chan struct{}, 10)
		var continueFailCount int32 = 0

		subscribe = in.subscribe
		push.trigeRun(run, 0)
		lastProcessedseq = push.getLastPushSeq(subscribe)
		in.status = Running
		chainlog.Debug("runTask to push tx receipt", "subscribe", subscribe, "in.seq", in.seq)
		for {
			select {
			case lastesBlockSeq = <-in.seq:
				push.trigeRun(run, 0)
			case <-run:
				if subscribe == nil {
					push.trigeRun(run, time.Second)
					continue
				}
				//没有更新的区块，则不进行处理，同时等待一定的时间
				if lastProcessedseq >= lastesBlockSeq {
					chainlog.Debug("runTask to push tx receipt run3", "lastProcessedseq", lastProcessedseq, "lastesBlockSeq", lastesBlockSeq)
					push.trigeRun(run, waitNewBlock)
					continue
				}
				//确定一次推送的数量，如果需要更新的数量少于门限值，则一次只推送一个区块的交易数据
				seqCount := pushMaxSeq
				if lastProcessedseq+int64(seqCount) > lastesBlockSeq {
					seqCount = 1
				}
				chainlog.Debug("runTask to push tx receipt run4", "lastProcessedseq", lastProcessedseq, "seqCount", seqCount)
				data, updateSeq, err := push.getTxReceipts(subscribe, lastProcessedseq+1, seqCount, pushMaxSize)
				if err != nil {
					chainlog.Error("getTxReceipts", "err", err, "seq", lastProcessedseq+1, "maxSeq", seqCount,
						"Name", subscribe.Name, "contract:", subscribe.Contract)
					push.trigeRun(run, 1000*time.Millisecond)
					continue
				}

				if data != nil {
					err = push.postData(subscribe, data, updateSeq)
					if err != nil {
						continueFailCount++
						chainlog.Error("postdata failed", "err", err, "lastProcessedseq", lastProcessedseq,
							"Name", subscribe.Name, "contract:", subscribe.Contract, "continueFailCount", continueFailCount)
						if continueFailCount >= 3 {
							in.status = NotRunning
							chainlog.Error("postdata failed exceed 3 times", "in.status", in.status)
							return
						}
						//sleep 60s
						push.trigeRun(run, 60000*time.Millisecond)
						continue
					}
				}
				continueFailCount = 0
				//update seqid
				lastProcessedseq = updateSeq
				push.trigeRun(run, 0)
			}
		}
	}(input)
}

func (push *PushTxReceiptService) getTxReceipts(subscribe *types.SubscribeTxReceipt, startSeq int64, seqCount, maxSize int) ([]byte, int64, error) {
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
		for txIndex, tx := range detail.Block.Txs {
			if string(tx.Execer) == subscribe.Contract {
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

//seq= data.Seqs[0].Num+int64(len(data.Seqs))-1
func (push *PushTxReceiptService) postData(subscribe *types.SubscribeTxReceipt, postdata []byte, seq int64) (err error) {
	//post data in body
	chainlog.Info("postData", "seq", seq, "data", string(postdata))
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
	chainlog.Info("postData", "req", req)
	resp, err := push.client.Do(req)
	if err != nil {
		chainlog.Info("postData", "Do err", err)
		return err
	}
	defer resp.Body.Close()
	chainlog.Info("postData", "resp", resp)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	chainlog.Info("postData", "body", string(body))
	if string(body) != "ok" && string(body) != "OK" {
		chainlog.Error("postData fail", "name:", subscribe.Name, "Contract:", subscribe.Contract, "body", string(body))
		return types.ErrPushSeqPostData
	}
	chainlog.Debug("postData success", "name:", subscribe.Name, "Contract:", subscribe.Contract, "updateSeq", seq)
	return push.setLastPushSeq(subscribe.Name, subscribe.Contract, seq)
}

// GetLastPushSeq Seq的合法值从0开始的，所以没有获取到或者获取失败都应该返回-1
func (push *PushTxReceiptService) getLastPushSeq(subscribe *types.SubscribeTxReceipt) int64 {
	bytes, err := push.store.GetKey(calcTxReceiptLastSeqNumKey(subscribe.Name, subscribe.Contract))
	if bytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("getLastPushSeq", "error", err)
		}
		return -1
	}
	n, err := decodeHeight(bytes)
	if err != nil {
		return -1
	}
	storeLog.Info("getLastPushSeq", "name", subscribe.Name,
		"Contract:", subscribe.Contract, "num", n)

	return n
}

func (push *PushTxReceiptService) setLastPushSeq(name, contract string, num int64) error {
	return push.store.SetSync(calcTxReceiptLastSeqNumKey(name, contract), types.Encode(&types.Int64{Data: num}))
}
