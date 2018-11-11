package init

//系统级别的签名定义
//为了安全考虑，默认情况下，我们希望只定义合约内部的签名，系统级别的签名对所有的合约都有效
import (
	_ "github.com/33cn/chain33/system/crypto/ed25519"
	_ "github.com/33cn/chain33/system/crypto/secp256k1"
	_ "github.com/33cn/chain33/system/crypto/sm2"
)
