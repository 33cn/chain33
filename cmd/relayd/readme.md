# relayd 方案实现

## 实现

* 作为一个独立的进程，运行信息可以配置，守护进程(supervisor)，跨平台，高性能。
* 连接多个比特币节点数，进行数据筛选，防止节点的攻击。
* relayd 连接chain33通过GRPC实现，连接btcd可以通过websockets，或者HTTPS，两者都要实现。
* 本地监听通过GRPC或者HTTP来实现，监听外部消息。
* 定时获取btc链的同步信息，以及定时获取chain33的订单信息，以及触发超时信息工作。
* 心跳检查，连接chain33与btc两个TCP连接。

## 关注那些信息

* 各个区块头部，可以到blockinfo.org网站获取。
* SPV验证需要的信息。
* 指定交易的相关信息。

## SPV
0. 从网络上获取并保存最长链的所有block header至本地；
1. 计算该交易的hash值tx_hash；
2. 定位到包含该tx_hash所在的区块，验证block header是否包含在已知的最长链中；
3. 从区块中获取构建merkle tree所需的hash值；
4. 根据这些hash值计算merkle_root_hash；
5. 若计算结果与block header中的merkle_root_hash相等，则交易真实存在。
6. 根据该block header所处的位置，确定该交易已经得到多少个确认。

## 验证交易

* 根据指定的交易hash，获取指定的信息
* 根据hash是否能找到交易。
* 交易双方的address（地址），必须保证被转账人的地址正确。
* 金额是否正确。
* 交易上链确认时间timestamp(时间戳)，必须在 10m + hash.time <= hash.time <= hash.time + 2h 范围内。
* 确认交易是否被上链。即SPV的验证。

## relay executor合约执行交易

* 确认交易，需要外界的触发
* relay合约执行交易需要被外界出发，意味着订单的状态：pending、locking、confirming、canceled、timeout、finished等状态。
* 订单状态:
    * pending: 挂单人发起挂单。
    * locking： 接单人接收单，锁住该订单，不允许被其他占用。
    * confirming: 接单人已经转账，正在确认转成是否成功的订单状态。
    * finished: 订单完成，交易已经得到确认。
    * canceled: 挂担人取消订单，只能处于timeout、pending状态的单子才能被取消。
    * timeout: pending状态的单一直无人接受，时间超时；lock状态的单一直没有人去撮合，时间超时； conforming状态的单，一直无法确定转账成功。
* unlock：准确说是一个动作，去解锁正在locking的订单，不能把它划分到订单的状态中，解锁完成立，订单马进入到pending状态。
* 状态装换图:
```


                      +--------------|---------------|
                    pending ---+ locking ---+ confirming ---+ finished
                    |                |               |
                    |                |               |
                    |----------------|---------------|------+ cancel
                    |                |               |
                    |                +               |
                     -----------+ timeout +----------


分析：
初始状态有一种：pending
中间状态有两种：locking、confirming
最终状态有三种：finished、cancel、timeout.

挂单方有unlock和撤单两种操作，unlock是在其他状态基础上取消到pending，撤单是撤销到canceled状态
接单方只有unlock操作, 是在其他状态基础上取消到pending

locking状态unlock需要等待12个小时或者12*6个BTC block  因为BTC交易等待的时间很长
confirming 状态，撤销需要等待4*12小时或者4*12*6个BTC block

finish状态要等待BTC n个块之后，n缺省为6，可以由买BTC的一方设定

注意：为了安全考虑，目前只允许特殊的账户发送BTC Header信息，目前是testNet的genesis addr,正式使用时候要修改

结论：
状态的转换需要外部事件的触发。
```

## 预言机（触发机）

* 发送交易，触发链执行交易，执行合约。
* 订单的状态相互之间转换，这种状态转换需要等待外界的触发程序去触发链去执行，怎么实现触发程序？，是否需要relayd去充当？

## 答疑

* 会有人问，随便发送个hash，怎么确定交易？ 答：不正确的hash，relayd得不到正确的查询，返回错误，取消chain33订单的unlock状态。
* 如果转账不正确，怎确认？ 答：根据relayd获取的金额进行确认，查看是否正确。
* 双花问题：即在chain33这条链上怎么确认?同样是交易，chain33会检查，另外executor(relay)执行时，会检查hash是否被记录过，即：hash锁定。即只有转账人才能确定，锁定转账人。
* 双花问题：认证交易是否属于这个转账人，在不知道用户私钥的情况下。仅仅就是一个交易hash怎么判断？两次交易，怎么挂钩？挂钩任意一个吗？
* 双花问题：hash代表的价值是否被双向挂钩无法确定，即，解决双花问题，chain33的的订单关联的锁定btc的地址，需要被锁定，意味着，btc地址占时锁定在chain33的订单上。
    锁定时间从订单的开始到结束。
