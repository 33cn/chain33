package blockchain

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

const (
	notRunning               = int32(1)
	running                  = int32(2)
	pushBlockMaxSeq          = 10
	pushTxReceiptMaxSeq      = 100
	pushMaxSize              = 1 * 1024 * 1024
	maxPushSubscriber        = int(128)
	subscribeStatusActive    = int32(1)
	subscribeStatusNotActive = int32(2)
	postFail2Sleep           = int32(60) //一次发送失败，sleep的次数
	chanBufCap               = int(10)
	encodeJSON               = "jrpc"
	encodeGrpc               = "grpc"
)

// PushType ...
type PushType int32

const (
	//PushBlock push block
	PushBlock PushType = iota
	//PushBlockHeader push block header
	PushBlockHeader
	//PushTxReceipt push tx receipt
	PushTxReceipt
	//PushTxResult push tx result
	PushTxResult
	//PushEVMEvent push evem tx event
	PushEVMEvent
)

// String format string
func (p PushType) String() string {
	str := [...]string{"PushBlock", "PushBlockHeader", "PushTxReceipt", "PushTxResult", "PushEVMEvent", "NotSupported"}
	if p < 0 || int(p) >= len(str) {
		return "(unrecognized)"
	}
	return str[p]
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

// SequenceStore ...
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

// PostService ... post rawdata to subscriber
type PostService interface {
	PostData(subscribe *types.PushSubscribeReq, postdata []byte, seq int64) (err error)
}

// 当前的实现是为每个订阅者单独启动一个协程goroute，然后单独为每个subscriber分别过滤指定类型的交易，
// 进行归类，这种方式集成了区块推送方式的处理机制，但是对于订阅者数量大的情况，势必会浪费cpu的开销，
// 数据库的读取开销不会额外增加明星，因为会有cach
// TODO：后续需要考虑将区块推送和交易执行回执推送进行重构，提高并行推送效率
// pushNotify push Notify
type pushNotify struct {
	subscribe      *types.PushSubscribeReq
	seqUpdateChan  chan int64
	closechan      chan struct{}
	status         int32
	postFail2Sleep int32
}

// Push ...
type Push struct {
	store          CommonStore
	sequenceStore  SequenceStore
	tasks          map[string]*pushNotify
	mu             sync.Mutex
	postService    PostService
	cfg            *types.Chain33Config
	postFail2Sleep int32
	postwg         *sync.WaitGroup
}

// PushClient ...
type PushClient struct {
	client  *http.Client
	qclient queue.Client
}

func buildRPCData(data []byte, subscribe *types.PushSubscribeReq) (*types.PushData, int64, error) {
	var ty int64
	var pushData types.PushData
	pushData.Name = subscribe.GetName()
	switch (PushType)(subscribe.Type) {
	case PushBlock:
		var block types.BlockSeqs
		types.Decode(data, &block)
		pushData.Value = &types.PushData_BlockSeqs{BlockSeqs: &block}
		ty = types.EventPushBlock
	case PushBlockHeader:
		var header types.HeaderSeqs
		types.Decode(data, &header)
		pushData.Value = &types.PushData_HeaderSeqs{HeaderSeqs: &header}
		ty = types.EventPushBlockHeader
	case PushTxReceipt:
		var txreceipt types.TxReceipts4Subscribe
		types.Decode(data, &txreceipt)
		pushData.Value = &types.PushData_TxReceipts{TxReceipts: &txreceipt}
		ty = types.EventPushTxReceipt
	case PushTxResult:
		var rxResult types.TxResultSeqs
		types.Decode(data, &rxResult)
		pushData.Value = &types.PushData_TxResult{TxResult: &rxResult}
		ty = types.EventPushTxResult
	case PushEVMEvent:
		var evmlogs types.EVMTxLogsInBlks
		types.Decode(data, &evmlogs)
		pushData.Value = &types.PushData_EvmLogs{EvmLogs: &evmlogs}
		ty = types.EventPushEVM
	default:
		return nil, ty, errors.New("wrong pushType")

	}
	return &pushData, ty, nil

}

// PostData ...
func (pushClient *PushClient) PostData(subscribe *types.PushSubscribeReq, postdata []byte, seq int64) (err error) {
	//post data in body
	if subscribe.GetEncode() == encodeGrpc && subscribe.GetURL() == "" { //通过queue模块推送给rpc订阅者 GRPC订阅模式
		data, ty, err := buildRPCData(postdata, subscribe)
		if err != nil {
			return err
		}
		pmsg := pushClient.qclient.NewMessage("rpc", ty, data)
		err = pushClient.qclient.SendTimeout(pmsg, true, time.Second)
		if err != nil {
			chainlog.Info("PostData", "err:", err, "sub type", (PushType)(subscribe.Type).String(), "encode:", subscribe.Encode)
			return err
		}
		resp, err := pushClient.qclient.WaitTimeout(pmsg, time.Second)
		if err != nil {
			chainlog.Error("PostData", "WaitTimeout err:", err, "sub type", (PushType)(subscribe.Type).String(), "encode:", subscribe.Encode)
			return err
		}
		if !resp.GetData().(*types.Reply).GetIsOk() {
			return errors.New(string(resp.GetData().(*types.Reply).GetMsg()))
		}
		return nil
	}
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

// ProcAddBlockSeqCB 添加seq callback
func (chain *BlockChain) procSubscribePush(subscribe *types.PushSubscribeReq) error {
	if !chain.enablePushSubscribe {
		chainlog.Error("Push is not enabled for subscribed")
		return types.ErrPushNotSupport
	}

	if !chain.isRecordBlockSequence {
		chainlog.Error("procSubscribePush can be enable after the RecordBlockSequence is configured ")
		return types.ErrRecordBlockSequence
	}

	if subscribe == nil {
		chainlog.Error("procSubscribePush para is null")
		return types.ErrInvalidParam
	}

	if chain.client.GetConfig().IsEnable("reduceLocaldb") && PushType(subscribe.Type) == PushTxReceipt {
		chainlog.Error("Tx receipts are reduced on this node")
		return types.ErrTxReceiptReduced
	}
	return chain.push.addSubscriber(subscribe)
}

// ProcListPush 列出所有已经设置的推送订阅
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
		var onePush types.PushWithStatus
		err := types.Decode(value, &onePush)
		if err != nil {
			return nil, err
		}
		listSeqCBs.Pushes = append(listSeqCBs.Pushes, onePush.Push)
	}
	return &listSeqCBs, nil
}

