package blockchain

import (
	dbm "common/db"
	"common/merkle"
	"container/list"
	"errors"
	"fmt"
	"queue"
	"time"
	"types"

	log "github.com/inconshreveable/log15"
)

var (
	//cache ������block����
	DefCacheSize int = 500

	//һ����������ȡblock����
	MaxFetchBlockNum  int64  = 100
	TimeoutSeconds    int64  = 15
	BatchBlockNum     int64  = 100
	reqTimeoutSeconds        = time.Duration(TimeoutSeconds)
	Datadir           string = "datadir"
)

var chainlog = log.New("module", "blockchain")

const trySyncIntervalMS = 1000

type BlockChain struct {
	qclient queue.IClient

	// ���ô洢���ݵ�db��
	blockStore *BlockStore

	//cache  ����block������ٲ�ѯ
	cache      map[string]*list.Element
	cacheSize  int
	cacheQueue *list.List

	//Block ͬ���׶����ڻ���block��Ϣ��
	blockPool *BlockPool

	//��������block��ʱ�Ĵ���
	reqBlk map[int64]*bpRequestBlk
}

func New() *BlockChain {

	//��ʼ��blockstore ��txindex  db
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", Datadir)
	blockStore := NewBlockStore(blockStoreDB)

	pool := NewBlockPool()
	reqblk := make(map[int64]*bpRequestBlk)

	return &BlockChain{
		blockStore: blockStore,
		cache:      make(map[string]*list.Element),
		cacheSize:  DefCacheSize,
		cacheQueue: list.New(),
		blockPool:  pool,
		reqBlk:     reqblk,
	}
}

