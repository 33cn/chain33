package tendermint

import (
	"errors"
	"sync"

	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
)

type BaseQueue struct {
	oldProposal  []*types.Proposal
	mutex        sync.Mutex
	front        int
	rear         int
	cap          int
	height2Index map[int64]int //proposal of height
	index2Height map[int]int64 //to find proposal of height
}

func (bq *BaseQueue) InitQueue(capacity int) {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()
	if capacity <= 0 {
		tendermintlog.Error("InitQueue:Queue capacity must > 0")
		return
	}
	if bq.oldProposal == nil {
		bq.oldProposal = make([]*types.Proposal, capacity+1)
	} else {
		tendermintlog.Error("InitQueue:Queue already inited or not clear last time")
		panic("Queue already inited or not clear last time")
	}
	bq.front = 0
	bq.rear = 0
	bq.cap = cap(bq.oldProposal)
	bq.height2Index = make(map[int64]int, capacity)
	bq.index2Height = make(map[int]int64, capacity)
}

func (bq *BaseQueue) ClearQueue() {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()
	if bq.oldProposal == nil {
		bq.front = 0
		bq.rear = 0
		tendermintlog.Error("Queue is already nil")
		return
	}
	bq.oldProposal = nil
	bq.front = 0
	bq.rear = 0
	bq.cap = 0
	for index := range bq.index2Height {
		delete(bq.index2Height, index)
	}
	bq.index2Height = nil
	for height := range bq.height2Index {
		delete(bq.height2Index, height)
	}
	bq.height2Index = nil
}

func (bq *BaseQueue) IsEmpty() (bool, error) {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()
	if bq.cap == 0 {
		tendermintlog.Error("IsEmpty:Queue is not init or already cleared")
		return false, errors.New("Queue is not init or already cleared")
	}
	return bq.front == bq.rear, nil
}

func (bq *BaseQueue) IsFull() (bool, error) {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()
	if bq.cap == 0 {
		tendermintlog.Error("IsFull:Queue is not init or already cleared")
		return false, errors.New("Queue is not init or already cleared")
	}
	return (bq.rear+1)%bq.cap == bq.front, nil
}

func (bq *BaseQueue) Length() (int, error) {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()
	if bq.cap == 0 {
		tendermintlog.Error("Length:Queue is not init or already cleared")
		return 0, errors.New("Queue is not init or already cleared")
	}
	return (bq.rear - bq.front + bq.cap) % bq.cap, nil
}

//may not useful
func (bq *BaseQueue) GetHead() (*types.Proposal, error) {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()
	if bq.cap == 0 {
		tendermintlog.Error("GetHead:Queue is not init or already cleared")
		return nil, errors.New("Queue is not init or already cleared")
	}
	return bq.oldProposal[bq.front], nil
}

func (bq *BaseQueue) Enqueue(proposal *types.Proposal) error {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()
	if bq.cap == 0 {
		tendermintlog.Error("Enqueue:Queue is not init or already cleared")
		return errors.New("Queue is not init or already cleared")
	}
	//if full, cover the oldest one
	if (bq.rear+1)%bq.cap == bq.front {
		//1.update map
		height := bq.oldProposal[bq.front].Height
		index := bq.height2Index[height]
		delete(bq.index2Height, index)
		delete(bq.height2Index, height)
		//2.remove
		bq.oldProposal[bq.front] = nil
		bq.front = (bq.front + 1) % bq.cap
	}

	bq.oldProposal[bq.rear] = proposal
	//update map
	bq.height2Index[proposal.Height] = bq.rear
	bq.index2Height[bq.rear] = proposal.Height

	bq.rear = (bq.rear + 1) % bq.cap

	return nil
}

//may not useful
func (bq *BaseQueue) Dequeue() (proposal *types.Proposal, err error) {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()
	if bq.cap == 0 {
		tendermintlog.Error("Dequeue:Queue is not init or already cleared")
		return nil, errors.New("Queue is not init or already cleared")
	}
	if bq.front == bq.rear {
		tendermintlog.Error("Dequeue:Queue is Queue is empty")
		return nil, errors.New("Queue is empty")
	}

	proposal = bq.oldProposal[bq.front]
	//update map
	height := proposal.Height
	index := bq.height2Index[height]
	delete(bq.index2Height, index)
	delete(bq.height2Index, height)

	bq.oldProposal[bq.front] = nil
	bq.front = (bq.front + 1) % bq.cap
	return proposal, nil
}

func (bq *BaseQueue) QueryElem(height int64) (proposal *types.Proposal, err error) {
	bq.mutex.Lock()
	defer bq.mutex.Unlock()
	if bq.cap == 0 {
		tendermintlog.Error("QueryElem:Queue is not init or already cleared")
		return nil, errors.New("Queue is not init or already cleared")
	}
	if index, ok := bq.height2Index[height]; ok {
		return bq.oldProposal[index], nil
	}
	//tendermintlog.Debug("QueryElem: not found proposal of height", "height", height)
	return nil, errors.New("not found proposal")
}
