# 流程（不考虑P2P）

## rpc

用户的交易首先进入rpc模块，rpc模块通过 EventTx 事件发送交易到 mempool 模块.

## mempool

mempool 模块收到EventTx 消息以后，保存到内存.

mempool 收到TxAddBlock 消息，会把block 中有的Tx 从mempool中删除

mempool 收到EventTxList 消息，把交易发送给 consense，并把交易从内存中删除

## consense

consense 模块从 mempool（EventTxList） 取出交易列表

consense 生成交易hash列表发送给EventTxHashList 给blockchain.

consense 生成区块，发送区块给 (EventAddBlock) blockchain 模块.

consense 收到（EventAddBlock）区块信息，更新共识状态

## blockchain 模块

blockchain 模块保存区块到硬盘，并更新高度后，
会把发送 （EventAddBlock） 事件给 mempool 和 consense

blockchain 收到 EventTxHashList ，找出已经打包到区块的交易Hash发送给 (EventTxHashListReply)  