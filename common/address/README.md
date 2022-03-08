# address
>实现公钥到地址的转换, 支持实现自定义地址格式, 地址驱动插件化

## addressID
> 地址驱动类型值
- 每个地址驱动有唯一的类型及名称值
- 类型值范围 [0, 8)
- 底层相关设计, 参考chain33/types/sign.md

## 配置
>地址驱动配置说明

### DefaultDriver
>配置默认的地址驱动, 默认地址格式作为执行器以及未指定addressID时地址格式转换
```toml
[address] #示例
DefaultDriver="eth"
```

### EnableHeight
> 分叉高度配置, 即区块高度达到配置高度后启用插件, 不配置采用内置的启用高度, 负数表示不启用
```toml
[address] #示例
[address.enableHeight]
"btc" = -1
"eth"= 100
``` 