func (chain *BlockChain) SetQueue(q *queue.Queue) {
	chain.qclient = q.GetClient()
	chain.qclient.Sub("blockchain")

	//recv ��Ϣ�Ĵ���
	go chain.ProcRecvMsg()

	// ��ʱͬ�������е�block to db
	go chain.poolRoutine()
}
func (chain *BlockChain) ProcRecvMsg() {
	for msg := range chain.qclient.Recv() {
		msgtype := msg.Ty
		switch msgtype {
		case types.EventQueryTx:
			txhash := (msg.Data).(types.RequestHash)
			merkleproof, err := chain.ProcQueryTxMsg(txhash.Hash)
			if err != nil {
				chainlog.Error("ProcQueryTxMsg", "err", err.Error())
				var reply types.Reply
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
				msg.Reply(chain.qclient.NewMessage("blockchain", types.EventReply, &reply))
			} else {
				msg.Reply(chain.qclient.NewMessage("blockchain", types.EventMerkleProof, merkleproof))
			}

		case types.EventGetBlocks:
			requestblocks := (msg.Data).(*types.RequestBlocks)
			blocks, err := chain.ProcGetBlocksMsg(requestblocks)
			if err != nil {
				chainlog.Error("ProcGetBlocksMsg", "err", err.Error())
				var reply types.Reply
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
				msg.Reply(chain.qclient.NewMessage("blockchain", types.EventReply, &reply))
			} else {
				msg.Reply(chain.qclient.NewMessage("blockchain", types.EventBlocks, blocks))
			}

		case types.EventAddBlock:
			var block *types.Block
			var reply types.Reply
			reply.IsOk = true
			block = msg.Data.(*types.Block)
			err := chain.ProcAddBlockMsg(block)
			if err != nil {
				chainlog.Error("ProcAddBlockMsg", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(chain.qclient.NewMessage("blockchain", types.EventReply, &reply))

		case types.EventAddBlocks:
			var blocks *types.Blocks
			var reply types.Reply
			reply.IsOk = true
			blocks = msg.Data.(*types.Blocks)
			err := chain.ProcAddBlocksMsg(blocks)
			if err != nil {
				chainlog.Error("ProcAddBlocksMsg", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(chain.qclient.NewMessage("blockchain", types.EventReply, &reply))

		case types.EventGetBlockHeight:
			var replyBlockHeight types.ReplyBlockHeight
			replyBlockHeight.Height = chain.GetBlockHeight()
			msg.Reply(chain.qclient.NewMessage("blockchain", types.EventReplyBlockHeight, &replyBlockHeight))

		case types.EventTxHashList:
			txhashlist := (msg.Data).(*types.TxHashList)
			duptxhashlist := chain.GetDuplicateTxHashList(txhashlist)
			msg.Reply(chain.qclient.NewMessage("blockchain", types.EventTxHashListReply, duptxhashlist))

		default:
			chainlog.Info("ProcRecvMsg unknow msg", "msgtype", msgtype)
		}
	}
}

func (chain *BlockChain) poolRoutine() {
	trySyncTicker := time.NewTicker(trySyncIntervalMS * time.Millisecond)
FOR_LOOP:
	for {
		select {
		case <-trySyncTicker.C:
			// ��ʱͬ�������е�block��Ϣ��db���ݿ���
			newbatch := chain.blockStore.NewBatch(true)
			var stratblockheight int64 = 0
			var endblockheight int64 = 0
			var i int64
			// ������������BatchBlockNum��block��db��
			currentheight := chain.blockStore.Height()
			for i = 1; i <= BatchBlockNum; i++ {

				block := chain.blockPool.GetBlock(currentheight + i)
				if block == nil && i == 1 { //��Ҫ���صĵ�һ��nextblock�����ڣ��˳�forѭ��������һ����ʱ
					continue FOR_LOOP
				}
				// ������������block��С��BatchBlockNumʱ��ͬ�����е�block��db��
				if block == nil {
					break
				}
				//���ڼ�¼ͬ����ʼ�ĵ�һ��block�߶ȣ�����ͬ�����֮��ɾ�������е�block��¼
				if i == 1 {
					stratblockheight = block.Height
				}
				//����tx��Ϣ��db��
				err := chain.blockStore.indexTxs(newbatch, block)
				if err != nil {
					break
				}
				//����block��Ϣ��db��
				err = chain.blockStore.SaveBlock(newbatch, block)
				if err != nil {
					break
				}
				//��¼ͬ������ʱ���һ��block�ĸ߶ȣ�����ͬ�����֮��ɾ�������е�block��¼
				endblockheight = block.Height
			}
			newbatch.Write()

			//����db�е�blockheight��blockStore.Height
			chain.blockStore.UpdateHeight()

			//ɾ�������е�block
			for j := stratblockheight; j <= endblockheight; j++ {
				//���Ѿ��洢��blocks��ӵ�list�����б��ڲ���
				block := chain.blockPool.GetBlock(j)
				if block != nil {
					chain.cacheBlock(block)
				}
				//֪ͨmempool��consenseģ��
				chain.SendAddBlockEvent(block)

				chain.blockPool.DelBlock(j)
			}
			continue FOR_LOOP
		}
	}
}

/*
�������ܣ�
EventQueryTx(types.RequestHash) : rpcģ����� blockchain ģ�� ���� EventQueryTx(types.RequestHash) ��Ϣ ��
��ѯ���׵�Ĭ�˶������ظ���Ϣ EventMerkleProof(types.MerkleProof)
�ṹ�壺
type RequestHash struct {Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`}
type MerkleProof struct {Hashs [][]byte `protobuf:"bytes,1,rep,name=hashs,proto3" json:"hashs,omitempty"}
*/
func (chain *BlockChain) ProcQueryTxMsg(txhash []byte) (proof *types.MerkleProof, err error) {
	txresult, err := chain.GetTxResultFromDb(txhash)
	if err != nil {
		return nil, err
	}
	block, err := chain.GetBlock(txresult.Height)
	if err != nil {
		return nil, err
	}
	merkleproof, err := GetMerkleProof(block.Txs, txresult.Index)
	if err != nil {
		return nil, err
	}
	return merkleproof, nil
}

func (chain *BlockChain) GetDuplicateTxHashList(txhashlist *types.TxHashList) (duptxhashlist *types.TxHashList) {

	var dupTxHashList types.TxHashList

	for _, txhash := range txhashlist.Hashes {
		txresult, err := chain.GetTxResultFromDb(txhash)
		if err == nil && txresult != nil {
			dupTxHashList.Hashes = append(dupTxHashList.Hashes, txhash)
			//chainlog.Debug("GetDuplicateTxHashList txresult", "height", txresult.Height, "index", txresult.Index)
			//chainlog.Debug("GetDuplicateTxHashList txresult  tx", "txinfo", txresult.Tx.String())
		}
	}
	return &dupTxHashList
}

/*
EventGetBlocks(types.RequestGetBlock): rpc ģ�� ���� blockchain ģ�鷢�� EventGetBlocks(types.RequestGetBlock) ��Ϣ��
�����ǲ�ѯ �������Ϣ, �ظ���Ϣ�� EventBlocks(types.Blocks)
type RequestBlocks struct {
	Start int64 `protobuf:"varint,1,opt,name=start" json:"start,omitempty"`
	End   int64 `protobuf:"varint,2,opt,name=end" json:"end,omitempty"`}
type Blocks struct {Items []*Block `protobuf:"bytes,1,rep,name=items" json:"items,omitempty"`}
*/
func (chain *BlockChain) ProcGetBlocksMsg(requestblock *types.RequestBlocks) (respblocks *types.Blocks, err error) {
	blockhight := chain.GetBlockHeight()
	if requestblock.Start >= blockhight {
		outstr := fmt.Sprintf("input Start height :%d  but current height:%d", requestblock.Start, blockhight)
		err = errors.New(outstr)
		return nil, err
	}
	end := requestblock.End
	if requestblock.End >= blockhight {
		end = blockhight
	}
	start := requestblock.Start
	count := end - start + 1
	chainlog.Debug("ProcGetBlocksMsg", "blockscount", count)

	var blocks types.Blocks
	blocks.Items = make([]*types.Block, count)
	j := 0
	for i := start; i <= end; i++ {
		block, err := chain.GetBlock(i)
		if err == nil && block != nil {
			blocks.Items[j] = block
		} else {
			return nil, err
		}
		j++
	}
	return &blocks, nil
}

/*
EventAddBlock(types.Block), P2Pģ�����ϵͳ���� EventAddBlock(types.Block) �����󣬱�ʾ���һ�����顣
��ʱ�򣬹㲥���������鲻�ǵ�ǰ�߶�+1���ڵȴ�һ����ʱʱ���Ժ󡣿��������������顣
type Block struct {
	ParentHash []byte         `protobuf:"bytes,1,opt,name=parentHash,proto3" json:"parentHash,omitempty"`
	TxHash     []byte         `protobuf:"bytes,2,opt,name=txHash,proto3" json:"txHash,omitempty"`
	BlockTime  int64          `protobuf:"varint,3,opt,name=blockTime" json:"blockTime,omitempty"`
	Txs        []*Transaction `protobuf:"bytes,4,rep,name=txs" json:"txs,omitempty"`
}
*/
func (chain *BlockChain) ProcAddBlockMsg(block *types.Block) (err error) {
	currentheight := chain.GetBlockHeight()

	//����������Ҫ�ĸ߶�ֱ�ӷ���
	if currentheight >= block.Height {
		outstr := fmt.Sprintf("input add height :%d ,current store height:%d", block.Height, currentheight)
		err = errors.New(outstr)
		return err
	} else if block.Height == currentheight+1 { //������Ҫ�ĸ߶ȣ�ֱ�Ӵ洢��db��
		newbatch := chain.blockStore.NewBatch(true)

		//����tx��Ϣ��db��
		chain.blockStore.indexTxs(newbatch, block)
		if err != nil {
			return err
		}
		//����block��Ϣ��db��
		err := chain.blockStore.SaveBlock(newbatch, block)
		if err != nil {
			return err
		}
		newbatch.Write()

		//����db�е�blockheight��blockStore.Height
		chain.blockStore.UpdateHeight()

		//����block��ӵ������б��ڲ���
		chain.cacheBlock(block)

		//ɾ����block�ĳ�ʱ����
		chain.RemoveReqBlk(block.Height, block.Height)

		//֪ͨmempool��consenseģ��
		chain.SendAddBlockEvent(block)

		return nil
	} else {
		// ����block ����ӵ�������blockpool�С�
		//����һ����ʱ��ʱ��������ڹ涨ʱ����û���յ��ͷ���һ��FetchBlock��Ϣ��p2pģ��
		//����currentheight+1 �� block.Height-1֮���blocks
		chain.blockPool.AddBlock(block)
		chain.WaitReqBlk(currentheight+1, block.Height-1)
	}
	return nil
}

/*
�������ܣ�
ͨ����P2Pģ���� EventFetchBlock(types.RequestGetBlock)���������ڵ������������飬
P2P�����յ������Ϣ�󣬻���blockchain ģ��ظ��� EventReply��
�����ڵ�����������Χ�����飬P2Pģ���յ������ڵ㷢�������ݣ�
�ᷢ����EventAddBlocks(types.Blocks) �� blockchain ģ�飬
blockchain ģ��ظ� EventReply
�ṹ�壺
*/
func (chain *BlockChain) FetchBlock(reqblk *types.RequestBlocks) (err error) {
	if chain.qclient == nil {
		fmt.Println("chain client not bind message queue.")
		err := errors.New("chain client not bind message queue")
		return err
	}

	chainlog.Debug("FetchBlock", "StartHeight", reqblk.Start, "EndHeight", reqblk.End)
	blockcount := reqblk.End - reqblk.Start
	if blockcount > MaxFetchBlockNum {
		chainlog.Error("FetchBlock", "blockscount", blockcount, "MaxFetchBlockNum", MaxFetchBlockNum)
		err := errors.New("FetchBlock blockcount > MaxFetchBlockNum")
		return err
	}
	var requestblock types.RequestBlocks

	requestblock.Start = reqblk.Start
	requestblock.End = reqblk.End

	msg := chain.qclient.NewMessage("p2p", types.EventFetchBlocks, &requestblock)
	chain.qclient.Send(msg, true)
	resp, err := chain.qclient.Wait(msg)
	if err != nil {
		chainlog.Error("FetchBlock", "qclient.Wait err:", err)
		return err
	}
	return resp.Err()
}

//blockchain ģ��add block��db֮��֪ͨmempool ��consenseģ������Ӧ�ĸ���
func (chain *BlockChain) SendAddBlockEvent(block *types.Block) (err error) {
	if chain.qclient == nil {
		fmt.Println("chain client not bind message queue.")
		err := errors.New("chain client not bind message queue")
		return err
	}
	if block == nil {
		chainlog.Error("SendAddBlockEvent block is null")
		return nil
	}
	chainlog.Debug("SendAddBlockEvent", "Height", block.Height)

	msg := chain.qclient.NewMessage("mempool", types.EventAddBlock, block)
	chain.qclient.Send(msg, true)
	_, err = chain.qclient.Wait(msg)
	if err != nil {
		chainlog.Error("SendAddBlockEvent", "send to mempool err:", err)
	}

	msg = chain.qclient.NewMessage("consense", types.EventAddBlock, block)
	chain.qclient.Send(msg, true)
	_, err = chain.qclient.Wait(msg)
	if err != nil {
		chainlog.Error("SendAddBlockEvent", "send to consense err:", err)
	}

	return nil
}

func (chain *BlockChain) ProcAddBlocksMsg(blocks *types.Blocks) (err error) {

	//���Ȼ��浽pool��,��poolRoutine��ʱͬ����db��,blocks̫���ʱд��db���ʱ�ܳ�
	blocklen := len(blocks.Items)
	startblockheight := blocks.Items[0].Height
	endblockheight := blocks.Items[blocklen-1].Height

	for _, block := range blocks.Items {
		chain.blockPool.AddBlock(block)
	}
	//ɾ����blocks�ĳ�ʱ����
	chain.RemoveReqBlk(startblockheight, endblockheight)

	return nil
}

func (chain *BlockChain) GetBlockHeight() int64 {
	return chain.blockStore.height
}

//���ڻ�ȡָ���߶ȵ�block�������ڻ����л�ȡ����������ھʹ�db�л�ȡ

func (chain *BlockChain) GetBlock(height int64) (block *types.Block, err error) {

	// Check the cache.
	elem, ok := chain.cache[string(height)]
	if ok {
		// Already exists. Move to back of cacheQueue.
		chain.cacheQueue.MoveToBack(elem)
		return elem.Value.(*types.Block), nil
	} else {
		//��blockstore db��ͨ��block height��ȡblock
		blockinfo := chain.blockStore.LoadBlock(height)
		if blockinfo != nil {
			chain.cacheBlock(blockinfo)
			return blockinfo, nil
		}
	}
	err = errors.New("GetBlock error")
	return nil, err
}

//���block��cache�У�������ٲ�ѯ
func (chain *BlockChain) cacheBlock(block *types.Block) {

	// Create entry in cache and append to cacheQueue.
	elem := chain.cacheQueue.PushBack(block)
	chain.cache[string(block.Height)] = elem

	// Maybe expire an item.
	if chain.cacheQueue.Len() > chain.cacheSize {
		height := chain.cacheQueue.Remove(chain.cacheQueue.Front()).(*types.Block).Height
		delete(chain.cache, string(height))
	}
}

/*
ͨ��txhash ��txindex db�л�ȡtx��Ϣ
type TxResult struct {
	Height int64                 `json:"height"`
	Index  int32                 `json:"index"`
	Tx     *types.Transaction    `json:"tx"`
}
*/
func (chain *BlockChain) GetTxResultFromDb(txhash []byte) (tx *types.TxResult, err error) {

	txinfo, err := chain.blockStore.GetTx(txhash)
	if err != nil {
		return nil, err
	}
	return txinfo, nil
}

//ɾ����Ӧblock�ĳ�ʱ�������
func (chain *BlockChain) RemoveReqBlk(startheight int64, endheight int64) {
	chainlog.Debug("RemoveReqBlk", "startheight", startheight, "endheight", endheight)

	delete(chain.reqBlk, startheight)
}

//������Ӧblock�ĳ�ʱ�������
func (chain *BlockChain) WaitReqBlk(start int64, end int64) {
	chainlog.Debug("WaitReqBlk", "startheight", start, "endheight", end)

	reqblk := chain.reqBlk[start]
	if reqblk != nil {
		(chain.reqBlk[start]).resetTimeout()
	} else {
		blockcount := end - start + 1

		if blockcount <= MaxFetchBlockNum {
			requestBlk := newBPRequestBlk(chain, start, end)
			chain.reqBlk[start] = requestBlk
			(chain.reqBlk[start]).resetTimeout()

		} else { //��Ҫ��η�������
			multiple := blockcount / MaxFetchBlockNum
			remainder := blockcount % MaxFetchBlockNum
			var count int64
			for count = 0; count < multiple; count++ {
				startheight := count*MaxFetchBlockNum + start
				endheight := startheight + MaxFetchBlockNum - 1

				requestBlk := newBPRequestBlk(chain, startheight, endheight)
				chain.reqBlk[startheight] = requestBlk
				(chain.reqBlk[startheight]).resetTimeout()
			}
			//  ��Ҫ���������������
			if count == multiple && remainder != 0 {
				startheight := count*MaxFetchBlockNum + start
				endheight := startheight + remainder - 1

				requestBlk := newBPRequestBlk(chain, startheight, endheight)
				chain.reqBlk[startheight] = requestBlk
				(chain.reqBlk[startheight]).resetTimeout()
			}
		}
	}
}

//��Ӧblock����ʱ����FetchBlock
func (chain *BlockChain) sendTimeout(startheight int64, endheight int64) {

	var reqBlock types.RequestBlocks
	reqBlock.Start = startheight
	reqBlock.End = endheight
	chain.FetchBlock(&reqBlock)
}

//-------------------------------------

type bpRequestBlk struct {
	blockchain  *BlockChain
	startheight int64
	endheight   int64
	timeout     *time.Timer
	didTimeout  bool //�����жϷ���һ�κ󣬳�ʱ�˻�Ҫ��Ҫ�ٴη���
}

func newBPRequestBlk(chain *BlockChain, startheight int64, endheight int64) *bpRequestBlk {
	bprequestBlk := &bpRequestBlk{
		blockchain:  chain,
		startheight: startheight,
		endheight:   endheight,
	}
	return bprequestBlk
}

func (bpReqBlk *bpRequestBlk) resetTimeout() {
	if bpReqBlk.timeout == nil {
		bpReqBlk.timeout = time.AfterFunc(time.Second*reqTimeoutSeconds, bpReqBlk.onTimeout)
	} else {
		bpReqBlk.timeout.Reset(time.Second * reqTimeoutSeconds)
	}
}

func (bpReqBlk *bpRequestBlk) onTimeout() {
	bpReqBlk.blockchain.sendTimeout(bpReqBlk.startheight, bpReqBlk.endheight)
	bpReqBlk.didTimeout = true
}

//  ��ȡָ��txindex  ��txs�е�merkleproof ��ע�ͣ�index��0��ʼ
func GetMerkleProof(Txs []*types.Transaction, index int32) (*types.MerkleProof, error) {

	var txproof types.MerkleProof
	txlen := len(Txs)

	//����tx��hashֵ
	leaves := make([][]byte, txlen)
	for index, tx := range Txs {
		leaves[index] = tx.Hash()
		//chainlog.Info("GetMerkleProof txhash", "index", index, "txhash", tx.Hash())
	}

	merkleproof, _ := merkle.ComputeMerkleBranch(leaves, int(index))
	txproof.Hashs = merkleproof
	return &txproof, nil
}
