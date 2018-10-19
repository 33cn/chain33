/*
Package paracross 实现了跨链合约.

合约功能如下
 1. 跨链共识
 1. 平行链挖矿
 1. 跨链转账
 1. 转账/合约到转账

跨链共识
 1. 需要先在主链配置平行链挖矿节点，才能开始工作
 1. 主链节点需要配置 isRecordBlockSequence=true， 另行链节点才能有效同步
 1. 另行链挖矿节点需要超过2/3 达成一致， 才能完成共识

平行链挖矿
 1. 构造本平行链交易， 记录当前区块的信息
 1. 空区块看上也是有交易的

帐号模型遵循现在的命名格式
 1. 用户或合约帐号: malv-{exec}-{symbol}-{addr()}
 1. 合约子帐号:     malv-{exec}-{symbol}-{addr(exec)}:{addr(user)}

帐号模型举例: (参数国盾平行链: user.p.guodun ， token的symbol: TEST)
 1. mavl-token-symbol{TEST}-addr{userA}
    * 用户A 的 token TEST 帐号
 1. mavl-token-symbol{TEST}-addr{paracross}:addr{userA}
    * 用户A 在 paracross 合约中 token TEST 子帐号
 1. mavl-token-symbol{TEST}-addr{paracross}:addr{user.p.guodun.paracross}
    * 国盾平行链 在 paracross 合约中 token TEST 子帐号
 1. mavl-paracross-symbol{token.TEST}-addr{userA}
    * 用户A 在另行链中 paracross 中的 主链资产token TEST 子帐号
 1. malv-paracross-symbol{token.TEST}-addr{trade}:addr{userA}
    * 用户A 在另行链中 trade 合约中的 主链资产token TEST 子帐号
 1. malv-paracross-symbol{token.TEST}-addr{trade}
    * trade合约中的 主链资产token TEST 子帐号

资产符号用产币合约加币的符号来表示， 如 token.TEST
 1. 但主链 trade 合约已经有token 数据,  其表示没有前缀为 TEST

转账/合约到转账
 1. 目前开启资产从主链到平行链的转移， 故这组操作只在平行链生效
 1. 资产就像原生在paracross 执行器中(想象token合约里的不同的token)
 1. 可在平行链上转账给他人， 也可以转账到其他合约进行使用
*/
package paracross
