// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

/*
执行器就是状态机，根据交易类型，会有对应的执行器去执行。
执行器由一些原子的命令组成。

在执行交易的时候，会在一个虚拟的执行环境下，模拟执行。
错误的交易允许被执行，只要他的手续费是够的。
手续费不够的交易直接抛弃。

执行的过程中，会产生一些Event，Event作为交易的Receipt

//input
ReplyTxList

//output
Receipts->(IsOk, EventLogs, KVSet)

var kvs types.KV
var receipts types.Receipts
for tx := range txs {
	storeSet, events, ok := execTx(tx) //对于错误的交易 events 就是：InvalidTx
	//执行的过程中，会更新内存模拟数据库，这里主要是余额信息
}

执行器的设计原则：

1. 不改变存储状态
2. 无状态设计

TODO:

执行器可以进一步分布式化：

1. 预分析，编译出执行指令
2. 分析指令的独立性。大部分可能非关联。
*/