// ProcGetLastPushSeq Seq的合法值从0开始的，所以没有获取到或者获取失败都应该返回-1
func (chain *BlockChain) ProcGetLastPushSeq(name string) (int64, error) {
	if !chain.isRecordBlockSequence {
		return -1, types.ErrRecordBlockSequence
	}
	if !chain.enablePushSubscribe {
		return -1, types.ErrPushNotSupport
	}

	lastSeqbytes, err := chain.push.store.GetKey(calcLastPushSeqNumKey(name))
	if lastSeqbytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("getSeqCBLastNum", "error", err)
		}
		return -1, types.ErrPushNotSubscribed
	}
	n, err := decodeHeight(lastSeqbytes)
	if err != nil {
		return -1, err
	}
	storeLog.Error("getSeqCBLastNum", "name", name, "num", n)

	return n, nil
}

func newpush(commonStore CommonStore, seqStore SequenceStore, qclient queue.Client) *Push {
	tasks := make(map[string]*pushNotify)

	pushClient := &PushClient{
		qclient: qclient,
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
		sequenceStore:  seqStore,
		tasks:          tasks,
		postService:    pushClient,
		cfg:            qclient.GetConfig(),
		postFail2Sleep: postFail2Sleep,
		postwg:         &sync.WaitGroup{},
	}
	service.init()

	return service
}

