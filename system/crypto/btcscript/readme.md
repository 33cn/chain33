
## 比特币脚本集成

### 集成方案

比特币脚本集成主要涉及两个绑定

#### 交易绑定

由于比特币脚本内部仍然是基于比特币交易，锁定脚本和解锁脚本对应的是比特币交易，
无法约束到chain33交易，需要完成交易绑定，相关步骤如下：

1. 根据锁定脚本生成解锁脚本时，需要构造临时的比特币交易btctx
2. btctx只包含一个输入UTXO，且该UTXO的索引哈希为chain33交易的哈希
3. 这样基于btctx生成的解锁脚本就和chain33交易相互绑定，保证了chain33交易不被篡改

#### 交易发送方绑定

锁定脚本和解锁脚本智能约束交易是否正确，
在chain33交易中，需要在签名结构中指定公钥，即对应交易发送方的公钥，
需要通过脚本限定交易发送方的地址

1. 在比特币脚本签名验证中，签名的数据需要包含锁定脚本和解锁脚本
2. chain33交易签名公钥设为Hash256(锁定脚本)，即交易的发送方地址由锁定脚本生成
3. 类似于比特币P2SH，采用比特币脚本验证方式，交易的发起地址都是基于锁定脚本哈希计算
4. 在验证时，即需要验证公钥是否和锁定脚本一一对应，即绑定了chain33交易的发送方地址



### 相关脚本
介绍相关脚本实现方案

#### 多重签名

多重签名脚本可以通过接口调用直接构建，M <Public Key 1> <Public Key 2> … <Public Key N> N CHECKMULTISIGVERIFY

设有地址A，B，C，在chain33中完成3:2多重签名大概步骤如下

1. 构造锁定脚本S1， 2 PubkeyA PubkeyB PubkeyC 3 CHECKMULTISIG
2. 生成chain33多重签名地址X = Pubkey2Addr(Hash256(S1))
3. 在chain33中，在地址X中存入需要多重签名的资产
4. 提取X资产需提供解锁脚本，即对应的多重签名，任意ABC中的两个私钥的数据签名


#### 钱包找回

1. 构造延时存证交易，即payload为实际需要延时的交易，链上执行后需要记录延时交易的开始时刻

2. 基于比特元CSV（check sequence verify）操作码， 构造锁定脚本，即限定utxo在相对时长后能花费

3. CSV在比特币中需要基于一个开始时间点，在集成到chain33中，使用1中构造的none延时交易执行时间点作为CSV的开始时间，并进行相对时长验证

4. 钱包找回原始地址A，备份地址B，即地址A私钥丢失时，可以使用备份地址B的私钥控制资产，但需要延时验证

5. 构建锁定脚本S， IF <A's Pubkey> CHECKSIGVERIFY ELSE <1 day> CHECKSEQUENCEVERIFY DROP <B's Pubkey> CHECKSIGVERIFY ENDIF

> 基于锁定脚本S生成脚本地址X，将需要找回控制的资产转入X, A的私钥可以控制X的资产，B的私钥转移X的资产时，需要满足延时时长一天

#### 延时转账

延时转账可支持，绝对时刻和相对时长两种

##### 绝对时刻

1. 基于CLTV（check lock time verify）操作

2. 地址A和B，A需要向B进行延时一小时转账，A直接根据当前区块时间或高度计算出延时截止时刻T

3. 构建锁定脚本S， IF <A's Pubkey> CHECKSIGVERIFY ELSE <T> CHECKLOCKTIMEVERIFY DROP <B's Pubkey> CHECKSIGVERIFY ENDIF

> 根据S生成中间地址X，A首先向X进行相应资产转账, A的私钥可以一直控制X的资产，而B的私钥当且仅当在时刻T后能控制X的资产

##### 相对时长
基于none延时交易，类似于钱包找回操作

1. 构造none合约延时交易，即payload为实际需要延时的交易，链上执行后需要记录延时交易的开始时刻

2. 基于比特元CSV（check sequence verify）操作码， 构造锁定脚本，即限定utxo在相对时长后能花费

3. CSV在比特币中需要基于一个开始时间点，在集成到chain33中，使用1中构造的none延时交易执行时间点作为CSV的开始时间，并进行相对时长验证

4. 延时转账发起方A，接收方B，A向B进行延时转账

5. 构建锁定脚本S， IF <A's Pubkey> CHECKSIGVERIFY ELSE <1 hour> CHECKSEQUENCEVERIFY DROP <B's Pubkey> CHECKSIGVERIFY ENDIF

> 基于锁定脚本S生成脚本地址X，将需要延时转账的资产转入X, A的私钥可以控制X的资产，B的私钥转移X的资产时，需要满足延时时长一个小时
