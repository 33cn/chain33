## 脚本使用


### 钱包资产找回

#### 相关概念

> 脚本地址X

该地址由钱包找回脚本哈希生成地址, 由于私钥未知, 地址上的资产需要使用钱包找回脚本对应的约束进行提取

> 控制地址A

该地址为钱包找回主地址, 可以基于比特币脚本构造签名, 实时提取地址X中的资产

> 找回地址B

该地址为钱包找回副本地址, 可以基于比特币脚本构造签名, 延时提取地址X中的资产, 目前仅支持2个找回地址

> 钱包找回脚本S

一种比特币脚本, 由地址A和B对应公钥, 以及延时等数据构成

> 基本场景
- 用户生成地址A和B及X, 地址X用于线上资产存储, 地址A离线控制X资产存取
- 地址A私钥丢失时, 可使用地址B的私钥进行延时找回地址X资产
- 实际使用中, 地址B也可由第三方托管控制


#### 操作步骤

> 获取钱包找回脚本地址X
- 已知钱包找回控制地址A，找回地址B，相对延时时长T
- 调用接口获取获取钱包找回地址X，[相关rpc](README.md#chain33getwalletrecoveraddress)
- 用户使用链上转账功能，将需要被钱包找回控制的的资产转入到地址X
- X对应的私钥未知，需要由地址A或B的私钥构造比特币脚本签名，进行资产操作

> 实时提取X资产
- 控制地址A可以随时对X的资产进行提取，即构造发送方为X的原始转账交易
- 交易需要采用比特币脚本类型签名， [相关rpc](README.md#chain33signwalletrecovertx)
- 发送交易到链上执行

> 延时提取X资产
- 找回地址B可以基于延时交易提取X的资产, 即构造发送方为X的原始转账交易tx1, 但tx1需要提交到链上并等待延时打包
- 交易tx1需要采用比特币脚本类型签名, [相关rpc](README.md#chain33signwalletrecovertx)
- 基于tx1, 相对延时T, 构造延时存证交易tx2，[构造方法](../../../dapp/none/README.md#延时存证交易)
- 将tx2签名并发送至链上执行，tx1将作为tx2的交易内容, 暂存于链上等待延时打包
- 通过tx1哈希, 查询tx1是否被打包执行


#### rpc接口

##### chain33.GetWalletRecoverAddress

> 请求结构 ReqGetWalletRecoverAddr

|字段名称 |类型|含义
|---|---|---|
|ctrPubKey|string|控制地址公钥, secp256k1算法, 16进制
|recoverPubKey|[]string|找回地址公钥数组, secp256k1算法, 16进制, 目前只支持两个地址,即数组长度为2
|relativeDelayTime|int64|钱包找回相对延时时长, 单位秒




> 响应
 
|名称 |类型|含义
|---|---|---|
|result|string|脚本地址X



##### chain33.SignWalletRecoverTx

> 请求结构 ReqSignWalletRecoverTx

|字段名称 |类型|含义
|---|---|---|
|walletRecoverParam|ReqGetWalletRecoverAddr|钱包找回信息结构
|signAddr|string|签名地址 
|privKey|string|签名地址的私钥, secp256k1算法, 16进制, 不指定私钥时,将从本地钱包获取对应私钥
|rawTx|string| 原始交易, 16进制


> 响应 

|名称 |类型|含义
|---|---|---|
|result|string|签名后的交易, hex格式
