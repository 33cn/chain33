package mempool

import "code.aliyun.com/chain33/chain33/types"

// 定义优先级队列存储对象
type Item struct {
	value    *types.Transaction
	priority int64
	index    int
}

// 优先级队列需要实现heap的interface
type PriorityQueue []*Item

// 绑定Len方法
func (pq PriorityQueue) Len() int {
	return len(pq)
}

// 绑定Less方法
func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

// 绑定swap方法
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index, pq[j].index = i, j
}

// 绑定pop方法，将index置为-1是为了标识该数据已经出了优先级队列了
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	item.index = -1
	return item
}

// 绑定push方法
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

// 删除在优先级队列中特定位置对象
func (pq *PriorityQueue) Remove(index int) interface{} {
	n := pq.Len() - 1
	if n != index {
		pq.Swap(index, n)
	}
	p := pq.Pop()
	// heap.Fix(pq, index)
	return p
}

// 返回第一个元素
func (pq *PriorityQueue) Top() interface{} {
	p := *pq
	return p[0]
}

// 查找指定位置的元素
func (pq *PriorityQueue) Find(i int) interface{} {
	p := *pq
	return p[i]
}