// 初始化: 从数据库读出seq的数目
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
		} //过滤掉grpc的推送
		chainlog.Debug("Push init", "Push Name", pushWithStatus.Push.Name, "pushWithStatus.Status", pushWithStatus.Status)
		if pushWithStatus.Status == subscribeStatusActive {
			subscribes = append(subscribes, pushWithStatus.Push)
		}

	}
	for _, subscribe := range subscribes {
		if subscribe.GetEncode() != encodeGrpc { // grpc 不需要节点启动之后主动推送过去
			chainlog.Info("Push init", "Going to add Task to Push for Name", subscribe.Name)

			push.addTask(subscribe)
		}

	}
}

// Close ...
func (push *Push) Close() {
	push.mu.Lock()
	for _, task := range push.tasks {
		close(task.closechan)
	}
	push.mu.Unlock()
	push.postwg.Wait()
}

func (push *Push) addSubscriber(subscribe *types.PushSubscribeReq) error {
	if subscribe == nil {
		chainlog.Error("addSubscriber input para is null")
		return types.ErrInvalidParam
	}
	if PushType(subscribe.Type) < PushBlock || PushType(subscribe.Type) > PushEVMEvent {
		chainlog.Error("addSubscriber input type is error", "type", subscribe.Type)
		return types.ErrInvalidParam
	}

	//在订阅类型为PushTxReceipt，PushEVMEvent 下必须要求配置Contract 变量
	if PushType(subscribe.Type) == PushTxReceipt || PushEVMEvent == PushType(subscribe.Type) {
		if len(subscribe.GetContract()) == 0 {
			chainlog.Error("addSubscriber input type is error", "type", subscribe.Type)
			return errors.New(types.ErrInvalidParam.Error() + ":Contract must be configure")
		}
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
		if err := push.check2ResumePush(subscribeInDB); nil != err {
			return err
		}
		//当前可能是notActive的情况，重新激活后，则直接将其状态设置为active
		return push.setActive(subscribe)
	}
	push.mu.Lock()
	if len(push.tasks) >= maxPushSubscriber {
		chainlog.Error("addSubscriber too many push subscriber")
		push.mu.Unlock()
		return types.ErrTooManySeqCB
	}
	push.mu.Unlock()
	//处理需要从指定高度开始推送的订阅请求
	if subscribe.LastSequence > 0 {
		sequence, err := push.sequenceStore.GetBlockSequence(subscribe.LastSequence)
		if err != nil {
			chainlog.Error("addSubscriber continue-seqUpdateChan-push", "load-1", err)
			return err
		}
		// 注册点，在节点上存在
		// 同一高度，不一定同一个hash，有分叉的可能；但同一个hash必定同一个高度
		if bytes.Equal(sequence.GetHash(), common.HexToHash(subscribe.LastBlockHash).Bytes()) {
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
		var pushWithStatus types.PushWithStatus
		err = types.Decode(value, &pushWithStatus)
		if err != nil {
			return false, nil
		}
		return true, pushWithStatus.Push
	}
	return false, nil
}

func (push *Push) subscriberCount() int64 {
	return push.store.PrefixCount(pushPrefix)
}

// 向数据库添加交易回执订阅信息
func (push *Push) persisAndStart(subscribe *types.PushSubscribeReq) error {
	if len(subscribe.Name) > 128 || len(subscribe.URL) > 1024 {
		storeLog.Error("Invalid para to persisAndStart due to wrong length", "len(subscribe.Name)=", len(subscribe.Name),
			"len(subscribe.Contract)=", len(subscribe.Contract))
		return errors.New(types.ErrInvalidParam.Error() + ": Name or URL exceeds limit（len(Name)<128),len(URL)<1024")
	}

	if subscribe.GetURL() == "" && subscribe.GetEncode() != "grpc" {
		//非grpc通信必须要求配置url
		storeLog.Info("persisAndStart", "url empty", subscribe.GetURL(), "encode:", subscribe.GetEncode())
		return errors.New(types.ErrInvalidParam.Error() + ": URL must be configure")
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

func (push *Push) setActive(subscribe *types.PushSubscribeReq) error {
	key := calcPushKey(subscribe.Name)
	value, err := push.store.GetKey(key)
	if nil != err {
		return err
	}
	var pushWithStatus types.PushWithStatus
	err = types.Decode(value, &pushWithStatus)
	if err != nil {
		return err
	}
	//如果本身就是active状态，则不进行状态的更新
	if pushWithStatus.Status == subscribeStatusActive {
		return nil
	}

	pushWithStatus.Status = subscribeStatusActive
	storeLog.Info("setActive", "key", string(key), "subscribe", subscribe)
	return push.store.SetSync(key, types.Encode(&pushWithStatus))
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
			seqUpdateChan: make(chan int64, chanBufCap),
			closechan:     make(chan struct{}),
			status:        notRunning,
		}
		push.runTask(push.tasks[keyStr])
		storeLog.Info("check2ResumePush new pushNotify created")
		return nil
	}

	if running == atomic.LoadInt32(&notify.status) {
		storeLog.Info("Is already in state:running", "postFail2Sleep", atomic.LoadInt32(&notify.postFail2Sleep))
		atomic.StoreInt32(&notify.postFail2Sleep, 0)
		return nil
	}
	storeLog.Info("check2ResumePush to resume a push", "name", subscribe.Name)

	push.runTask(push.tasks[keyStr])
	return nil
}

// 每次add一个新push时,发送最新的seq
func (push *Push) updateLastSeq(name string) {
	last, err := push.sequenceStore.LoadBlockLastSequence()
	if err != nil {
		chainlog.Error("LoadBlockLastSequence", "err", err)
		return
	}

	notify := push.tasks[string(calcPushKey(name))]
	notify.seqUpdateChan <- last
	chainlog.Debug("updateLastSeq", "last", last, "notify.seqUpdateChan", len(notify.seqUpdateChan))
}

// addTask 每个name 有一个task, 通知新增推送
func (push *Push) addTask(subscribe *types.PushSubscribeReq) {
	push.mu.Lock()
	defer push.mu.Unlock()
	keyStr := string(calcPushKey(subscribe.Name))
	push.tasks[keyStr] = &pushNotify{
		subscribe:      subscribe,
		seqUpdateChan:  make(chan int64, chanBufCap),
		closechan:      make(chan struct{}),
		status:         notRunning,
		postFail2Sleep: 0,
	}

	push.runTask(push.tasks[keyStr])
}

func trigeRun(run chan struct{}, sleep time.Duration, name string) {
	chainlog.Info("trigeRun", name, "name", "sleep", sleep, "run len", len(run))
	if sleep > 0 {
		time.Sleep(sleep)
	}
	go func() {
		run <- struct{}{}
	}()
}

func (push *Push) runTask(input *pushNotify) {
	//触发goroutine运行
	push.updateLastSeq(input.subscribe.Name)

	go func(in *pushNotify) {
		var lastesBlockSeq int64
		var continueFailCount int32
		var err error

		subscribe := in.subscribe
		lastProcessedseq := push.getLastPushSeq(subscribe)
		atomic.StoreInt32(&in.status, running)

		runChan := make(chan struct{}, 10)
		pushMaxSeq := pushBlockMaxSeq
		if PushType(subscribe.Type) == PushTxReceipt {
			pushMaxSeq = pushTxReceiptMaxSeq
		}

		chainlog.Debug("start push with info", "subscribe name", subscribe.Name, "Type", PushType(subscribe.Type).String())
		for {
			select {
			case <-runChan:
				if atomic.LoadInt32(&input.postFail2Sleep) > 0 {
					if postFail2SleepNew := atomic.AddInt32(&input.postFail2Sleep, -1); postFail2SleepNew > 0 {
						chainlog.Debug("wait another ticker for post fail", "postFail2Sleep", postFail2SleepNew, "name", in.subscribe.Name)
						trigeRun(runChan, time.Second, subscribe.Name)
						continue
					}
				}

			case lastestSeq := <-in.seqUpdateChan:
				chainlog.Debug("runTask recv:", "lastestSeq", lastestSeq, "subscribe name", subscribe.Name, "Type", PushType(subscribe.Type).String())
				//首先判断是否存在发送失败的情况，如果存在，则进行进行sleep操作
				if atomic.LoadInt32(&input.postFail2Sleep) > 0 {
					if postFail2SleepNew := atomic.AddInt32(&input.postFail2Sleep, -1); postFail2SleepNew > 0 {
						chainlog.Debug("wait another ticker for post fail", "postFail2Sleep", postFail2SleepNew, "name", in.subscribe.Name)
						trigeRun(runChan, time.Second, subscribe.Name)
						continue
					}
				}
				//获取当前最新的sequence,这样就可以一次性发送多个区块的信息，而不需要每次从通知chan中获取最新sequence
				if lastesBlockSeq, err = push.sequenceStore.LoadBlockLastSequence(); err != nil {
					chainlog.Error("LoadBlockLastSequence", "err", err)
					return
				}

				//没有更新的区块，则不进行处理，同时等待一定的时间
				if lastProcessedseq >= lastesBlockSeq {
					continue
				}
				if lastProcessedseq <= 0 { //如果不配置startSeq 则默认从最新的seq开始
					lastProcessedseq = lastesBlockSeq
					continue
				}
				chainlog.Debug("another new block", "subscribe name", subscribe.Name, "Type", PushType(subscribe.Type).String(),
					"last push sequence", lastProcessedseq, "lastest sequence", lastesBlockSeq,
					"time second", time.Now().Second())
				//确定一次推送的数量，如果需要更新的数量少于门限值，则一次只推送一个区块的交易数据
				seqCount := pushMaxSeq
				if seqCount > int(lastesBlockSeq-lastProcessedseq) {
					seqCount = int(lastesBlockSeq - lastProcessedseq)
				}
				data, updateSeq, err := push.getPushData(subscribe, lastProcessedseq+1, seqCount, pushMaxSize)
				if err != nil {
					chainlog.Error("getPushData", "err", err, "seqCurrent", lastProcessedseq+1, "maxSeq", seqCount,
						"Name", subscribe.Name, "pushType:", PushType(subscribe.Type).String())
					continue
				}

				if data != nil {
					err = push.postService.PostData(subscribe, data, updateSeq)
					if err != nil {
						continueFailCount++
						chainlog.Error("postdata failed", "err", err, "lastProcessedseq", lastProcessedseq,
							"Name", subscribe.Name, "pushType:", PushType(subscribe.Type).String(), "continueFailCount", continueFailCount)
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
							//多次Post失败后，把这个subscriber设置为NoActive状态，停止这个task的运行
							_ = push.store.SetSync(key, types.Encode(pushWithStatus))
							push.postwg.Done()
							return
						}
						//sleep 60s，每次1s，总计60次，在每次结束时，等待接收方重新进行请求推送
						atomic.StoreInt32(&input.postFail2Sleep, push.postFail2Sleep)
						trigeRun(runChan, time.Second, subscribe.Name)
						continue
					}
					_ = push.setLastPushSeq(subscribe.Name, updateSeq)
				}
				continueFailCount = 0
				lastProcessedseq = updateSeq
				// 在联盟链情况下, 无新增交易的情况下, 不会完成从新开始同步
				// 在公链情况下, 需要有新区块才能触发推送,
				// 所以这里在未同步到最新区块, 需要主动触发同步
				if lastProcessedseq < lastesBlockSeq {
					push.mu.Lock()
					if len(in.seqUpdateChan) == 0 {
						in.seqUpdateChan <- lastesBlockSeq
					}
					push.mu.Unlock()
				}
			case <-in.closechan:
				push.postwg.Done()
				chainlog.Info("getPushData", "push task closed for subscribe", subscribe.Name)
				return
			}
		}

	}(input)
	push.postwg.Add(1)
}

