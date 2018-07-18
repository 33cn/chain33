# paracross 执行器

## 执行器依赖

 1. 配置多个节点参与共识
    * configPrefix-title-node-list, 列表是对应节点签名用的公钥地址

## 状态
 1. prefix-title: height, 记录已经达成共识的高度
 1. prefix-title-height: 记录对应高度的状态

## 执行逻辑

只有一种action，提交对应高度的平行链状态

检查
 1. 节点是否是平行链的有效节点： 配置是否存在，在不在配置里
 1. 高度是否已经过了共识
 1. 数据有效性检测

执行
 1. 记录状态到 prefix-title-height, 记录过的节点覆盖上次记录(分叉情况)
 1. 是否触发对应title-height达成共识
 1. 如果没有达成共识， 执行结束
 1. 达成共识， 记录 prefix-title
 1. kv-log-1: 状态变化
 1. kv-log-2: 达成共识 (看是否达成共识)

达成共识条件
 1. 对应title-height的同一个状态的数据超过配置节点的 2/3


## 本地数据添加
 1. 记录交易信息 prifex-title-height-addr
 1. 其他数据看查询需要

## 本地数据删除
 1. 记录交易信息 prifex-title-height-addr
