
# 交易签名

## 签名类型
>即签名结构中Ty字段, 以下统称为signID

```go
type Signature struct {
      
      	Ty     int32  `protobuf:"varint,1,opt,name=ty,proto3" json:"ty,omitempty"`
      	Pubkey []byte `protobuf:"bytes,2,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
      	Signature []byte `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
 }      
``` 

#### 底层设计
> 最初设计中, signID即cryptoID, 在引入多地址格式兼容(#1181)后, signID集成了cryptoID和addressID含义

* addressID, 即签名公钥转换为地址时采用的地址类型ID, 在signID中占3位, 即低位第13~15位
* cryptoID, 即签名对应的加密算法插件类型ID, 在signID中, 除addressID以外的其余位
* 默认addressID为0, 此时cryptoID和signID值依然相等

#### 集成算法
>　通过位运算集成(兼容最初的signID逻辑, cryptoID只能作为低位) 
* signID = (addressID << 12) | cryptoID
* addressID = 0x00007000 & signID >> 12
* cryptoID = signID & 0xffff8fff


#### 相关接口
> signID, cryptoID, addressID转换接口

* EncodeSignID, 基于cryptoID和addressID, 计算signID
* ExtractAddressID, 基于signID, 提取addressID(主要用于交易的fromAddr计算)
* ExtractCryptoID, 基于signID, 提取cryptoID


#### 相关文档
- 加密模块介绍, chain33/common/crypto/README.md
- 地址模块介绍, chain33/common/address/README.md