// UpdateSeq sequence 更新通知
func (push *Push) UpdateSeq(seq int64) {
	push.mu.Lock()
	defer push.mu.Unlock()
	for _, notify := range push.tasks {
		//再写入seq（一定不会block，因为加了lock，不存在两个同时写channel的情况）
		if len(notify.seqUpdateChan) < chanBufCap {
			chainlog.Info("new block Update Seq notified", "subscribe", notify.subscribe.Name, "current sequence", seq, "length", len(notify.seqUpdateChan), "status", atomic.LoadInt32(&notify.status))
			notify.seqUpdateChan <- seq
		}
		chainlog.Info("new block UpdateSeq", "subscribe", notify.subscribe.Name, "current sequence", seq, "length", len(notify.seqUpdateChan), "status", atomic.LoadInt32(&notify.status))
	}
}

func (push *Push) getPushData(subscribe *types.PushSubscribeReq, startSeq int64, seqCount, maxSize int) ([]byte, int64, error) {
	switch PushType(subscribe.Type) {
	case PushBlock:
		return push.getBlockSeqs(subscribe.Encode, startSeq, seqCount, maxSize)
	case PushBlockHeader:
		return push.getHeaderSeqs(subscribe.Encode, startSeq, seqCount, maxSize)
	case PushTxReceipt:
		return push.getTxReceipts(subscribe, startSeq, seqCount, maxSize)
	case PushTxResult:
		return push.getTxResults(subscribe.Encode, startSeq, seqCount)
	case PushEVMEvent:
		return push.getEVMEvent(subscribe, startSeq, seqCount, maxSize)
	default:
		return nil, 0, errors.New("wrong subscribe type")
	}
}

