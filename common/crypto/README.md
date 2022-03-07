# crypto

## 功能
* 支持 ed25519, secp256k1, sm2
* 统一的 PrivKey，Pubkey, Signature 接口, 详见 `crypto.go`

## 依赖
* sm2 编译依赖 gmssl 2.0 版本
* 安装：`bash ./deps/install_gmssl.sh`



## 配置
>支持加密插件使能开启, 分叉高度等配置

### EnableTypes
>指定开启若干加密插件(插件名称)，不配置默认启用所有
```toml
[crypto] #示例
enableTypes=["secp256k1", "sm2"]
```

### EnableHeight
> 分叉高度配置, 即区块高度达到配置高度后启用插件, 不配置采用内置的启用高度, 负数表示不启用
```toml
[crypto] #示例
[crypto.enableHeight]
"secp256k1" = 0
"sm2"= 100
``` 

## cryptoID
加密插件ID值, int32类型

### 手动指定

>注册插件时, 支持手动指定插件ID(crypto.WithRegOptionTypeID(manualID))

* 手动指定ID范围, (0, 4096), 低位12位 
* 建议单元测试调用CryptoGetCryptoList接口查看已注册ID情况.


### 自动生成
> 注册时未指定手动ID, 将根据插件名称随机生成, 存在冲突时需要手动指定解决


### 相关说明
> cryptoID主要用于交易签名算法类型判定, cryptoID不等同于交易SignatureID

* 接口chain33/types/ExtractCryptoID, 基于signID解析cryptoID
* 底层设计, 参考chain33/types/sign.md
