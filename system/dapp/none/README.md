## 存证合约


### 交易类型

#### 普通存证交易
普通存证交易只在链上存储记录，不做业务逻辑执行

#### 延时存证交易
将延时交易提交到链上，即交易的payload包含一个需要延时的交易


##### 交易请求

```proto
message CommitDelayTx {

    string delayTx             = 1; //延时交易, 16进制格式
    int64  relativeDelayHeight = 2; //相对延时时长，相对区块高度
}
```

[Transaction结构定义](../../../types/proto/transaction.proto#L88)


##### 交易回执

```proto
message CommitDelayTxLog {
    string submitter        = 1; // 提交者
    string delayTxHash      = 2; // 延时交易哈希
    int64  delayBeginHeight = 3; // 延时开始区块高度
}
```


##### 交易构造接口及参数

- 创建交易通用json rpc接口，Chain33.CreateTransaction
- execer: "none"
- actionName: "CommitDelayTx"
- payload: [CommitDelayTx](README.md#交易请求)

