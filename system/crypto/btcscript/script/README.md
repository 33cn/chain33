## 脚本使用


### 钱包资产找回

#### 操作步骤

> 获取钱包找回地址X
- 已知钱包找回控制地址A，找回地址B，相对延时时长T
- 调用接口获取获取钱包找回地址X，[相关rpc](README.md#chain33getwalletrecoveraddress)
- 用户使用链上转账功能，将需要被钱包找回控制的的资产转入到地址X
- X对应的私钥未知，需要由地址A或B的私钥构造比特币脚本签名，进行资产操作

> 控制地址提取X资产
- 控制地址A可以随时对X的资产进行提取，即构造发送方为X的原始转账交易
- 交易需要采用比特币脚本类型签名， [相关rpc](README.md#chain33signwalletrecovertx)
- 发送交易到链上执行

> 找回地址提取X资产
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
|recoverPubKey|string|找回地址公钥, secp256k1算法, 16进制
|relativeDelayHeight|int64|钱包找回相对延时时长, 区块高度值




> 响应
 
|名称 |类型|含义
|---|---|---|
|result|string|钱包找回地址



##### chain33.SignWalletRecoverTx

> 请求结构 ReqSignWalletRecoverTx

|字段名称 |类型|含义
|---|---|---|
|walletRecoverParam|ReqGetWalletRecoverAddr|钱包找回信息结构
|privKey|string|签名地址私钥, secp256k1算法, 16进制
|rawTx|string| 原始交易, 16进制


> 响应 

|名称 |类型|含义
|---|---|---|
|result|string|签名后的交易, hex格式
