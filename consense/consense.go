package consense

import "code.aliyun.com/chain33/chain33/queue"

type Consense struct{}

func New(name string) *Consense {
	return &Consense{}
}

func (con *Consense) SetQueue(q *queue.Queue) {

}