func (push *Push) getEVMEvent(subscribe *types.PushSubscribeReq, startSeq int64, seqCount, maxSize int) ([]byte, int64, error) {
	evmlogs := &types.EVMTxLogsInBlks{}
	totalSize := 0
	actualIterCount := 0
	for i := startSeq; i < startSeq+int64(seqCount); i++ {
		chainlog.Debug("getEVMEvent", "startSeq:", i)
		seqdata, err := push.sequenceStore.GetBlockSequence(i)
		if err != nil {
			return nil, -1, err
		}
		detail, _, err := push.sequenceStore.LoadBlockBySequence(i)
		if err != nil {
			return nil, -1, err
		}

		evmLogsPerBlk := &types.EVMTxLogPerBlk{}
		chainlog.Debug("getEVMEvent", "height:", detail.Block.Height, "tx numbers:", len(detail.Block.Txs),
			"Receipts numbers:", len(detail.Receipts))
		for txIndex, tx := range detail.Block.Txs {
			//确认是订阅的交易类型
			if !strings.Contains(string(tx.Execer), "evm") {
				continue
			}
			var evmAction types.EVMContractAction4Chain33
			err := types.Decode(tx.Payload, &evmAction)
			if nil != err {
				chainlog.Error("getEVMEvent", "Failed to decode EVMContractAction for evm tx with hash:", common.ToHex(tx.Hash()))
				continue
			}
			if subscribe.Contract[evmAction.ContractAddr] {
				chainlog.Debug("getEVMEvent", "txIndex:", txIndex)
				//因为只有交易执行成功时，才会存证log信息，所以需要事先判断
				if types.ExecOk != detail.Receipts[txIndex].Ty {
					continue
				}
				//填充log数据
				evmLogsPerTx := &types.EVMLogsPerTx{}
				for _, log := range detail.Receipts[txIndex].Logs {
					//TyLogEVMEventData = 605 这个log类型定义在evm合约内部
					//只填充event类型的log数据
					if 605 != log.Ty {
						continue
					}
					var evmLog types.EVMLog
					err := types.Decode(log.Log, &evmLog)
					if nil != err {
						return nil, -1, err
					}
					evmLogsPerTx.Logs = append(evmLogsPerTx.Logs, &evmLog)
				}
				if nil != evmLogsPerTx.Logs {
					txAndLogs := &types.EVMTxAndLogs{
						Tx:        tx,
						LogsPerTx: evmLogsPerTx,
					}
					evmLogsPerBlk.TxAndLogs = append(evmLogsPerBlk.TxAndLogs, txAndLogs)
				}
			}
		}
		if len(evmLogsPerBlk.TxAndLogs) > 0 {
			evmLogsPerBlk.Height = detail.Block.Height
			evmLogsPerBlk.BlockHash = detail.Block.Hash(push.cfg)
			evmLogsPerBlk.ParentHash = detail.Block.ParentHash
			evmLogsPerBlk.PreviousHash = []byte{}
			evmLogsPerBlk.AddDelType = int32(seqdata.Type)
			evmLogsPerBlk.SeqNum = i
		}
		size := types.Size(evmLogsPerBlk)
		if len(evmLogsPerBlk.TxAndLogs) > 0 && totalSize+size < maxSize {
			evmlogs.Logs4EVMPerBlk = append(evmlogs.Logs4EVMPerBlk, evmLogsPerBlk)
			totalSize += size
			chainlog.Debug("get EVMEvent subscribed for pushing", "Name", subscribe.Name, "contract:", subscribe.Contract,
				"height=", evmLogsPerBlk.Height)
		} else if totalSize+size > maxSize {
			break
		}
		actualIterCount++
	}

	updateSeq := startSeq + int64(actualIterCount) - 1
	if len(evmlogs.Logs4EVMPerBlk) == 0 {
		return nil, updateSeq, nil
	}
	chainlog.Info("getEVMEvent", "updateSeq", updateSeq, "actualIterCount", actualIterCount)

	var postdata []byte
	var err error
	if subscribe.Encode == encodeJSON {
		postdata, err = types.PBToJSON(evmlogs)
		if err != nil {
			return nil, -1, err
		}
	} else {
		postdata = types.Encode(evmlogs)
	}

	return postdata, updateSeq, nil
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
	if subscribe.Encode == encodeJSON {
		postdata, err = types.PBToJSON(txReceipts)
		if err != nil {
			return nil, -1, err
		}
	} else {
		postdata = types.Encode(txReceipts)
	}

	return postdata, updateSeq, nil
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

func (push *Push) getTxResults(encode string, seq int64, seqCount int) ([]byte, int64, error) {
	var txResultSeqs types.TxResultSeqs
	for i := seq; i < seq+int64(seqCount); i++ {
		blockSeq, _, err := push.getBlockDataBySeq(i)
		if err != nil {
			return nil, -1, err
		}
		txResults := types.TxResultPerBlock{
			Items:      make([]*types.TxHashWithReceiptType, len(blockSeq.Detail.Receipts)),
			Height:     blockSeq.Detail.Block.Height,
			BlockHash:  blockSeq.Detail.Block.Hash(push.cfg),
			ParentHash: blockSeq.Detail.Block.ParentHash,
			AddDelType: int32(blockSeq.Seq.Type),
			SeqNum:     blockSeq.Num,
		}
		for i := range txResults.Items {
			txResults.Items[i] = &types.TxHashWithReceiptType{
				Hash: blockSeq.Detail.Block.Txs[i].Hash(),
				Ty:   blockSeq.Detail.Receipts[i].Ty,
			}
		}
		txResultSeqs.Items = append(txResultSeqs.Items, &txResults)
	}

	var postdata []byte
	var err error
	if encode == encodeJSON {
		postdata, err = types.PBToJSON(&txResultSeqs)
		if err != nil {
			return nil, -1, err
		}
	} else {
		postdata = types.Encode(&txResultSeqs)
	}
	return postdata, txResultSeqs.Items[0].SeqNum + int64(len(txResultSeqs.Items)) - 1, nil
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
	if encode == encodeJSON {
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

	if encode == encodeJSON {
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
	chainlog.Info("getLastPushSeq", "name", subscribe.Name,
		"Contract:", subscribe.Contract, "num", n)

	return n
}

func (push *Push) setLastPushSeq(name string, num int64) error {
	return push.store.SetSync(calcLastPushSeqNumKey(name), types.Encode(&types.Int64{Data: num}))
}
